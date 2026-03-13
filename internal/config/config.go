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
		return JiraSettings{}, errors.New("jira is not enabled; set [jira].enabled = true in .aj/config.toml or AJ_JIRA_ENABLED=true")
	}
	if settings.BaseURL == "" {
		return JiraSettings{}, errors.New("jira base URL is required; set [jira].base_url in .aj/config.toml or AJ_JIRA_BASE_URL")
	}
	if settings.Email == "" {
		return JiraSettings{}, errors.New("jira email is required; set AJ_JIRA_EMAIL")
	}
	if settings.APIToken == "" {
		return JiraSettings{}, errors.New("jira API token is required; set AJ_JIRA_API_TOKEN")
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
