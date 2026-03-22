# Contributing to ghlabels

Thank you for your interest in contributing! This guide covers how to set up, develop, test, and release.

## Prerequisites

- [Go](https://go.dev/) 1.21+
- [GoReleaser](https://goreleaser.com/) (for local release testing)
- [Docker](https://www.docker.com/) (for building container images)
- A GitHub token (`GITHUB_TOKEN`, `GH_TOKEN`, or `gh auth login`)

Alternatively, install tools via [mise](https://mise.jdx.dev/):

```sh
mise install
```

## Development

### Clone and build

```sh
git clone https://github.com/kamontat/ghlabels.git
cd ghlabels
go build -o ghlabels .
```

### Project structure

| File | Purpose |
|------|---------|
| `main.go` | CLI entry point (cobra), argument validation, orchestration |
| `github.go` | GitHub API client with auth and retry logic |
| `sync.go` | Per-repo label sync logic |
| `log.go` | Colored log output to stderr |

### Making changes

1. Create a branch from `main`:
   ```sh
   git checkout -b my-feature
   ```
2. Make your changes.
3. Build and verify:
   ```sh
   go build -o ghlabels .
   go vet ./...
   ```
4. Test manually with `--dry-run` against a test repo:
   ```sh
   ./ghlabels --repo my-test-repo --dry-run --verbose
   ```
5. Commit and open a pull request against `main`.

## Testing

### CI

A CI workflow runs automatically on pushes to `main` and on pull requests. It performs:

- `go build` and `go vet`
- `--help` output verification
- A `--dry-run --verbose` sync against this repo itself (copies labels from the repo back to itself, confirming the full code path runs without error)

### Manual testing

This project also relies on manual testing with `--dry-run` mode. When testing changes:

- Always use `--dry-run` first to verify what would change without modifying anything.
- Use `--verbose` to see detailed output.
- Test both `--copy-from` (copy from an existing repo) and org default label flows.
- Test with `--repo` (single target) and `--all-repos` (bulk).
- Verify edge cases: empty label sets, repos with no labels, `--no-delete` behavior.

```sh
# Preview sync from org defaults
./ghlabels --repo test-repo --dry-run --verbose

# Preview sync from another repo
./ghlabels --copy-from source-repo --repo test-repo --dry-run --verbose

# Preview bulk sync with exclusions
./ghlabels --all-repos --exclude important-repo --dry-run
```

## Release flow

Releases are automated via [GoReleaser](https://goreleaser.com/) and GitHub Actions.

### What gets published

- **Binaries**: Linux, macOS, and Windows for both amd64 and arm64
- **Docker images**: Multi-arch (`amd64`/`arm64`) pushed to `ghcr.io/kamontat/ghlabels`
- **GitHub Release**: With auto-generated changelog and downloadable archives

### How to release

1. Ensure `main` is in a releasable state.
2. Tag the commit with a semver version prefixed with `v`:
   ```sh
   git tag v0.1.0
   git push origin v0.1.0
   ```
3. GitHub Actions will automatically:
   - Build binaries for all platforms
   - Build and push Docker images to GHCR
   - Create a GitHub Release with changelog and artifacts

### Testing the release locally

You can dry-run GoReleaser locally to verify the configuration:

```sh
goreleaser release --snapshot --clean
```

This builds all artifacts without publishing. Check the `dist/` directory for output.

### Versioning

This project uses [Semantic Versioning](https://semver.org/):

- **Patch** (`v0.0.x`): Bug fixes
- **Minor** (`v0.x.0`): New features, backwards compatible
- **Major** (`vx.0.0`): Breaking changes

### Changelog

The changelog is auto-generated from commit messages. Commits prefixed with `docs:`, `test:`, or `chore:` are excluded. Write clear, descriptive commit messages for meaningful release notes.
