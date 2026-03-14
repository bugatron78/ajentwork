package store

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"ajentwork/internal/config"
	"ajentwork/internal/jira"
)

type JiraSpaceExistsOptions struct {
	RepoPath string
	Key      string
}

type JiraSpaceCreateOptions struct {
	RepoPath string
	Key      string
	Name     string
	Type     string
	Template string
}

type JiraSpaceEnsureOptions struct {
	RepoPath string
	Key      string
	Name     string
	Type     string
	Template string
}

type JiraSpaceListOptions struct {
	RepoPath string
	Query    string
	Limit    int
}

type JiraSpaceExistsResult struct {
	Key    string       `json:"key"`
	Exists bool         `json:"exists"`
	Space  jira.Project `json:"space"`
}

type JiraSpaceCreateResult struct {
	Created bool         `json:"created"`
	Space   jira.Project `json:"space"`
}

type JiraSpaceListResult struct {
	Query  string         `json:"query,omitempty"`
	Limit  int            `json:"limit"`
	Spaces []jira.Project `json:"spaces"`
}

func JiraSpaceExists(opts JiraSpaceExistsOptions) (JiraSpaceExistsResult, error) {
	settings, err := config.ResolveJiraSettings(opts.RepoPath)
	if err != nil {
		return JiraSpaceExistsResult{}, err
	}
	key := resolveSpaceKey(strings.TrimSpace(opts.Key), settings.Project)
	if key == "" {
		return JiraSpaceExistsResult{}, errors.New("jira space key is required; pass --key or set [jira].project")
	}

	client := jiraClientFromSettings(settings)
	space, exists, err := getProjectByKey(context.Background(), client, key)
	if err != nil {
		return JiraSpaceExistsResult{}, err
	}
	return JiraSpaceExistsResult{
		Key:    key,
		Exists: exists,
		Space:  space,
	}, nil
}

func CreateJiraSpace(opts JiraSpaceCreateOptions) (JiraSpaceCreateResult, error) {
	settings, err := config.ResolveJiraSettings(opts.RepoPath)
	if err != nil {
		return JiraSpaceCreateResult{}, err
	}
	key := resolveSpaceKey(strings.TrimSpace(opts.Key), settings.Project)
	if key == "" {
		return JiraSpaceCreateResult{}, errors.New("jira space key is required; pass --key or set [jira].project")
	}
	name := strings.TrimSpace(opts.Name)
	if name == "" {
		return JiraSpaceCreateResult{}, errors.New("jira space name is required")
	}

	projectType := normalizeProjectType(opts.Type)
	templateKey, err := resolveTemplateKey(projectType, strings.TrimSpace(opts.Template))
	if err != nil {
		return JiraSpaceCreateResult{}, err
	}

	client := jiraClientFromSettings(settings)
	if _, exists, err := getProjectByKey(context.Background(), client, key); err != nil {
		return JiraSpaceCreateResult{}, err
	} else if exists {
		return JiraSpaceCreateResult{}, fmt.Errorf("jira space %s already exists", key)
	}

	user, err := client.GetMyself(context.Background())
	if err != nil {
		return JiraSpaceCreateResult{}, err
	}
	if user.AccountID == "" {
		return JiraSpaceCreateResult{}, errors.New("jira current user account id is required to create a space")
	}

	space, err := client.CreateProject(context.Background(), jira.CreateProjectInput{
		Key:                key,
		Name:               name,
		ProjectTypeKey:     projectType,
		ProjectTemplateKey: templateKey,
		LeadAccountID:      user.AccountID,
	})
	if err != nil {
		return JiraSpaceCreateResult{}, err
	}
	return JiraSpaceCreateResult{
		Created: true,
		Space:   space,
	}, nil
}

func EnsureJiraSpace(opts JiraSpaceEnsureOptions) (JiraSpaceCreateResult, error) {
	settings, err := config.ResolveJiraSettings(opts.RepoPath)
	if err != nil {
		return JiraSpaceCreateResult{}, err
	}
	key := resolveSpaceKey(strings.TrimSpace(opts.Key), settings.Project)
	if key == "" {
		return JiraSpaceCreateResult{}, errors.New("jira space key is required; pass --key or set [jira].project")
	}

	client := jiraClientFromSettings(settings)
	if space, exists, err := getProjectByKey(context.Background(), client, key); err != nil {
		return JiraSpaceCreateResult{}, err
	} else if exists {
		return JiraSpaceCreateResult{
			Created: false,
			Space:   space,
		}, nil
	}

	return CreateJiraSpace(JiraSpaceCreateOptions{
		RepoPath: opts.RepoPath,
		Key:      key,
		Name:     opts.Name,
		Type:     opts.Type,
		Template: opts.Template,
	})
}

func ListJiraSpaces(opts JiraSpaceListOptions) (JiraSpaceListResult, error) {
	settings, err := config.ResolveJiraSettings(opts.RepoPath)
	if err != nil {
		return JiraSpaceListResult{}, err
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = 20
	}
	client := jiraClientFromSettings(settings)
	spaces, err := client.SearchProjects(context.Background(), strings.TrimSpace(opts.Query), limit)
	if err != nil {
		return JiraSpaceListResult{}, err
	}
	return JiraSpaceListResult{
		Query:  strings.TrimSpace(opts.Query),
		Limit:  limit,
		Spaces: spaces,
	}, nil
}

func jiraClientFromSettings(settings config.JiraSettings) jira.Client {
	return jira.Client{
		BaseURL:    settings.BaseURL,
		Email:      settings.Email,
		APIToken:   settings.APIToken,
		HTTPClient: jira.DefaultHTTPClient,
	}
}

func getProjectByKey(ctx context.Context, client jira.Client, key string) (jira.Project, bool, error) {
	project, err := client.GetProject(ctx, key)
	if err == nil {
		return project, true, nil
	}
	var apiErr *jira.APIError
	if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
		return jira.Project{}, false, nil
	}
	return jira.Project{}, false, err
}

func resolveSpaceKey(requestedKey, configuredKey string) string {
	if requestedKey != "" {
		return strings.TrimSpace(requestedKey)
	}
	return strings.TrimSpace(configuredKey)
}

func normalizeProjectType(projectType string) string {
	projectType = strings.TrimSpace(projectType)
	if projectType == "" {
		return "software"
	}
	return projectType
}

func resolveTemplateKey(projectType, requestedTemplate string) (string, error) {
	if strings.TrimSpace(requestedTemplate) != "" {
		return strings.TrimSpace(requestedTemplate), nil
	}
	switch projectType {
	case "software":
		return "com.pyxis.greenhopper.jira:gh-simplified-agility-scrum", nil
	default:
		return "", fmt.Errorf("template is required for jira space type %q; pass --template explicitly", projectType)
	}
}
