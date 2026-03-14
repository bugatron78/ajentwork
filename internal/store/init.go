package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type InitOptions struct {
	RepoPath          string
	Force             bool
	JiraEnabled       bool
	JiraBaseURL       string
	JiraProject       string
	EnsureJiraSpace   bool
	JiraSpaceName     string
	JiraSpaceType     string
	JiraSpaceTemplate string
}

type InitResult struct {
	RepoPath         string `json:"repo_path"`
	ConfigPath       string `json:"config_path"`
	Created          bool   `json:"created"`
	AlreadyReady     bool   `json:"already_ready"`
	JiraEnabled      bool   `json:"jira_enabled"`
	JiraProject      string `json:"jira_project,omitempty"`
	JiraSpaceCreated bool   `json:"jira_space_created,omitempty"`
}

func InitRepo(opts InitOptions) (InitResult, error) {
	if opts.RepoPath == "" {
		return InitResult{}, errors.New("repo path is required")
	}

	root := filepath.Clean(opts.RepoPath)
	ajDir := filepath.Join(root, ".aj")
	configPath := filepath.Join(ajDir, "config.toml")
	createdAny := false
	opts = normalizeInitOptions(opts)

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
			JiraEnabled:  opts.JiraEnabled,
			JiraProject:  opts.JiraProject,
		}, nil
	case err == nil && opts.Force:
		createdAny = true
	case err != nil && !errors.Is(err, os.ErrNotExist):
		return InitResult{}, fmt.Errorf("stat %s: %w", configPath, err)
	default:
		createdAny = true
	}

	if err := os.WriteFile(configPath, []byte(defaultConfig(opts)), 0o644); err != nil {
		return InitResult{}, fmt.Errorf("write %s: %w", configPath, err)
	}

	result := InitResult{
		RepoPath:     root,
		ConfigPath:   configPath,
		Created:      createdAny,
		AlreadyReady: false,
		JiraEnabled:  opts.JiraEnabled,
		JiraProject:  opts.JiraProject,
	}

	if opts.EnsureJiraSpace {
		spaceResult, err := EnsureJiraSpace(JiraSpaceEnsureOptions{
			RepoPath: root,
			Key:      opts.JiraProject,
			Name:     opts.JiraSpaceName,
			Type:     opts.JiraSpaceType,
			Template: opts.JiraSpaceTemplate,
		})
		if err != nil {
			return InitResult{}, err
		}
		result.JiraSpaceCreated = spaceResult.Created
		result.JiraProject = spaceResult.Space.Key
	}

	return result, nil
}

func defaultConfig(opts InitOptions) string {
	return fmt.Sprintf(`schema_version = 1
default_output = "brief"
default_lease_ttl = "4h"

[jira]
enabled = %t
base_url = %q
project = %q

[jira.status_map]
"To Do" = "todo"
"In Progress" = "in_progress"
"Blocked" = "blocked"
"In Review" = "in_review"
"Done" = "done"

[jira.lifecycle]
comment_on_done = false
comment_on_block = false
comment_on_handoff = false
`, opts.JiraEnabled, opts.JiraBaseURL, opts.JiraProject)
}

func normalizeInitOptions(opts InitOptions) InitOptions {
	opts.JiraBaseURL = strings.TrimSpace(opts.JiraBaseURL)
	opts.JiraProject = strings.TrimSpace(opts.JiraProject)
	opts.JiraSpaceName = strings.TrimSpace(opts.JiraSpaceName)
	opts.JiraSpaceType = strings.TrimSpace(opts.JiraSpaceType)
	opts.JiraSpaceTemplate = strings.TrimSpace(opts.JiraSpaceTemplate)

	if opts.EnsureJiraSpace {
		opts.JiraEnabled = true
	}
	if opts.JiraProject != "" || opts.JiraBaseURL != "" {
		opts.JiraEnabled = true
	}
	return opts
}
