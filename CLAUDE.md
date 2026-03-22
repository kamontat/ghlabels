# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

Go CLI tool (`ghlabels`) that syncs GitHub labels across repositories in an organization. It can copy labels from an existing repo or use org default labels (by creating/deleting a temporary repo to read defaults).

## Prerequisites

- Go 1.21+
- A GitHub token (via `GITHUB_TOKEN`, `GH_TOKEN` env var, or `gh auth login`)

## Building & Running

```sh
# Build
go build -o ghlabels .

# Dry run against a single repo
./ghlabels --repo my-repo --dry-run

# Sync all repos in org (default org: kc-workspace)
./ghlabels --all-repos

# Copy from a specific source repo
./ghlabels --copy-from source-repo --repo target-repo
```

## Architecture

The project is organized into four Go files, all in `package main`:

- **`main.go`**: CLI entry point using cobra, argument validation, orchestration (`run` function), source label fetching, repo listing, and summary output
- **`github.go`**: `GitHubClient` wrapping `google/go-github` — handles auth (GITHUB_TOKEN / GH_TOKEN / `gh auth token`), API calls with exponential backoff retry on rate limits, and all GitHub operations (labels CRUD, repo list/create/delete)
- **`sync.go`**: `syncRepo` — per-repo sync logic with case-insensitive label matching (via `strings.ToLower`), add/update/delete operations, and stats tracking
- **`log.go`**: Colored log output (info/warn/error/dry-run/verbose) to stderr

Label comparison is case-insensitive. Labels are matched by name; color and description differences trigger updates. Temp repo cleanup uses a deferred function.
