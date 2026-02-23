package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/sid-technologies/pilum/lib/cache"
	"github.com/sid-technologies/pilum/lib/errors"
	"github.com/sid-technologies/pilum/lib/exitcodes"
	"github.com/sid-technologies/pilum/lib/output"
	"github.com/sid-technologies/pilum/lib/path"

	"github.com/spf13/cobra"
)

func CacheCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cache",
		Short: "Manage the build cache",
		Long:  "View and manage the hash-based build cache stored in .pilum/build-cache.json.",
	}

	cmd.AddCommand(cacheStatusCmd())
	cmd.AddCommand(cacheClearCmd())

	return cmd
}

func cacheStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "status",
		Aliases: []string{"st"},
		Short:   "Show build cache status",
		Long:    "Show cached services, their content hashes, tags, and timestamps.",
		RunE: withJSON(func(_ *cobra.Command, _ []string) (any, error) {
			root, err := path.FindProjectRoot()
			if err != nil {
				return nil, exitcodes.WithCode(exitcodes.NoServices,
					errors.Wrap(err, "error finding project root"))
			}

			c, err := cache.Load(root)
			if err != nil {
				return nil, errors.Wrap(err, "error loading cache")
			}

			if len(c) == 0 {
				output.Info("No build cache found")
				return map[string]any{"entries": map[string]any{}, "file_size": 0}, nil
			}

			// Get file size
			var fileSize int64
			if info, statErr := os.Stat(cache.FilePath(root)); statErr == nil {
				fileSize = info.Size()
			}

			printCacheStatus(c, fileSize)

			return map[string]any{
				"entries":   c,
				"file_size": fileSize,
			}, nil
		}),
	}
}

func cacheClearCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "clear",
		Aliases: []string{"rm"},
		Short:   "Clear the build cache",
		Long:    "Delete the build cache file, or clear a single service entry with --service.",
		RunE: withJSON(func(cmd *cobra.Command, _ []string) (any, error) {
			root, err := path.FindProjectRoot()
			if err != nil {
				return nil, exitcodes.WithCode(exitcodes.NoServices,
					errors.Wrap(err, "error finding project root"))
			}

			service, _ := cmd.Flags().GetString("service")

			if service != "" {
				return clearSingleService(root, service)
			}

			return clearAllCache(root)
		}),
	}

	cmd.Flags().StringP("service", "s", "", "Clear cache for a specific service only")

	return cmd
}

func clearSingleService(root, service string) (any, error) {
	c, err := cache.Load(root)
	if err != nil {
		return nil, errors.Wrap(err, "error loading cache")
	}

	if _, exists := c[service]; !exists {
		output.Warning("No cache entry for %s", service)
		return map[string]any{"success": true, "cleared": 0}, nil
	}

	delete(c, service)

	if err := cache.Save(root, c); err != nil {
		return nil, errors.Wrap(err, "error saving cache")
	}

	output.Success("Cleared cache for %s", service)
	return map[string]any{"success": true, "cleared": 1, "service": service}, nil
}

func clearAllCache(root string) (any, error) {
	// Load first to report count
	c, _ := cache.Load(root)
	count := len(c)

	fp := cache.FilePath(root)
	if err := os.Remove(fp); err != nil && !os.IsNotExist(err) {
		return nil, exitcodes.WithCode(exitcodes.IO, errors.Wrap(err, "error removing cache file"))
	}

	if count == 0 {
		output.Info("No build cache to clear")
	} else {
		output.Success("Cleared all build cache (%d entries)", count)
	}

	return map[string]any{"success": true, "cleared": count}, nil
}

func printCacheStatus(c cache.Cache, fileSize int64) {
	sizeStr := formatBytes(fileSize)
	output.Header(fmt.Sprintf("Build Cache (%d entries, %s)", len(c), sizeStr))

	for name, entry := range c {
		hash := entry.Hash
		if len(hash) > 12 {
			hash = hash[:12]
		}

		age := time.Since(entry.Timestamp).Round(time.Second)
		ageStr := formatAge(age)

		fmt.Printf("  %-24s %s%s%s  %-10s %s%s%s\n",
			name,
			output.Muted, hash, output.Reset,
			entry.Tag,
			output.Muted, ageStr, output.Reset,
		)
	}
	fmt.Println()
}

func formatAge(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

func formatBytes(b int64) string {
	switch {
	case b < 1024:
		return fmt.Sprintf("%d B", b)
	case b < 1024*1024:
		return fmt.Sprintf("%.1f KB", float64(b)/1024)
	default:
		return fmt.Sprintf("%.1f MB", float64(b)/(1024*1024))
	}
}

// nolint: gochecknoinits // Standard Cobra pattern for initializing commands
func init() {
	rootCmd.AddCommand(CacheCmd())
}
