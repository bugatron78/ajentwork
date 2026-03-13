package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type InitOptions struct {
	RepoPath string
	Force    bool
}

type InitResult struct {
	RepoPath     string `json:"repo_path"`
	ConfigPath   string `json:"config_path"`
	Created      bool   `json:"created"`
	AlreadyReady bool   `json:"already_ready"`
}

func InitRepo(opts InitOptions) (InitResult, error) {
	if opts.RepoPath == "" {
		return InitResult{}, errors.New("repo path is required")
	}

	root := filepath.Clean(opts.RepoPath)
	ajDir := filepath.Join(root, ".aj")
	configPath := filepath.Join(ajDir, "config.toml")
	createdAny := false

	dirs := []string{
		ajDir,
		filepath.Join(ajDir, "issues"),
		filepath.Join(ajDir, "artifacts"),
		filepath.Join(ajDir, "cache"),
		filepath.Join(ajDir, "locks"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return InitResult{}, fmt.Errorf("create %s: %w", dir, err)
		}
	}

	_, err := os.Stat(configPath)
	switch {
	case err == nil && !opts.Force:
		return InitResult{
			RepoPath:     root,
			ConfigPath:   configPath,
			Created:      false,
			AlreadyReady: true,
		}, nil
	case err == nil && opts.Force:
		createdAny = true
	case err != nil && !errors.Is(err, os.ErrNotExist):
		return InitResult{}, fmt.Errorf("stat %s: %w", configPath, err)
	default:
		createdAny = true
	}

	if err := os.WriteFile(configPath, []byte(defaultConfig()), 0o644); err != nil {
		return InitResult{}, fmt.Errorf("write %s: %w", configPath, err)
	}

	return InitResult{
		RepoPath:     root,
		ConfigPath:   configPath,
		Created:      createdAny,
		AlreadyReady: false,
	}, nil
}

func defaultConfig() string {
	return `schema_version = 1
default_output = "brief"
default_lease_ttl = "4h"

[jira]
enabled = false
base_url = ""
project = ""

[jira.status_map]
"To Do" = "todo"
"In Progress" = "in_progress"
"Blocked" = "blocked"
"In Review" = "in_review"
"Done" = "done"
`
}
