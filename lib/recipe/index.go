package recipe

import (
	"slices"
	"strings"

	"github.com/sid-technologies/pilum/lib/errors"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
)

// Index finds the recipe for a service. The pipeline and `pilum check` both
// use it, so a service check accepts is one the pipeline can run.
type Index struct {
	byKey      map[string]Recipe
	byProvider map[string][]string
}

// NewIndex indexes recipes by their composite key ("gcp-cloud-run") and by
// provider.
func NewIndex(recipes []Info) *Index {
	ix := &Index{
		byKey:      make(map[string]Recipe, len(recipes)),
		byProvider: make(map[string][]string),
	}

	for _, rec := range recipes {
		key := rec.RecipeKey()
		ix.byKey[key] = rec.Recipe
		ix.byProvider[rec.Provider] = append(ix.byProvider[rec.Provider], key)
	}

	for _, keys := range ix.byProvider {
		slices.Sort(keys)
	}

	return ix
}

// Len returns the number of indexed recipes.
func (ix *Index) Len() int {
	return len(ix.byKey)
}

// Find returns the recipe for svc: the recipe whose key is the service's
// type, or else the provider's recipe when the provider has only one (so
// `provider: homebrew` finds homebrew-package). A provider with several
// recipes is never guessed: `provider: gcp` alone used to pick whichever gcp
// recipe sorted first, which builds and pushes but never deploys.
func (ix *Index) Find(svc *serviceinfo.ServiceInfo) (Recipe, bool) {
	rec, problem := ix.find(svc)
	return rec, problem == ""
}

// Lookup is Find with an error that says why nothing matched.
func (ix *Index) Lookup(svc *serviceinfo.ServiceInfo) (Recipe, error) {
	rec, problem := ix.find(svc)
	if problem != "" {
		return Recipe{}, errors.New("%s", problem)
	}

	return rec, nil
}

func (ix *Index) find(svc *serviceinfo.ServiceInfo) (Recipe, string) {
	key := svc.RecipeKey()

	rec, ok := ix.byKey[key]
	if ok {
		return rec, ""
	}

	// A type naming a recipe that doesn't exist is a typo, not a request
	// for the provider's default.
	if svc.Type != "" && svc.Type != svc.Provider {
		return Recipe{}, "no recipe named '" + key + "'"
	}

	candidates := ix.byProvider[svc.Provider]
	switch len(candidates) {
	case 0:
		return Recipe{}, "no recipe for '" + key + "'"
	case 1:
		return ix.byKey[candidates[0]], ""
	}

	return Recipe{}, "provider '" + svc.Provider + "' has several recipes; set type: to one of " +
		strings.Join(candidates, ", ")
}
