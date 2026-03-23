# ghlabels

Sync GitHub labels across repositories in an organization. Labels can be sourced from the organization's default label set or copied from an existing repository.

## Features

- **Sync to specific repos** or **all repos** in an organization
- **Copy labels from any repo** (`--copy-from-repo`) or from **another org's defaults** (`--copy-from-org`)
- **Org default labels** automatically used when no copy source is specified (via temp repo)
- **Dry-run mode** to preview changes without applying them
- **Add, update, and delete** labels to match the source exactly
- **Skip deletion** with `--no-delete` to only add/update labels
- **Exclude repos** from bulk sync with `--exclude`
- **Fork and archive filtering** — excluded by default, opt-in with flags
- **Automatic rate limit handling** with exponential backoff retries

## Prerequisites

- [Go](https://go.dev/) 1.21+ (to build)
- A GitHub token via one of:
  - `GITHUB_TOKEN` environment variable
  - `GH_TOKEN` environment variable
  - [gh CLI](https://cli.github.com/) authenticated (`gh auth login`)
- Token scopes: `repo`, `delete_repo` (if using org default labels via temp repo)

## Installation

```sh
go install github.com/kamontat/ghlabels@latest
```

Or build from source:

```sh
git clone https://github.com/kamontat/ghlabels.git
cd ghlabels
go build -o ghlabels .
```

## Usage

```sh
ghlabels [flags]
```

### Flags

| Flag | Description |
|------|-------------|
| `-r, --repo REPO` | Target repo(s) to sync (owner/repo, repeatable) |
| `-a, --all-repos OWNER` | Sync all repos in the given owner/org |
| `--copy-from-repo REPO` | Copy labels from an existing repo (owner/repo) |
| `--copy-from-org ORG` | Copy default labels from an organization |
| `-n, --dry-run` | Preview changes without applying them |
| `-e, --exclude REPO` | Exclude repo(s) from sync (with `--all-repos`, repeatable) |
| `--no-delete` | Only add/update labels, skip deleting extra labels |
| `--include-forks` | Include forked repos (excluded by default) |
| `--include-archived` | Include archived repos (excluded by default) |
| `--temp-repo NAME` | Custom temp repo name (default: `.github-kamontat-ghlabels`) |
| `-v, --verbose` | Show detailed output |
| `-h, --help` | Show help |

## Examples

### Sync org default labels to a single repo

```sh
ghlabels --repo my-org/my-repo --dry-run
```

### Sync org default labels to multiple repos

```sh
ghlabels --repo my-org/repo-a --repo my-org/repo-b
```

### Sync all repos in the organization

```sh
ghlabels --all-repos my-org
```

### Sync all repos except specific ones

```sh
ghlabels --all-repos my-org --exclude legacy-repo --exclude archived-project
```

### Copy labels from an existing repo to another

```sh
ghlabels --copy-from-repo my-org/source-repo --repo my-org/target-repo
```

### Copy labels from a repo in a different org

```sh
ghlabels --copy-from-repo other-org/source-repo --repo my-org/target-repo
```

### Copy default labels from a different org

```sh
ghlabels --copy-from-org other-org --repo my-org/target-repo
```

### Add/update labels only (keep existing labels that aren't in source)

```sh
ghlabels --copy-from-repo my-org/source-repo --repo my-org/target-repo --no-delete
```

### Sync all repos including forks and archived repos

```sh
ghlabels --all-repos my-org --include-forks --include-archived
```
