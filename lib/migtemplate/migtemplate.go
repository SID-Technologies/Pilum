// Package migtemplate creates the instance template for a MIG release: a copy
// of the template the MIG runs now, with only the image metadata changed.
//
// The template runs its container the way Google recommends since the
// container startup agent was retired: a startup script on Container-Optimized
// OS reads the image from instance metadata (container-image by default) and
// runs `docker run`. Pilum only ever changes that one metadata value.
//
// gcloud can't do this on its own. `instance-templates create` has no flag to
// copy another template, and rebuilding one from flags would drop whatever
// the setup script configured that Pilum doesn't know about (service account,
// scopes, network tags, shielded VM, labels...). So the current template is
// read with gcloud, edited as JSON, and inserted through the Compute API.
package migtemplate

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/sid-technologies/pilum/ingredients/gcp"
	"github.com/sid-technologies/pilum/lib/errors"
)

const computeAPI = "https://compute.googleapis.com/compute/v1"

// Runner executes a read-only gcloud command and returns its stdout.
type Runner func(cmd []string) (string, error)

// Doer sends an HTTP request. *http.Client satisfies it.
type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Request identifies the MIG and the template to create for it.
type Request struct {
	Project        string
	MIG            string
	Zone           string
	Region         string
	Name           string // New template name, <base>-<tag>
	Image          string // Full image reference to run
	ImageKey       string // Metadata key holding the image
	TemplateRegion string // Empty for a global template

	Overrides Overrides
}

// Overrides are template settings the service manages. A zero value field
// keeps what the current template has.
type Overrides struct {
	MachineType    string
	Network        string // Name or resource path
	Subnet         string // Name or resource path
	SubnetRegion   string // Region a bare subnet name resolves in
	ExternalIP     string // "", "true" or "false"
	NetworkTags    []string
	SetNetworkTags bool
	ServiceAccount string

	Env        []string // KEY=VALUE
	SetEnv     bool     // Replace container-env (otherwise kept as is)
	Secrets    []string // NAME=projects/.../secrets/.../versions/...
	SetSecrets bool     // Replace container-secrets (otherwise kept as is)
}

func (r Request) location() []string {
	if r.Zone != "" {
		return []string{"--zone", r.Zone}
	}
	return []string{"--region", r.Region}
}

// Result says what Create did.
type Result struct {
	Source  string // Template the MIG was running
	Created bool   // False when the template already existed with this image
}

// Create makes the release template unless an identical one already exists,
// so a retried deploy picks up where it left off.
func Create(req Request, run Runner, client Doer) (Result, error) {
	if req.Project == "" || req.MIG == "" || req.Name == "" || req.Image == "" {
		return Result{}, errors.New("project, mig, name and image are required")
	}
	if req.Zone == "" && req.Region == "" {
		return Result{}, errors.New("set mig.zone (zonal MIG) or mig.region (regional MIG)")
	}
	if req.ImageKey == "" {
		req.ImageKey = gcp.DefaultImageMetadataKey
	}
	err := validateOverrides(req.Overrides)
	if err != nil {
		return Result{}, err
	}

	describeMIG := append([]string{"gcloud", "compute", "instance-groups", "managed", "describe", req.MIG,
		"--format", "json", "--project", req.Project}, req.location()...)
	migJSON, err := run(describeMIG)
	if err != nil {
		return Result{}, errors.Wrap(err, "describing MIG %s", req.MIG)
	}
	source, err := CurrentTemplate(migJSON)
	if err != nil {
		return Result{}, errors.Wrap(err, "instance group %s", req.MIG)
	}
	result := Result{Source: TemplateName(source)}

	region := TemplateRegion(source)
	if region != req.TemplateRegion {
		return Result{}, errors.New("instance group %s runs %s (%s), but new templates would be %s; set mig.template_region to match",
			req.MIG, result.Source, scope(region), scope(req.TemplateRegion))
	}

	// A rerun after the rollout started (say the wait timed out): nothing to
	// create, and the rolling update that follows is a no-op.
	if result.Source == req.Name {
		return result, nil
	}

	sourceJSON, err := run(gcp.GenerateDescribeTemplateCommand(req.Project, source))
	if err != nil {
		return Result{}, errors.Wrap(err, "describing template %s", result.Source)
	}
	body, err := InsertBody(sourceJSON, req)
	if err != nil {
		return Result{}, errors.Wrap(err, "template %s", result.Source)
	}

	exists, err := checkExisting(req, run)
	if err != nil || exists {
		return result, err
	}

	token, err := run([]string{"gcloud", "auth", "print-access-token"})
	if err != nil {
		return Result{}, errors.Wrap(err, "getting an access token")
	}
	err = insert(client, strings.TrimSpace(token), InsertURL(req.Project, req.TemplateRegion), body)
	if err != nil {
		return Result{}, errors.Wrap(err, "creating template %s", req.Name)
	}
	result.Created = true
	return result, nil
}

// checkExisting reports whether the template exists already. Existing with the
// same image is a retry and fine; with another image it is a reused tag.
func checkExisting(req Request, run Runner) (bool, error) {
	listed, err := run([]string{"gcloud", "compute", "instance-templates", "list",
		"--filter", "name=" + req.Name, "--format", "json", "--project", req.Project})
	if err != nil {
		return false, errors.Wrap(err, "checking for template %s", req.Name)
	}
	var templates []templateJSON
	err = json.Unmarshal([]byte(listed), &templates)
	if err != nil {
		return false, errors.Wrap(err, "invalid template list JSON")
	}
	for _, t := range templates {
		if t.Name != req.Name || TemplateRegion(t.SelfLink) != req.TemplateRegion {
			continue
		}
		image := t.Metadata(req.ImageKey)
		if image != req.Image {
			return false, errors.New("template %s already exists and runs %s, not %s; use a new --tag", req.Name, image, req.Image)
		}
		return true, nil
	}
	return false, nil
}

// templateJSON is the subset of an instance template Pilum reads.
type templateJSON struct {
	Name              string    `json:"name"`
	SelfLink          string    `json:"selfLink"`
	CreationTimestamp time.Time `json:"creationTimestamp"`
	Properties        struct {
		Metadata struct {
			Items []struct {
				Key   string `json:"key"`
				Value string `json:"value"`
			} `json:"items"`
		} `json:"metadata"`
	} `json:"properties"`
}

// Metadata returns the template's metadata value for key, "" if unset.
func (t templateJSON) Metadata(key string) string {
	for _, item := range t.Properties.Metadata.Items {
		if item.Key == key {
			return item.Value
		}
	}
	return ""
}

// CurrentTemplate returns the self link of the template a MIG runs. A MIG
// mid-canary has two versions and no single current template, so that is an
// error rather than a guess.
func CurrentTemplate(migJSON string) (string, error) {
	var mig struct {
		InstanceTemplate string `json:"instanceTemplate"`
		Versions         []struct {
			InstanceTemplate string `json:"instanceTemplate"`
		} `json:"versions"`
	}
	err := json.Unmarshal([]byte(migJSON), &mig)
	if err != nil {
		return "", errors.Wrap(err, "invalid MIG JSON")
	}
	switch len(mig.Versions) {
	case 0:
		if mig.InstanceTemplate == "" {
			return "", errors.New("instance group has no instance template")
		}
		return mig.InstanceTemplate, nil
	case 1:
		return mig.Versions[0].InstanceTemplate, nil
	default:
		return "", errors.New("instance group has %d versions (canary in progress?); finish or cancel it first", len(mig.Versions))
	}
}

var regionalLink = regexp.MustCompile(`/regions/([^/]+)/instanceTemplates/`)

// TemplateRegion returns the region of a regional template link, "" if global.
func TemplateRegion(link string) string {
	m := regionalLink.FindStringSubmatch(link)
	if m != nil {
		return m[1]
	}
	return ""
}

// TemplateName returns the last path segment of a template link.
func TemplateName(link string) string {
	return link[strings.LastIndex(link, "/")+1:]
}

// InsertURL is the Compute API collection a new template is inserted into.
func InsertURL(project, region string) string {
	if region == "" {
		return computeAPI + "/projects/" + project + "/global/instanceTemplates"
	}
	return computeAPI + "/projects/" + project + "/regions/" + region + "/instanceTemplates"
}

// InsertBody turns a described template into an insert request for a copy
// named req.Name running req.Image. Everything under properties is carried
// over untouched apart from the image metadata and whatever req.Overrides set.
func InsertBody(sourceJSON string, req Request) ([]byte, error) {
	var source map[string]any
	err := json.Unmarshal([]byte(sourceJSON), &source)
	if err != nil {
		return nil, errors.Wrap(err, "invalid template JSON")
	}
	props, ok := source["properties"].(map[string]any)
	if !ok {
		return nil, errors.New("template has no properties")
	}

	err = setImage(props, req.ImageKey, req.Image)
	if err != nil {
		return nil, err
	}
	err = applyOverrides(props, req.Project, req.Overrides)
	if err != nil {
		return nil, err
	}
	stripOutputOnly(props)

	body := map[string]any{"name": req.Name, "properties": props}
	desc, ok := source["description"].(string)
	if ok && desc != "" {
		body["description"] = desc
	}
	return json.Marshal(body)
}

// setImage points the template's image metadata at image. The key must
// already exist: it proves the startup script is one that reads it.
func setImage(props map[string]any, imageKey, image string) error {
	metadata, ok := props["metadata"].(map[string]any)
	if !ok {
		return missingImageKey(imageKey)
	}
	items, ok := metadata["items"].([]any)
	if !ok {
		return missingImageKey(imageKey)
	}

	replaced := false
	for _, raw := range items {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		switch item["key"] {
		case gcp.LegacyContainerDeclarationKey:
			return errors.New("template uses %s, the retired container startup agent; GCP no longer creates VMs from it "+
				"(since July 31, 2026). Run the container from a startup script that reads the %s metadata instead",
				gcp.LegacyContainerDeclarationKey, imageKey)
		case imageKey:
			item["value"] = image
			replaced = true
		default:
			// Everything else, the startup script included, is copied as is.
		}
	}
	if !replaced {
		return missingImageKey(imageKey)
	}
	return nil
}

// stripOutputOnly drops fingerprints, which describe the source template's
// state and aren't part of an insert.
func stripOutputOnly(props map[string]any) {
	metadata, ok := props["metadata"].(map[string]any)
	if ok {
		delete(metadata, "fingerprint")
	}
	nics, ok := props["networkInterfaces"].([]any)
	if !ok {
		return
	}
	for _, raw := range nics {
		nic, ok := raw.(map[string]any)
		if ok {
			delete(nic, "fingerprint")
		}
	}
}

func applyOverrides(props map[string]any, project string, o Overrides) error {
	if o.MachineType != "" {
		props["machineType"] = o.MachineType
	}
	if o.ServiceAccount != "" {
		props["serviceAccounts"] = []any{map[string]any{
			"email":  o.ServiceAccount,
			"scopes": serviceAccountScopes(props),
		}}
	}
	if o.SetNetworkTags {
		props["tags"] = map[string]any{"items": stringsToAny(o.NetworkTags)}
	}
	if o.SetEnv {
		setMetadata(props, gcp.EnvMetadataKey, o.Env)
	}
	if o.SetSecrets {
		setMetadata(props, gcp.SecretsMetadataKey, o.Secrets)
	}

	if o.Network == "" && o.Subnet == "" && o.ExternalIP == "" {
		return nil
	}
	nic, err := primaryInterface(props)
	if err != nil {
		return err
	}
	if o.Network != "" {
		nic["network"] = resourcePath(o.Network, "projects/"+project+"/global/networks/")
	}
	if o.Subnet != "" {
		nic["subnetwork"] = resourcePath(o.Subnet, "projects/"+project+"/regions/"+o.SubnetRegion+"/subnetworks/")
		if o.Network == "" {
			// The subnet determines the network. Keeping the old one would
			// conflict if the subnet is in a different VPC.
			delete(nic, "network")
		}
	}
	switch o.ExternalIP {
	case "false":
		delete(nic, "accessConfigs")
	case "true":
		configs, ok := nic["accessConfigs"].([]any)
		if !ok || len(configs) == 0 {
			nic["accessConfigs"] = []any{map[string]any{"type": "ONE_TO_ONE_NAT", "name": "External NAT"}}
		}
	default:
		// Unset: keep the template's.
	}
	return nil
}

// primaryInterface returns nic0, creating it if the template has none.
func primaryInterface(props map[string]any) (map[string]any, error) {
	nics, ok := props["networkInterfaces"].([]any)
	if !ok || len(nics) == 0 {
		nic := map[string]any{}
		props["networkInterfaces"] = []any{nic}
		return nic, nil
	}
	nic, ok := nics[0].(map[string]any)
	if !ok {
		return nil, errors.New("template has a malformed network interface")
	}
	return nic, nil
}

// serviceAccountScopes keeps the template's scopes when swapping the account,
// defaulting to cloud-platform (access is then governed by the account's IAM
// roles, as Google recommends).
func serviceAccountScopes(props map[string]any) []any {
	defaultScopes := []any{"https://www.googleapis.com/auth/cloud-platform"}
	accounts, ok := props["serviceAccounts"].([]any)
	if !ok || len(accounts) == 0 {
		return defaultScopes
	}
	account, ok := accounts[0].(map[string]any)
	if !ok {
		return defaultScopes
	}
	scopes, ok := account["scopes"].([]any)
	if !ok || len(scopes) == 0 {
		return defaultScopes
	}
	return scopes
}

// setMetadata replaces key with lines joined by newlines, removing it when
// lines is empty.
func setMetadata(props map[string]any, key string, lines []string) {
	metadata, ok := props["metadata"].(map[string]any)
	if !ok {
		metadata = map[string]any{}
		props["metadata"] = metadata
	}
	items, ok := metadata["items"].([]any)
	if !ok {
		items = nil
	}

	kept := make([]any, 0, len(items)+1)
	for _, raw := range items {
		item, ok := raw.(map[string]any)
		if ok && item["key"] == key {
			continue
		}
		kept = append(kept, raw)
	}
	if len(lines) > 0 {
		kept = append(kept, map[string]any{"key": key, "value": strings.Join(lines, "\n")})
	}
	metadata["items"] = kept
}

// resourcePath expands a bare name with prefix; paths and URLs pass through.
func resourcePath(value, prefix string) string {
	if strings.Contains(value, "/") {
		return value
	}
	return prefix + value
}

func stringsToAny(values []string) []any {
	out := make([]any, len(values))
	for i, v := range values {
		out[i] = v
	}
	return out
}

var envName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// validateOverrides rejects env and secret entries the startup script's
// env file can't carry: one KEY=VALUE per line, so no newlines in values.
func validateOverrides(o Overrides) error {
	for _, list := range [][]string{o.Env, o.Secrets} {
		for _, entry := range list {
			name, value, found := strings.Cut(entry, "=")
			if !found || !envName.MatchString(name) {
				return errors.New("invalid env entry %q: want NAME=VALUE with NAME made of letters, digits and _", entry)
			}
			if strings.ContainsAny(value, "\r\n") {
				return errors.New("env var %s contains a newline, which an env file can't hold", name)
			}
		}
	}
	switch o.ExternalIP {
	case "", "true", "false":
	default:
		return errors.New("external IP must be true or false, got %q", o.ExternalIP)
	}
	return nil
}

func missingImageKey(key string) error {
	return errors.New("template has no %s metadata; its startup script must read the image from it "+
		"(set mig.image_metadata_key if it uses another key)", key)
}

// operation is a Compute API long-running operation.
type operation struct {
	SelfLink string `json:"selfLink"`
	Status   string `json:"status"`
	Error    *struct {
		Errors []struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"errors"`
	} `json:"error"`
}

func (op operation) err() error {
	if op.Error == nil || len(op.Error.Errors) == 0 {
		return nil
	}
	msgs := make([]string, len(op.Error.Errors))
	for i, e := range op.Error.Errors {
		msgs[i] = e.Code + ": " + e.Message
	}
	return errors.New("%s", strings.Join(msgs, "; "))
}

// maxWaits bounds the wait loop. Each wait call blocks up to two minutes
// server side, and creating a template takes seconds.
const maxWaits = 5

// insert posts the template and waits for the operation to finish.
func insert(client Doer, token, url string, body []byte) error {
	op, err := call(client, token, url, body)
	if err != nil {
		return err
	}
	for i := 0; op.Status != "DONE"; i++ {
		if i == maxWaits {
			return errors.New("operation %s still %s", op.SelfLink, op.Status)
		}
		if op, err = call(client, token, op.SelfLink+"/wait", nil); err != nil {
			return err
		}
	}
	return op.err()
}

func call(client Doer, token, url string, body []byte) (operation, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return operation{}, errors.Wrap(err, "building request")
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return operation{}, errors.Wrap(err, "calling Compute API")
	}
	defer func() { _ = resp.Body.Close() }()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return operation{}, errors.Wrap(err, "reading Compute API response")
	}
	if resp.StatusCode >= http.StatusMultipleChoices {
		return operation{}, errors.New("compute API returned %s: %s", resp.Status, strings.TrimSpace(string(data)))
	}

	var op operation
	err = json.Unmarshal(data, &op)
	if err != nil {
		return operation{}, errors.Wrap(err, "invalid operation JSON")
	}
	return op, op.err()
}

func scope(region string) string {
	if region == "" {
		return "global"
	}
	return "regional in " + region
}
