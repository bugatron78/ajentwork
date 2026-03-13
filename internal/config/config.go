package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"ajentwork/internal/domain"
)

type Config struct {
	SchemaVersion   int
	DefaultOutput   string
	DefaultLeaseTTL string
	Jira            JiraConfig
}

type JiraConfig struct {
	Enabled   bool
	BaseURL   string
	Project   string
	StatusMap map[string]domain.Status
	Lifecycle JiraLifecycleConfig
}

type JiraLifecycleConfig struct {
	CommentOnDone    bool
	CommentOnBlock   bool
	CommentOnHandoff bool
}

type JiraSettings struct {
	Enabled   bool
	BaseURL   string
	Project   string
	Email     string
	APIToken  string
	StatusMap map[string]domain.Status
}

func Load(repoPath string) (Config, error) {
	configPath := filepath.Join(filepath.Clean(repoPath), ".aj", "config.toml")
	raw, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{}, fmt.Errorf("aj is not initialized in %s (run `aj init` first)", filepath.Clean(repoPath))
		}
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	cfg := Config{
		Jira: JiraConfig{
			StatusMap: map[string]domain.Status{},
		},
	}
	section := ""

	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "["), "]"))
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return Config{}, fmt.Errorf("invalid config line %q", line)
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch section {
		case "":
			switch key {
			case "schema_version":
				parsed, err := strconv.Atoi(value)
				if err != nil {
					return Config{}, fmt.Errorf("invalid schema_version: %w", err)
				}
				cfg.SchemaVersion = parsed
			case "default_output":
				cfg.DefaultOutput, err = parseQuoted(value)
				if err != nil {
					return Config{}, fmt.Errorf("parse default_output: %w", err)
				}
			case "default_lease_ttl":
				cfg.DefaultLeaseTTL, err = parseQuoted(value)
				if err != nil {
					return Config{}, fmt.Errorf("parse default_lease_ttl: %w", err)
				}
			}
		case "jira":
			switch key {
			case "enabled":
				parsed, err := strconv.ParseBool(value)
				if err != nil {
					return Config{}, fmt.Errorf("parse jira.enabled: %w", err)
				}
				cfg.Jira.Enabled = parsed
			case "base_url":
				cfg.Jira.BaseURL, err = parseQuoted(value)
				if err != nil {
					return Config{}, fmt.Errorf("parse jira.base_url: %w", err)
				}
			case "project":
				cfg.Jira.Project, err = parseQuoted(value)
				if err != nil {
					return Config{}, fmt.Errorf("parse jira.project: %w", err)
				}
			}
		case "jira.status_map":
			name, err := parseQuoted(key)
			if err != nil {
				return Config{}, fmt.Errorf("parse jira.status_map key: %w", err)
			}
			statusRaw, err := parseQuoted(value)
			if err != nil {
				return Config{}, fmt.Errorf("parse jira.status_map value: %w", err)
			}
			status, err := domain.ParseStatus(statusRaw)
			if err != nil {
				return Config{}, fmt.Errorf("parse jira.status_map status: %w", err)
			}
			cfg.Jira.StatusMap[name] = status
		case "jira.lifecycle":
			parsed, err := strconv.ParseBool(value)
			if err != nil {
				return Config{}, fmt.Errorf("parse %s: %w", key, err)
			}
			switch key {
			case "comment_on_done":
				cfg.Jira.Lifecycle.CommentOnDone = parsed
			case "comment_on_block":
				cfg.Jira.Lifecycle.CommentOnBlock = parsed
			case "comment_on_handoff":
				cfg.Jira.Lifecycle.CommentOnHandoff = parsed
			}
		}
	}

	return cfg, nil
}

func ResolveJiraSettings(repoPath string) (JiraSettings, error) {
	cfg, err := Load(repoPath)
	if err != nil {
		return JiraSettings{}, err
	}

	settings := JiraSettings{
		Enabled:   cfg.Jira.Enabled,
		BaseURL:   strings.TrimSpace(cfg.Jira.BaseURL),
		Project:   strings.TrimSpace(cfg.Jira.Project),
		Email:     strings.TrimSpace(os.Getenv("AJ_JIRA_EMAIL")),
		APIToken:  strings.TrimSpace(os.Getenv("AJ_JIRA_API_TOKEN")),
		StatusMap: cfg.Jira.StatusMap,
	}

	if value := strings.TrimSpace(os.Getenv("AJ_JIRA_ENABLED")); value != "" {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return JiraSettings{}, fmt.Errorf("parse AJ_JIRA_ENABLED: %w", err)
		}
		settings.Enabled = parsed
	}
	if value := strings.TrimSpace(os.Getenv("AJ_JIRA_BASE_URL")); value != "" {
		settings.BaseURL = value
	}
	if value := strings.TrimSpace(os.Getenv("AJ_JIRA_PROJECT")); value != "" {
		settings.Project = value
	}

	if !settings.Enabled {
		return JiraSettings{}, errors.New(jiraDisabledMessage(repoPath, settings))
	}
	if settings.BaseURL == "" {
		return JiraSettings{}, errors.New(jiraBaseURLMessage(repoPath))
	}
	if settings.Email == "" {
		return JiraSettings{}, errors.New(jiraCredentialsMessage("AJ_JIRA_EMAIL"))
	}
	if settings.APIToken == "" {
		return JiraSettings{}, errors.New(jiraCredentialsMessage("AJ_JIRA_API_TOKEN"))
	}
	return settings, nil
}

func parseQuoted(value string) (string, error) {
	unquoted, err := strconv.Unquote(value)
	if err != nil {
		return "", err
	}
	return unquoted, nil
}

func jiraDisabledMessage(repoPath string, settings JiraSettings) string {
	configPath := filepath.Join(filepath.Clean(repoPath), ".aj", "config.toml")
	baseURL := settings.BaseURL
	if baseURL == "" {
		baseURL = "https://your-domain.atlassian.net"
	}
	project := settings.Project
	if project == "" {
		project = "ABC"
	}
	return strings.Join([]string{
		"jira is not enabled for this repo.",
		fmt.Sprintf("Add this to %s:", configPath),
		"[jira]",
		`enabled = true`,
		fmt.Sprintf(`base_url = %q`, baseURL),
		fmt.Sprintf(`project = %q`, project),
		"Or set AJ_JIRA_ENABLED=true temporarily.",
		"See: aj help jira",
	}, "\n")
}

func jiraBaseURLMessage(repoPath string) string {
	configPath := filepath.Join(filepath.Clean(repoPath), ".aj", "config.toml")
	return strings.Join([]string{
		"jira base URL is required.",
		fmt.Sprintf("Set [jira].base_url in %s or set AJ_JIRA_BASE_URL.", configPath),
		`Example: base_url = "https://your-domain.atlassian.net"`,
		"See: aj help jira",
	}, "\n")
}

func jiraCredentialsMessage(missing string) string {
	return strings.Join([]string{
		fmt.Sprintf("jira credential %s is required.", missing),
		"Set both AJ_JIRA_EMAIL and AJ_JIRA_API_TOKEN in your shell environment.",
		`Example: export AJ_JIRA_EMAIL="you@example.com"`,
		`Example: export AJ_JIRA_API_TOKEN="..."`,
		"See: aj help jira",
	}, "\n")
}
