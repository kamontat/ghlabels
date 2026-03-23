package main

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr string
		wantOrg string
	}{
		{
			name:    "no target specified",
			cfg:     Config{},
			wantErr: "no target repos specified",
		},
		{
			name:    "repo and all-repos mutually exclusive",
			cfg:     Config{Repos: []string{"org/repo"}, AllRepos: "org"},
			wantErr: "--repo and --all-repos are mutually exclusive",
		},
		{
			name:    "copy-from-repo and copy-from-org mutually exclusive",
			cfg:     Config{Repos: []string{"org/repo"}, CopyFromRepo: "org/src", CopyFromOrg: "other-org"},
			wantErr: "--copy-from-repo and --copy-from-org are mutually exclusive",
		},
		{
			name:    "copy-from-repo must be owner/repo format",
			cfg:     Config{Repos: []string{"org/repo"}, CopyFromRepo: "just-repo"},
			wantErr: "--copy-from-repo must be in owner/repo format",
		},
		{
			name:    "repo missing slash",
			cfg:     Config{Repos: []string{"just-repo"}},
			wantErr: "--repo must be in owner/repo format",
		},
		{
			name:    "repo second value missing slash",
			cfg:     Config{Repos: []string{"org/repo1", "bad"}},
			wantErr: "--repo must be in owner/repo format",
		},
		{
			name:    "valid single repo",
			cfg:     Config{Repos: []string{"myorg/myrepo"}},
			wantOrg: "myorg",
		},
		{
			name:    "valid multiple repos",
			cfg:     Config{Repos: []string{"owner/repo1", "owner/repo2"}},
			wantOrg: "owner",
		},
		{
			name:    "valid all-repos",
			cfg:     Config{AllRepos: "myorg"},
			wantOrg: "myorg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.cfg
			err := validateConfig(&cfg)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %q", tt.wantErr, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if cfg.Org != tt.wantOrg {
				t.Errorf("Org = %q, want %q", cfg.Org, tt.wantOrg)
			}
		})
	}
}

func TestListTargetRepos_SingleRepoMode(t *testing.T) {
	tests := []struct {
		name      string
		repos     []string
		wantRepos []string
	}{
		{
			name:      "single repo extracts name",
			repos:     []string{"owner/my-repo"},
			wantRepos: []string{"my-repo"},
		},
		{
			name:      "multiple repos extract names",
			repos:     []string{"org/repo-a", "org/repo-b", "org/repo-c"},
			wantRepos: []string{"repo-a", "repo-b", "repo-c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Repos: tt.repos}
			repos, err := listTargetRepos(nil, nil, cfg)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if len(repos) != len(tt.wantRepos) {
				t.Fatalf("got %d repos, want %d", len(repos), len(tt.wantRepos))
			}
			for i, r := range repos {
				if r != tt.wantRepos[i] {
					t.Errorf("repos[%d] = %q, want %q", i, r, tt.wantRepos[i])
				}
			}
		})
	}
}

func TestFlagParsing(t *testing.T) {
	tests := []struct {
		name  string
		args  []string
		check func(t *testing.T, cfg *Config)
	}{
		{
			name: "short flags",
			args: []string{"-r", "org/repo", "-n", "-v", "-e", "skip"},
			check: func(t *testing.T, cfg *Config) {
				if len(cfg.Repos) != 1 || cfg.Repos[0] != "org/repo" {
					t.Errorf("Repos = %v, want [org/repo]", cfg.Repos)
				}
				if !cfg.DryRun {
					t.Error("DryRun should be true")
				}
				if !cfg.Verbose {
					t.Error("Verbose should be true")
				}
				if len(cfg.ExcludeRepos) != 1 || cfg.ExcludeRepos[0] != "skip" {
					t.Errorf("ExcludeRepos = %v, want [skip]", cfg.ExcludeRepos)
				}
			},
		},
		{
			name: "long flags",
			args: []string{"--repo", "org/repo", "--dry-run", "--verbose", "--no-delete",
				"--copy-from-repo", "org/src", "--include-archived", "--include-forks",
				"--exclude", "x", "--temp-repo", "my-temp", "--all-repos", "myorg"},
			check: func(t *testing.T, cfg *Config) {
				if cfg.AllRepos != "myorg" {
					t.Errorf("AllRepos = %q, want %q", cfg.AllRepos, "myorg")
				}
				if cfg.CopyFromRepo != "org/src" {
					t.Errorf("CopyFromRepo = %q, want %q", cfg.CopyFromRepo, "org/src")
				}
				if !cfg.NoDelete {
					t.Error("NoDelete should be true")
				}
				if !cfg.IncludeArchived {
					t.Error("IncludeArchived should be true")
				}
				if !cfg.IncludeForks {
					t.Error("IncludeForks should be true")
				}
				if cfg.TempRepoName != "my-temp" {
					t.Errorf("TempRepoName = %q, want %q", cfg.TempRepoName, "my-temp")
				}
			},
		},
		{
			name: "copy-from-org flag",
			args: []string{"--repo", "org/repo", "--copy-from-org", "other-org"},
			check: func(t *testing.T, cfg *Config) {
				if cfg.CopyFromOrg != "other-org" {
					t.Errorf("CopyFromOrg = %q, want %q", cfg.CopyFromOrg, "other-org")
				}
			},
		},
		{
			name: "repeatable repo flag",
			args: []string{"--repo", "org/a", "--repo", "org/b", "--repo", "org/c"},
			check: func(t *testing.T, cfg *Config) {
				if len(cfg.Repos) != 3 {
					t.Fatalf("Repos length = %d, want 3", len(cfg.Repos))
				}
				want := []string{"org/a", "org/b", "org/c"}
				for i, r := range cfg.Repos {
					if r != want[i] {
						t.Errorf("Repos[%d] = %q, want %q", i, r, want[i])
					}
				}
			},
		},
		{
			name: "repeatable exclude flag",
			args: []string{"--all-repos", "org", "--exclude", "x", "--exclude", "y"},
			check: func(t *testing.T, cfg *Config) {
				if len(cfg.ExcludeRepos) != 2 {
					t.Fatalf("ExcludeRepos length = %d, want 2", len(cfg.ExcludeRepos))
				}
				if cfg.ExcludeRepos[0] != "x" || cfg.ExcludeRepos[1] != "y" {
					t.Errorf("ExcludeRepos = %v, want [x y]", cfg.ExcludeRepos)
				}
			},
		},
		{
			name: "temp-repo default",
			args: []string{"--repo", "org/repo"},
			check: func(t *testing.T, cfg *Config) {
				if cfg.TempRepoName != ".github-kamontat-ghlabels" {
					t.Errorf("TempRepoName = %q, want %q", cfg.TempRepoName, ".github-kamontat-ghlabels")
				}
			},
		},
		{
			name: "all-repos short flag",
			args: []string{"-a", "myorg"},
			check: func(t *testing.T, cfg *Config) {
				if cfg.AllRepos != "myorg" {
					t.Errorf("AllRepos = %q, want %q", cfg.AllRepos, "myorg")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{}
			cmd := buildCommand(cfg)
			cmd.SetArgs(tt.args)
			cmd.RunE = func(cmd *cobra.Command, args []string) error { return nil }
			if err := cmd.Execute(); err != nil {
				t.Fatalf("command execution failed: %s", err)
			}
			tt.check(t, cfg)
		})
	}
}
