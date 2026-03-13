package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAndResolveJiraSettings(t *testing.T) {
	repo := t.TempDir()
	ajDir := filepath.Join(repo, ".aj")
	if err := os.MkdirAll(ajDir, 0o755); err != nil {
		t.Fatalf("mkdir .aj: %v", err)
	}

	raw := `schema_version = 1
default_output = "brief"
default_lease_ttl = "4h"

[jira]
enabled = true
base_url = "https://example.atlassian.net"
project = "ABC"

[jira.status_map]
"To Do" = "todo"
"Done" = "done"
`
	if err := os.WriteFile(filepath.Join(ajDir, "config.toml"), []byte(raw), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	t.Setenv("AJ_JIRA_EMAIL", "agent@example.com")
	t.Setenv("AJ_JIRA_API_TOKEN", "secret")

	cfg, err := Load(repo)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if !cfg.Jira.Enabled || cfg.Jira.Project != "ABC" {
		t.Fatalf("unexpected jira config: %#v", cfg.Jira)
	}

	settings, err := ResolveJiraSettings(repo)
	if err != nil {
		t.Fatalf("resolve jira settings: %v", err)
	}
	if settings.BaseURL != "https://example.atlassian.net" || settings.Email != "agent@example.com" {
		t.Fatalf("unexpected jira settings: %#v", settings)
	}
}
