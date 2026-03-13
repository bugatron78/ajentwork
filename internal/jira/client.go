package jira

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

var DefaultHTTPClient = http.DefaultClient

type Client struct {
	BaseURL    string
	Email      string
	APIToken   string
	HTTPClient *http.Client
}

type Issue struct {
	Key         string
	URL         string
	Summary     string
	Description string
	IssueType   string
	Priority    string
	Status      string
	Updated     string
}

type CreateIssueInput struct {
	ProjectKey  string
	IssueType   string
	Summary     string
	Description string
}

type UpdateIssueInput struct {
	Summary     string
	Description string
}

type Transition struct {
	ID   string
	Name string
	To   string
}

func (c Client) SearchIssues(ctx context.Context, jql string, limit int) ([]Issue, error) {
	if strings.TrimSpace(jql) == "" {
		return nil, fmt.Errorf("jira search JQL is required")
	}
	if limit <= 0 {
		limit = 10
	}

	body := map[string]any{
		"jql":        strings.TrimSpace(jql),
		"maxResults": limit,
		"fields":     []string{"summary", "description", "issuetype", "priority", "status", "updated"},
	}
	var payload struct {
		Issues []struct {
			Key    string `json:"key"`
			Self   string `json:"self"`
			Fields struct {
				Summary     string          `json:"summary"`
				Description json.RawMessage `json:"description"`
				IssueType   struct {
					Name string `json:"name"`
				} `json:"issuetype"`
				Priority *struct {
					Name string `json:"name"`
				} `json:"priority"`
				Status struct {
					Name string `json:"name"`
				} `json:"status"`
				Updated string `json:"updated"`
			} `json:"fields"`
		} `json:"issues"`
	}
	if err := c.doJSON(ctx, http.MethodPost, "/rest/api/3/search/jql", body, &payload); err != nil {
		return nil, err
	}

	issues := make([]Issue, 0, len(payload.Issues))
	for _, issue := range payload.Issues {
		priority := ""
		if issue.Fields.Priority != nil {
			priority = issue.Fields.Priority.Name
		}
		issues = append(issues, Issue{
			Key:         issue.Key,
			URL:         strings.TrimRight(c.BaseURL, "/") + "/browse/" + issue.Key,
			Summary:     strings.TrimSpace(issue.Fields.Summary),
			Description: strings.TrimSpace(extractADFText(issue.Fields.Description)),
			IssueType:   strings.TrimSpace(issue.Fields.IssueType.Name),
			Priority:    strings.TrimSpace(priority),
			Status:      strings.TrimSpace(issue.Fields.Status.Name),
			Updated:     strings.TrimSpace(issue.Fields.Updated),
		})
	}
	return issues, nil
}

func (c Client) GetIssue(ctx context.Context, issueKey string) (Issue, error) {
	path := fmt.Sprintf("/rest/api/3/issue/%s?fields=summary,description,issuetype,priority,status,updated", url.PathEscape(strings.TrimSpace(issueKey)))
	var payload struct {
		Key    string `json:"key"`
		Self   string `json:"self"`
		Fields struct {
			Summary     string          `json:"summary"`
			Description json.RawMessage `json:"description"`
			IssueType   struct {
				Name string `json:"name"`
			} `json:"issuetype"`
			Priority *struct {
				Name string `json:"name"`
			} `json:"priority"`
			Status struct {
				Name string `json:"name"`
			} `json:"status"`
			Updated string `json:"updated"`
		} `json:"fields"`
	}

	if err := c.doJSON(ctx, http.MethodGet, path, nil, &payload); err != nil {
		return Issue{}, err
	}

	description := extractADFText(payload.Fields.Description)
	priority := ""
	if payload.Fields.Priority != nil {
		priority = payload.Fields.Priority.Name
	}
	return Issue{
		Key:         payload.Key,
		URL:         strings.TrimRight(c.BaseURL, "/") + "/browse/" + payload.Key,
		Summary:     strings.TrimSpace(payload.Fields.Summary),
		Description: strings.TrimSpace(description),
		IssueType:   strings.TrimSpace(payload.Fields.IssueType.Name),
		Priority:    strings.TrimSpace(priority),
		Status:      strings.TrimSpace(payload.Fields.Status.Name),
		Updated:     strings.TrimSpace(payload.Fields.Updated),
	}, nil
}

func (c Client) CreateIssue(ctx context.Context, input CreateIssueInput) (Issue, error) {
	body := map[string]any{
		"fields": map[string]any{
			"project": map[string]any{
				"key": input.ProjectKey,
			},
			"summary": input.Summary,
			"issuetype": map[string]any{
				"name": input.IssueType,
			},
		},
	}
	if strings.TrimSpace(input.Description) != "" {
		body["fields"].(map[string]any)["description"] = adfDocument(input.Description)
	}

	var payload struct {
		Key  string `json:"key"`
		Self string `json:"self"`
	}
	if err := c.doJSON(ctx, http.MethodPost, "/rest/api/3/issue", body, &payload); err != nil {
		return Issue{}, err
	}
	return Issue{
		Key:         payload.Key,
		URL:         strings.TrimRight(c.BaseURL, "/") + "/browse/" + payload.Key,
		Summary:     strings.TrimSpace(input.Summary),
		Description: strings.TrimSpace(input.Description),
		IssueType:   strings.TrimSpace(input.IssueType),
	}, nil
}

func (c Client) UpdateIssue(ctx context.Context, issueKey string, input UpdateIssueInput) error {
	body := map[string]any{
		"fields": map[string]any{
			"summary": input.Summary,
		},
	}
	if strings.TrimSpace(input.Description) != "" {
		body["fields"].(map[string]any)["description"] = adfDocument(input.Description)
	}

	path := fmt.Sprintf("/rest/api/3/issue/%s", url.PathEscape(strings.TrimSpace(issueKey)))
	return c.doJSON(ctx, http.MethodPut, path, body, nil)
}

func (c Client) GetTransitions(ctx context.Context, issueKey string) ([]Transition, error) {
	path := fmt.Sprintf("/rest/api/3/issue/%s/transitions", url.PathEscape(strings.TrimSpace(issueKey)))
	var payload struct {
		Transitions []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			To   struct {
				Name string `json:"name"`
			} `json:"to"`
		} `json:"transitions"`
	}
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &payload); err != nil {
		return nil, err
	}
	result := make([]Transition, 0, len(payload.Transitions))
	for _, transition := range payload.Transitions {
		result = append(result, Transition{
			ID:   strings.TrimSpace(transition.ID),
			Name: strings.TrimSpace(transition.Name),
			To:   strings.TrimSpace(transition.To.Name),
		})
	}
	return result, nil
}

func (c Client) TransitionIssue(ctx context.Context, issueKey, transitionID string) error {
	path := fmt.Sprintf("/rest/api/3/issue/%s/transitions", url.PathEscape(strings.TrimSpace(issueKey)))
	body := map[string]any{
		"transition": map[string]any{
			"id": transitionID,
		},
	}
	return c.doJSON(ctx, http.MethodPost, path, body, nil)
}

func (c Client) AddComment(ctx context.Context, issueKey, bodyText string) error {
	path := fmt.Sprintf("/rest/api/3/issue/%s/comment", url.PathEscape(strings.TrimSpace(issueKey)))
	body := map[string]any{
		"body": adfDocument(bodyText),
	}
	return c.doJSON(ctx, http.MethodPost, path, body, nil)
}

func (c Client) doJSON(ctx context.Context, method, path string, body any, out any) error {
	if strings.TrimSpace(c.BaseURL) == "" {
		return fmt.Errorf("jira base URL is required")
	}
	if strings.TrimSpace(c.Email) == "" || strings.TrimSpace(c.APIToken) == "" {
		return fmt.Errorf("jira credentials are required")
	}

	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal jira request: %w", err)
		}
		reader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(c.BaseURL, "/")+path, reader)
	if err != nil {
		return fmt.Errorf("create jira request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(c.Email+":"+c.APIToken)))

	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = DefaultHTTPClient
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send jira request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("jira API %s %s failed: %s: %s", method, path, resp.Status, strings.TrimSpace(string(bodyBytes)))
	}

	if out == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode jira response: %w", err)
	}
	return nil
}

func adfDocument(text string) map[string]any {
	paragraphs := make([]any, 0)
	for _, block := range strings.Split(strings.TrimSpace(text), "\n\n") {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		paragraphs = append(paragraphs, map[string]any{
			"type": "paragraph",
			"content": []any{
				map[string]any{
					"type": "text",
					"text": block,
				},
			},
		})
	}
	if len(paragraphs) == 0 {
		paragraphs = append(paragraphs, map[string]any{"type": "paragraph"})
	}
	return map[string]any{
		"type":    "doc",
		"version": 1,
		"content": paragraphs,
	}
}

func extractADFText(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var node map[string]any
	if err := json.Unmarshal(raw, &node); err != nil {
		return ""
	}
	var parts []string
	var walk func(any)
	walk = func(value any) {
		switch typed := value.(type) {
		case map[string]any:
			if text, ok := typed["text"].(string); ok {
				parts = append(parts, text)
			}
			if content, ok := typed["content"].([]any); ok {
				for _, child := range content {
					walk(child)
				}
				if nodeType, _ := typed["type"].(string); nodeType == "paragraph" {
					parts = append(parts, "\n")
				}
			}
		case []any:
			for _, child := range typed {
				walk(child)
			}
		}
	}
	walk(node)
	return strings.TrimSpace(strings.Join(parts, ""))
}
