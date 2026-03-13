package jira

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestClientGetAndCreateIssue(t *testing.T) {
	client := Client{
		BaseURL:  "https://example.atlassian.net",
		Email:    "agent@example.com",
		APIToken: "secret",
		HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("agent@example.com:secret"))
			if r.Header.Get("Authorization") != wantAuth {
				return jsonResponse(http.StatusUnauthorized, "unauthorized"), nil
			}

			switch {
			case r.Method == http.MethodPost && r.URL.Path == "/rest/api/3/search/jql":
				var payload map[string]any
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Fatalf("decode search payload: %v", err)
				}
				if payload["jql"] != `project = "ABC" AND text ~ "cache" ORDER BY updated DESC` {
					t.Fatalf("unexpected search jql payload: %#v", payload["jql"])
				}
				if payload["maxResults"] != float64(5) {
					t.Fatalf("unexpected search maxResults payload: %#v", payload["maxResults"])
				}
				response, _ := json.Marshal(map[string]any{
					"issues": []any{
						map[string]any{
							"key":  "ABC-789",
							"self": "https://example.atlassian.net/rest/api/3/issue/10002",
							"fields": map[string]any{
								"summary":   "Cache issue",
								"issuetype": map[string]any{"name": "Bug"},
								"priority":  map[string]any{"name": "High"},
								"status":    map[string]any{"name": "To Do"},
								"updated":   "2026-03-13T13:00:00.000+0000",
							},
						},
					},
				})
				return jsonResponse(http.StatusOK, string(response)), nil
			case r.Method == http.MethodGet && r.URL.Path == "/rest/api/3/issue/ABC-456/transitions":
				response, _ := json.Marshal(map[string]any{
					"transitions": []any{
						map[string]any{
							"id":   "31",
							"name": "Done",
							"to":   map[string]any{"name": "Done"},
						},
					},
				})
				return jsonResponse(http.StatusOK, string(response)), nil
			case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/rest/api/3/issue/ABC-123"):
				payload, _ := json.Marshal(map[string]any{
					"key":  "ABC-123",
					"self": "https://example.atlassian.net/rest/api/3/issue/10000",
					"fields": map[string]any{
						"summary": "Imported Jira bug",
						"description": map[string]any{
							"type":    "doc",
							"version": 1,
							"content": []any{
								map[string]any{
									"type": "paragraph",
									"content": []any{
										map[string]any{"type": "text", "text": "Fix the import path."},
									},
								},
							},
						},
						"issuetype": map[string]any{"name": "Bug"},
						"priority":  map[string]any{"name": "High"},
						"status":    map[string]any{"name": "In Progress"},
						"updated":   "2026-03-13T12:00:00.000+0000",
					},
				})
				return jsonResponse(http.StatusOK, string(payload)), nil
			case r.Method == http.MethodPost && r.URL.Path == "/rest/api/3/issue":
				var payload map[string]any
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Fatalf("decode create payload: %v", err)
				}
				fields := payload["fields"].(map[string]any)
				if fields["summary"] != "Exported task" {
					t.Fatalf("unexpected summary payload: %#v", fields["summary"])
				}
				response, _ := json.Marshal(map[string]any{
					"key":  "ABC-456",
					"self": "https://example.atlassian.net/rest/api/3/issue/10001",
				})
				return jsonResponse(http.StatusCreated, string(response)), nil
			case r.Method == http.MethodPut && r.URL.Path == "/rest/api/3/issue/ABC-456":
				var payload map[string]any
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Fatalf("decode update payload: %v", err)
				}
				fields := payload["fields"].(map[string]any)
				if fields["summary"] != "Updated task" {
					t.Fatalf("unexpected update summary payload: %#v", fields["summary"])
				}
				return jsonResponse(http.StatusNoContent, ""), nil
			case r.Method == http.MethodPost && r.URL.Path == "/rest/api/3/issue/ABC-456/transitions":
				return jsonResponse(http.StatusNoContent, ""), nil
			case r.Method == http.MethodPost && r.URL.Path == "/rest/api/3/issue/ABC-456/comment":
				var payload map[string]any
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Fatalf("decode comment payload: %v", err)
				}
				return jsonResponse(http.StatusCreated, `{"id":"10001"}`), nil
			default:
				return jsonResponse(http.StatusNotFound, "not found"), nil
			}
		})},
	}

	issue, err := client.GetIssue(context.Background(), "ABC-123")
	if err != nil {
		t.Fatalf("get issue: %v", err)
	}
	if issue.Key != "ABC-123" || issue.Description != "Fix the import path." {
		t.Fatalf("unexpected issue: %#v", issue)
	}

	issues, err := client.SearchIssues(context.Background(), `project = "ABC" AND text ~ "cache" ORDER BY updated DESC`, 5)
	if err != nil {
		t.Fatalf("search issues: %v", err)
	}
	if len(issues) != 1 || issues[0].Key != "ABC-789" {
		t.Fatalf("unexpected search issues: %#v", issues)
	}

	created, err := client.CreateIssue(context.Background(), CreateIssueInput{
		ProjectKey:  "ABC",
		IssueType:   "Task",
		Summary:     "Exported task",
		Description: "Goal: ship the export flow",
	})
	if err != nil {
		t.Fatalf("create issue: %v", err)
	}
	if created.Key != "ABC-456" {
		t.Fatalf("unexpected created issue: %#v", created)
	}

	if err := client.UpdateIssue(context.Background(), "ABC-456", UpdateIssueInput{
		Summary:     "Updated task",
		Description: "Updated description",
	}); err != nil {
		t.Fatalf("update issue: %v", err)
	}

	transitions, err := client.GetTransitions(context.Background(), "ABC-456")
	if err != nil {
		t.Fatalf("get transitions: %v", err)
	}
	if len(transitions) != 1 || transitions[0].ID != "31" {
		t.Fatalf("unexpected transitions: %#v", transitions)
	}

	if err := client.TransitionIssue(context.Background(), "ABC-456", "31"); err != nil {
		t.Fatalf("transition issue: %v", err)
	}

	if err := client.AddComment(context.Background(), "ABC-456", "Ready for review"); err != nil {
		t.Fatalf("add comment: %v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
