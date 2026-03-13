package config

import (
	"os"
	"path/filepath"
	"strings"
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

[jira.lifecycle]
comment_on_done = true
comment_on_block = false
comment_on_handoff = true
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
	if !cfg.Jira.Lifecycle.CommentOnDone || cfg.Jira.Lifecycle.CommentOnBlock || !cfg.Jira.Lifecycle.CommentOnHandoff {
		t.Fatalf("unexpected jira lifecycle config: %#v", cfg.Jira.Lifecycle)
	}

	settings, err := ResolveJiraSettings(repo)
	if err != nil {
		t.Fatalf("resolve jira settings: %v", err)
	}
	if settings.BaseURL != "https://example.atlassian.net" || settings.Email != "agent@example.com" {
		t.Fatalf("unexpected jira settings: %#v", settings)
	}
}

func TestResolveJiraSettingsGuidesDisabledConfig(t *testing.T) {
	repo := t.TempDir()
	writeTestConfig(t, repo, `schema_version = 1
default_output = "brief"
default_lease_ttl = "4h"

[jira]
enabled = false
base_url = "https://example.atlassian.net"
project = "ABC"
`)

	_, err := ResolveJiraSettings(repo)
	if err == nil {
		t.Fatalf("expected jira disabled error")
	}
	if !strings.Contains(err.Error(), "jira is not enabled for this repo.") || !strings.Contains(err.Error(), "See: aj help jira") {
		t.Fatalf("unexpected disabled guidance: %v", err)
	}
}

func TestResolveJiraSettingsGuidesMissingBaseURL(t *testing.T) {
	repo := t.TempDir()
	writeTestConfig(t, repo, `schema_version = 1
default_output = "brief"
default_lease_ttl = "4h"

[jira]
enabled = true
project = "ABC"
`)
	t.Setenv("AJ_JIRA_EMAIL", "agent@example.com")
	t.Setenv("AJ_JIRA_API_TOKEN", "secret")

	_, err := ResolveJiraSettings(repo)
	if err == nil {
		t.Fatalf("expected missing base url error")
	}
	if !strings.Contains(err.Error(), "jira base URL is required.") || !strings.Contains(err.Error(), "AJ_JIRA_BASE_URL") {
		t.Fatalf("unexpected base url guidance: %v", err)
	}
}

func TestResolveJiraSettingsGuidesMissingCredentials(t *testing.T) {
	repo := t.TempDir()
	writeTestConfig(t, repo, `schema_version = 1
default_output = "brief"
default_lease_ttl = "4h"

[jira]
enabled = true
base_url = "https://example.atlassian.net"
project = "ABC"
`)

	_, err := ResolveJiraSettings(repo)
	if err == nil {
		t.Fatalf("expected missing credentials error")
	}
	if !strings.Contains(err.Error(), "AJ_JIRA_EMAIL") || !strings.Contains(err.Error(), "AJ_JIRA_API_TOKEN") {
		t.Fatalf("unexpected credential guidance: %v", err)
	}
}

func writeTestConfig(t *testing.T, repo, raw string) {
	t.Helper()
	ajDir := filepath.Join(repo, ".aj")
	if err := os.MkdirAll(ajDir, 0o755); err != nil {
		t.Fatalf("mkdir .aj: %v", err)
	}
	if err := os.WriteFile(filepath.Join(ajDir, "config.toml"), []byte(raw), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
}
