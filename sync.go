package main

import (
	"context"
	"strings"
)

func syncRepo(ctx context.Context, client *GitHubClient, cfg *Config, repo string, sourceLabels []Label, stats *Stats) error {
	fullRepo := cfg.Org + "/" + repo

	logVerbose(cfg.Verbose, "Fetching current labels for %s...", fullRepo)
	targetLabels, err := client.ListLabels(ctx, cfg.Org, repo)
	if err != nil {
		return err
	}

	// Build case-insensitive lookup maps
	targetMap := make(map[string]Label)
	for _, l := range targetLabels {
		targetMap[strings.ToLower(l.Name)] = l
	}

	sourceMap := make(map[string]Label)
	for _, l := range sourceLabels {
		sourceMap[strings.ToLower(l.Name)] = l
	}

	repoAdded, repoUpdated, repoDeleted := 0, 0, 0

	// Add / Update
	for _, src := range sourceLabels {
		lower := strings.ToLower(src.Name)

		if tgt, exists := targetMap[lower]; !exists {
			if cfg.DryRun {
				logDry("Would add label '%s' (color: #%s) to %s", src.Name, src.Color, fullRepo)
			} else {
				logVerbose(cfg.Verbose, "Adding label '%s' to %s", src.Name, fullRepo)
				if err := client.CreateLabel(ctx, cfg.Org, repo, src); err != nil {
					logError("Failed to add label '%s' to %s: %s", src.Name, fullRepo, err)
					stats.Errors++
					continue
				}
			}
			repoAdded++
		} else if src.Color != tgt.Color || src.Description != tgt.Description {
			if cfg.DryRun {
				logDry("Would update label '%s' in %s (color: #%s -> #%s)", src.Name, fullRepo, tgt.Color, src.Color)
			} else {
				logVerbose(cfg.Verbose, "Updating label '%s' in %s", src.Name, fullRepo)
				if err := client.UpdateLabel(ctx, cfg.Org, repo, tgt.Name, src); err != nil {
					logError("Failed to update label '%s' in %s: %s", src.Name, fullRepo, err)
					stats.Errors++
					continue
				}
			}
			repoUpdated++
		}
	}

	// Delete
	if !cfg.NoDelete {
		for _, tgt := range targetLabels {
			lower := strings.ToLower(tgt.Name)
			if _, exists := sourceMap[lower]; !exists {
				if cfg.DryRun {
					logDry("Would delete label '%s' from %s", tgt.Name, fullRepo)
				} else {
					logVerbose(cfg.Verbose, "Deleting label '%s' from %s", tgt.Name, fullRepo)
					if err := client.DeleteLabel(ctx, cfg.Org, repo, tgt.Name); err != nil {
						logError("Failed to delete label '%s' from %s: %s", tgt.Name, fullRepo, err)
						stats.Errors++
						continue
					}
				}
				repoDeleted++
			}
		}
	}

	stats.LabelsAdded += repoAdded
	stats.LabelsUpdated += repoUpdated
	stats.LabelsDeleted += repoDeleted

	total := repoAdded + repoUpdated + repoDeleted
	if total > 0 {
		action := "Synced"
		if cfg.DryRun {
			action = "Would sync"
		}
		logInfo("%s %s: +%d ~%d -%d", action, fullRepo, repoAdded, repoUpdated, repoDeleted)
	} else {
		logVerbose(cfg.Verbose, "No changes needed for %s", fullRepo)
	}

	return nil
}
