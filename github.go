package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/google/go-github/v72/github"
)

// Label represents a GitHub issue label.
type Label struct {
	Name        string
	Color       string
	Description string
}

// GitHubClient wraps the go-github client.
type GitHubClient struct {
	client *github.Client
}

// NewGitHubClient creates an authenticated GitHub client.
// It tries GITHUB_TOKEN, GH_TOKEN env vars, then falls back to `gh auth token`.
func NewGitHubClient() (*GitHubClient, error) {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		token = os.Getenv("GH_TOKEN")
	}
	if token == "" {
		out, err := exec.Command("gh", "auth", "token").Output()
		if err != nil {
			return nil, fmt.Errorf("no GitHub token found; set GITHUB_TOKEN, GH_TOKEN, or run 'gh auth login'")
		}
		token = strings.TrimSpace(string(out))
	}
	if token == "" {
		return nil, fmt.Errorf("empty GitHub token")
	}

	client := github.NewClient(nil).WithAuthToken(token)
	return &GitHubClient{client: client}, nil
}

// withRetry retries an API call on rate limit errors with exponential backoff.
func (g *GitHubClient) withRetry(ctx context.Context, fn func() (*github.Response, error)) error {
	const maxRetries = 3
	backoff := time.Second

	for attempt := 0; ; attempt++ {
		resp, err := fn()
		if err == nil {
			return nil
		}

		if resp != nil && (resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusForbidden) {
			if attempt >= maxRetries {
				return fmt.Errorf("rate limited after %d retries: %w", maxRetries, err)
			}
			logWarn("Rate limited, retrying in %s (attempt %d/%d)...", backoff, attempt+1, maxRetries)
			select {
			case <-time.After(backoff):
				backoff *= 2
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		return err
	}
}

// ListLabels returns all labels for a repository.
func (g *GitHubClient) ListLabels(ctx context.Context, owner, repo string) ([]Label, error) {
	var allLabels []Label
	opts := &github.ListOptions{PerPage: 100}

	for {
		labels, resp, err := g.client.Issues.ListLabels(ctx, owner, repo, opts)
		if err != nil {
			return nil, err
		}

		for _, l := range labels {
			allLabels = append(allLabels, Label{
				Name:        l.GetName(),
				Color:       l.GetColor(),
				Description: l.GetDescription(),
			})
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allLabels, nil
}

// CreateLabel creates a new label in the repository.
func (g *GitHubClient) CreateLabel(ctx context.Context, owner, repo string, label Label) error {
	return g.withRetry(ctx, func() (*github.Response, error) {
		_, resp, err := g.client.Issues.CreateLabel(ctx, owner, repo, &github.Label{
			Name:        github.Ptr(label.Name),
			Color:       github.Ptr(label.Color),
			Description: github.Ptr(label.Description),
		})
		return resp, err
	})
}

// UpdateLabel updates an existing label in the repository.
func (g *GitHubClient) UpdateLabel(ctx context.Context, owner, repo, currentName string, label Label) error {
	return g.withRetry(ctx, func() (*github.Response, error) {
		_, resp, err := g.client.Issues.EditLabel(ctx, owner, repo, currentName, &github.Label{
			Name:        github.Ptr(label.Name),
			Color:       github.Ptr(label.Color),
			Description: github.Ptr(label.Description),
		})
		return resp, err
	})
}

// DeleteLabel removes a label from the repository.
func (g *GitHubClient) DeleteLabel(ctx context.Context, owner, repo, name string) error {
	return g.withRetry(ctx, func() (*github.Response, error) {
		resp, err := g.client.Issues.DeleteLabel(ctx, owner, repo, name)
		return resp, err
	})
}

// IsOrganization checks whether the given owner is a GitHub organization.
func (g *GitHubClient) IsOrganization(ctx context.Context, owner string) (bool, error) {
	user, _, err := g.client.Users.Get(ctx, owner)
	if err != nil {
		return false, fmt.Errorf("failed to look up %q: %w", owner, err)
	}
	return user.GetType() == "Organization", nil
}

// CanManageRepos checks whether the authenticated user has permission to create and delete
// repos in the given organization (requires admin or owner role).
func (g *GitHubClient) CanManageRepos(ctx context.Context, org string) (bool, error) {
	membership, _, err := g.client.Organizations.GetOrgMembership(ctx, "", org)
	if err != nil {
		return false, err
	}
	role := membership.GetRole()
	return role == "admin" || role == "owner", nil
}

// ListOrgRepos returns all repo names in an org, optionally filtering forks and archived repos.
func (g *GitHubClient) ListOrgRepos(ctx context.Context, org string, includeForks, includeArchived bool) ([]string, error) {
	var allRepos []string
	opts := &github.RepositoryListByOrgOptions{
		ListOptions: github.ListOptions{PerPage: 100},
		Type:        "all",
	}

	for {
		repos, resp, err := g.client.Repositories.ListByOrg(ctx, org, opts)
		if err != nil {
			return nil, err
		}

		for _, r := range repos {
			if !includeForks && r.GetFork() {
				continue
			}
			if !includeArchived && r.GetArchived() {
				continue
			}
			allRepos = append(allRepos, r.GetName())
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allRepos, nil
}

// CreatePrivateRepo creates a new private repository in the org.
func (g *GitHubClient) CreatePrivateRepo(ctx context.Context, org, name, description string) error {
	_, _, err := g.client.Repositories.Create(ctx, org, &github.Repository{
		Name:        github.Ptr(name),
		Description: github.Ptr(description),
		Private:     github.Ptr(true),
	})
	return err
}

// DeleteRepo deletes a repository.
func (g *GitHubClient) DeleteRepo(ctx context.Context, owner, repo string) error {
	_, err := g.client.Repositories.Delete(ctx, owner, repo)
	return err
}
