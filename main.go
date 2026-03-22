package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

// Config holds all CLI configuration.
type Config struct {
	Org             string
	DryRun          bool
	Verbose         bool
	NoDelete        bool
	AllRepos        bool
	IncludeForks    bool
	IncludeArchived bool
	CopyFrom        string
	TempRepoName    string
	TargetRepos     []string
	ExcludeRepos    []string
}

// Stats tracks sync counters.
type Stats struct {
	ReposSynced   int
	LabelsAdded   int
	LabelsUpdated int
	LabelsDeleted int
	Errors        int
}

func main() {
	cfg := &Config{}

	rootCmd := &cobra.Command{
		Use:   "ghlabels [flags]",
		Short: "Sync GitHub labels across repositories in an organization",
		Long: `Sync organization default labels to repositories.
Labels can be sourced from the org's default label set (via a temporary repo)
or copied from an existing repository.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cfg)
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	f := rootCmd.Flags()
	f.StringVarP(&cfg.Org, "org", "o", "kc-workspace", "Organization name")
	f.BoolVarP(&cfg.DryRun, "dry-run", "n", false, "Show what would change without making changes")
	f.StringArrayVarP(&cfg.TargetRepos, "repo", "r", nil, "Sync specific repo(s) (repeatable)")
	f.BoolVarP(&cfg.AllRepos, "all-repos", "a", false, "Sync all repos in the organization")
	f.StringArrayVarP(&cfg.ExcludeRepos, "exclude", "e", nil, "Exclude repo(s) from sync (repeatable)")
	f.StringVar(&cfg.CopyFrom, "copy-from", "", "Copy labels from an existing repo instead of org defaults")
	f.BoolVar(&cfg.IncludeForks, "include-forks", false, "Include forked repos (excluded by default)")
	f.BoolVar(&cfg.IncludeArchived, "include-archived", false, "Include archived repos (excluded by default)")
	f.BoolVar(&cfg.NoDelete, "no-delete", false, "Skip deleting labels absent from source")
	f.StringVar(&cfg.TempRepoName, "temp-repo", ".github-label-sync-temp", "Temp repo name for reading org defaults")
	f.BoolVarP(&cfg.Verbose, "verbose", "v", false, "Verbose output")

	if err := rootCmd.Execute(); err != nil {
		logError("%s", err)
		os.Exit(1)
	}
}

func run(cfg *Config) error {
	if len(cfg.TargetRepos) == 0 && !cfg.AllRepos {
		return fmt.Errorf("no target repos specified; use --repo REPO or --all-repos")
	}
	if len(cfg.TargetRepos) > 0 && cfg.AllRepos {
		return fmt.Errorf("--repo and --all-repos are mutually exclusive")
	}

	if cfg.DryRun {
		logInfo("Running in DRY-RUN mode")
	}

	client, err := NewGitHubClient()
	if err != nil {
		return err
	}
	logInfo("GitHub client authenticated")

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Track temp repo for cleanup
	tempRepoCreated := false
	defer func() {
		if tempRepoCreated {
			logInfo("Cleaning up temp repo %s/%s...", cfg.Org, cfg.TempRepoName)
			_ = client.DeleteRepo(context.Background(), cfg.Org, cfg.TempRepoName)
		}
	}()

	// Fetch source labels
	var sourceLabels []Label
	if cfg.CopyFrom != "" {
		sourceLabels, err = fetchFromRepo(ctx, client, cfg)
	} else {
		sourceLabels, tempRepoCreated, err = fetchFromOrgDefaults(ctx, client, cfg)
	}
	if err != nil {
		return err
	}

	logInfo("Found %d source labels", len(sourceLabels))
	if len(sourceLabels) == 0 {
		logWarn("No source labels found. Nothing to sync.")
		return nil
	}

	if cfg.Verbose {
		logVerbose(true, "Source labels:")
		for _, l := range sourceLabels {
			fmt.Fprintf(os.Stderr, "  - %s (#%s)\n", l.Name, l.Color)
		}
	}

	// List target repos
	repos, err := listTargetRepos(ctx, client, cfg)
	if err != nil {
		return err
	}

	logInfo("Found %d repos to sync", len(repos))
	if len(repos) == 0 {
		logWarn("No repos to sync.")
		return nil
	}

	// Sync each repo
	stats := &Stats{}
	for i, repo := range repos {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		logVerbose(cfg.Verbose, "[%d/%d] Syncing %s...", i+1, len(repos), repo)

		if syncErr := syncRepo(ctx, client, cfg, repo, sourceLabels, stats); syncErr != nil {
			logError("Failed to sync %s: %s", repo, syncErr)
			stats.Errors++
		}
		stats.ReposSynced++

		// Small delay between repos to avoid rate limits
		time.Sleep(300 * time.Millisecond)
	}

	printSummary(cfg, stats)

	if stats.Errors > 0 {
		return fmt.Errorf("%d errors occurred during sync", stats.Errors)
	}
	return nil
}

func fetchFromRepo(ctx context.Context, client *GitHubClient, cfg *Config) ([]Label, error) {
	source := cfg.CopyFrom
	if !strings.Contains(source, "/") {
		source = cfg.Org + "/" + source
	}
	parts := strings.SplitN(source, "/", 2)
	logInfo("Fetching labels from %s...", source)
	labels, err := client.ListLabels(ctx, parts[0], parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to fetch labels from %s: %w", source, err)
	}
	return labels, nil
}

func fetchFromOrgDefaults(ctx context.Context, client *GitHubClient, cfg *Config) ([]Label, bool, error) {
	logInfo("Creating temp repo %s/%s to read org default labels...", cfg.Org, cfg.TempRepoName)
	if err := client.CreatePrivateRepo(ctx, cfg.Org, cfg.TempRepoName, "Temporary repo for label sync - will be auto-deleted"); err != nil {
		return nil, false, fmt.Errorf("failed to create temp repo: %w", err)
	}
	tempRepoCreated := true

	// Poll until labels stabilize — GitHub propagates default labels asynchronously
	logVerbose(cfg.Verbose, "Waiting for default labels to propagate...")
	var labels []Label
	prevCount := -1
	for attempt := 0; attempt < 10; attempt++ {
		time.Sleep(2 * time.Second)

		labels, err = client.ListLabels(ctx, cfg.Org, cfg.TempRepoName)
		if err != nil {
			return nil, tempRepoCreated, fmt.Errorf("failed to fetch labels from temp repo: %w", err)
		}

		logVerbose(cfg.Verbose, "Attempt %d: found %d labels", attempt+1, len(labels))
		if len(labels) > 0 && len(labels) == prevCount {
			break
		}
		prevCount = len(labels)
	}

	logInfo("Deleting temp repo...")
	if err := client.DeleteRepo(ctx, cfg.Org, cfg.TempRepoName); err != nil {
		logWarn("Failed to delete temp repo immediately; cleanup will handle it")
	} else {
		tempRepoCreated = false
	}

	return labels, tempRepoCreated, nil
}

func listTargetRepos(ctx context.Context, client *GitHubClient, cfg *Config) ([]string, error) {
	if !cfg.AllRepos {
		return cfg.TargetRepos, nil
	}

	logInfo("Listing repos in %s...", cfg.Org)
	repos, err := client.ListOrgRepos(ctx, cfg.Org, cfg.IncludeForks, cfg.IncludeArchived)
	if err != nil {
		return nil, fmt.Errorf("failed to list repos: %w", err)
	}

	excludeSet := make(map[string]bool)
	for _, e := range cfg.ExcludeRepos {
		excludeSet[e] = true
	}
	excludeSet[cfg.TempRepoName] = true

	var filtered []string
	for _, r := range repos {
		if excludeSet[r] {
			logVerbose(cfg.Verbose, "Excluding repo: %s", r)
			continue
		}
		filtered = append(filtered, r)
	}

	return filtered, nil
}

func printSummary(cfg *Config, stats *Stats) {
	fmt.Fprintln(os.Stderr)
	logInfo("========== Summary ==========")
	if cfg.DryRun {
		logInfo("Mode: DRY-RUN (no changes made)")
	}
	logInfo("Repos processed: %d", stats.ReposSynced)
	logInfo("Labels added:    %d", stats.LabelsAdded)
	logInfo("Labels updated:  %d", stats.LabelsUpdated)
	logInfo("Labels deleted:  %d", stats.LabelsDeleted)
	if stats.Errors > 0 {
		logError("Errors: %d", stats.Errors)
	} else {
		logInfo("Errors: 0")
	}
	logInfo("=============================")
}
