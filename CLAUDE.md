# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

Go CLI tool (`ghlabels`) that syncs GitHub labels across repositories in an organization. It can copy labels from an existing repo (`--copy-from-repo`), from another org's defaults (`--copy-from-org`), or use the target org's default labels (by creating/deleting a temporary repo to read defaults).

## Prerequisites

- Go 1.21+
- A GitHub token (via `GITHUB_TOKEN`, `GH_TOKEN` env var, or `gh auth login`)

## Building & Running

```sh
# Build
go build -o ghlabels .

# Dry run against a single repo
./ghlabels --repo my-org/my-repo --dry-run

# Sync all repos in org
./ghlabels --all-repos my-org

# Copy from a specific source repo
./ghlabels --copy-from-repo my-org/source-repo --repo my-org/target-repo

# Copy default labels from another org
./ghlabels --copy-from-org other-org --repo my-org/target-repo
```

## Architecture

The project is organized into four Go files, all in `package main`:

- **`main.go`**: CLI entry point using cobra, argument validation, orchestration (`run` function), source label fetching, repo listing, and summary output
- **`github.go`**: `GitHubClient` wrapping `google/go-github` — handles auth (GITHUB_TOKEN / GH_TOKEN / `gh auth token`), API calls with exponential backoff retry on rate limits, org membership/permission checks, and all GitHub operations (labels CRUD, repo list/create/delete)
- **`sync.go`**: `syncRepo` — per-repo sync logic with case-insensitive label matching (via `strings.ToLower`), add/update/delete operations, and stats tracking
- **`log.go`**: Colored log output (info/warn/error/dry-run/verbose) to stderr

Label comparison is case-insensitive. Labels are matched by name; color and description differences trigger updates. Temp repo cleanup uses a deferred function.
