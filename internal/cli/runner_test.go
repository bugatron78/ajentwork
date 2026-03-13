package cli

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ajentwork/internal/jira"
)

func TestRunnerInitNewListShow(t *testing.T) {
	repo := t.TempDir()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	runner := NewRunner(stdout, stderr)

	if code := runner.Run([]string{"--repo", repo, "init"}); code != 0 {
		t.Fatalf("init exit code = %d, stderr = %s", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := runner.Run([]string{"--repo", repo, "new", "--kind", "task", "--title", "Bootstrap CLI", "--goal", "Ship the first command set", "--next", "Implement item storage"}); code != 0 {
		t.Fatalf("new exit code = %d, stderr = %s", code, stderr.String())
	}

	if !strings.Contains(stdout.String(), "created W-") {
		t.Fatalf("unexpected new output: %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := runner.Run([]string{"--repo", repo, "ls"}); code != 0 {
		t.Fatalf("ls exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Bootstrap CLI") {
		t.Fatalf("expected list output to contain item title, got %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := runner.Run([]string{"--repo", repo, "ls", "--format", "json"}); code != 0 {
		t.Fatalf("ls json exit code = %d, stderr = %s", code, stderr.String())
	}

	var items []map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &items); err != nil {
		t.Fatalf("unmarshal list json: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one listed item, got %d", len(items))
	}

	itemID, ok := items[0]["id"].(string)
	if !ok || itemID == "" {
		t.Fatalf("expected json list to include item id, got %#v", items[0]["id"])
	}

	stdout.Reset()
	stderr.Reset()
	if code := runner.Run([]string{"--repo", repo, "show", itemID}); code != 0 {
		t.Fatalf("show exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Goal: Ship the first command set") {
		t.Fatalf("unexpected show output: %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := runner.Run([]string{"--repo", repo, "update", itemID, "--summary", "implementation started", "--status", "in_progress", "--next", "Write tests"}); code != 0 {
		t.Fatalf("update exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "updated "+itemID) {
		t.Fatalf("unexpected update output: %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := runner.Run([]string{"--repo", repo, "done", itemID, "--summary", "feature shipped"}); code != 0 {
		t.Fatalf("done exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "done "+itemID) {
		t.Fatalf("unexpected done output: %s", stdout.String())
	}
}

func TestRunnerHelpSearch(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	runner := NewRunner(stdout, stderr)

	if code := runner.Run([]string{"help", "search", "ticket"}); code != 0 {
		t.Fatalf("help search exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "new") {
		t.Fatalf("expected help search to find new command, got %s", stdout.String())
	}
}

func TestRunnerTakeAndRelease(t *testing.T) {
	repo := t.TempDir()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	runner := NewRunner(stdout, stderr)

	if code := runner.Run([]string{"--repo", repo, "init"}); code != 0 {
		t.Fatalf("init exit code = %d, stderr = %s", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := runner.Run([]string{"--repo", repo, "new", "--kind", "task", "--title", "Claimable work", "--goal", "Test take and release", "--next", "Claim the task"}); code != 0 {
		t.Fatalf("new exit code = %d, stderr = %s", code, stderr.String())
	}

	parts := strings.Fields(stdout.String())
	if len(parts) < 2 {
		t.Fatalf("unexpected new output: %s", stdout.String())
	}
	itemID := parts[1]

	stdout.Reset()
	stderr.Reset()
	if code := runner.Run([]string{"--repo", repo, "take", itemID, "--agent", "coder-1", "--ttl", "2h"}); code != 0 {
		t.Fatalf("take exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "claimed "+itemID) {
		t.Fatalf("unexpected take output: %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := runner.Run([]string{"--repo", repo, "release", itemID}); code != 0 {
		t.Fatalf("release exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "released "+itemID) {
		t.Fatalf("unexpected release output: %s", stdout.String())
	}
}

func TestRunnerBlockUnblockHandoffAndReopen(t *testing.T) {
	repo := t.TempDir()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	runner := NewRunner(stdout, stderr)

	if code := runner.Run([]string{"--repo", repo, "init"}); code != 0 {
		t.Fatalf("init exit code = %d, stderr = %s", code, stderr.String())
	}

	if code := runner.Run([]string{"--repo", repo, "new", "--kind", "task", "--title", "Dependency", "--goal", "Unblock downstream work", "--next", "Ship dependency"}); code != 0 {
		t.Fatalf("new dependency exit code = %d, stderr = %s", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := runner.Run([]string{"--repo", repo, "new", "--kind", "task", "--title", "Coordinated work", "--goal", "Exercise coordination commands", "--next", "Start work"}); code != 0 {
		t.Fatalf("new item exit code = %d, stderr = %s", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := runner.Run([]string{"--repo", repo, "ls", "--format", "json"}); code != 0 {
		t.Fatalf("ls json exit code = %d, stderr = %s", code, stderr.String())
	}
	var items []map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &items); err != nil {
		t.Fatalf("unmarshal list json: %v", err)
	}

	var dependencyID, itemID string
	for _, item := range items {
		title, _ := item["title"].(string)
		id, _ := item["id"].(string)
		switch title {
		case "Dependency":
			dependencyID = id
		case "Coordinated work":
			itemID = id
		}
	}
	if dependencyID == "" || itemID == "" {
		t.Fatalf("expected both item ids, got dependency=%q item=%q", dependencyID, itemID)
	}

	stdout.Reset()
	stderr.Reset()
	if code := runner.Run([]string{"--repo", repo, "block", itemID, "--on", dependencyID, "--summary", "waiting on dependency"}); code != 0 {
		t.Fatalf("block exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "blocked "+itemID) {
		t.Fatalf("unexpected block output: %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := runner.Run([]string{"--repo", repo, "unblock", itemID, "--summary", "dependency shipped", "--status", "in_progress", "--next", "Resume implementation"}); code != 0 {
		t.Fatalf("unblock exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "unblocked "+itemID) {
		t.Fatalf("unexpected unblock output: %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := runner.Run([]string{"--repo", repo, "handoff", itemID, "--to", "reviewer-1", "--summary", "implementation ready for review"}); code != 0 {
		t.Fatalf("handoff exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "handed off "+itemID+" to reviewer-1") {
		t.Fatalf("unexpected handoff output: %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := runner.Run([]string{"--repo", repo, "done", itemID, "--summary", "review completed"}); code != 0 {
		t.Fatalf("done exit code = %d, stderr = %s", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := runner.Run([]string{"--repo", repo, "reopen", itemID, "--summary", "regression found", "--next", "Add a failing test"}); code != 0 {
		t.Fatalf("reopen exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "reopened "+itemID) {
		t.Fatalf("unexpected reopen output: %s", stdout.String())
	}
}

func TestRunnerNextAndInbox(t *testing.T) {
	repo := t.TempDir()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	runner := NewRunner(stdout, stderr)

	if code := runner.Run([]string{"--repo", repo, "init"}); code != 0 {
		t.Fatalf("init exit code = %d, stderr = %s", code, stderr.String())
	}
	if code := runner.Run([]string{"--repo", repo, "new", "--kind", "task", "--title", "Owned work", "--goal", "Track owned work", "--next", "Continue implementation"}); code != 0 {
		t.Fatalf("new owned exit code = %d, stderr = %s", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := runner.Run([]string{"--repo", repo, "ls", "--format", "json"}); code != 0 {
		t.Fatalf("ls json exit code = %d, stderr = %s", code, stderr.String())
	}
	var items []map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &items); err != nil {
		t.Fatalf("unmarshal list json: %v", err)
	}
	itemID := items[0]["id"].(string)

	stdout.Reset()
	stderr.Reset()
	if code := runner.Run([]string{"--repo", repo, "take", itemID, "--agent", "coder-1"}); code != 0 {
		t.Fatalf("take exit code = %d, stderr = %s", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := runner.Run([]string{"--repo", repo, "next", "--agent", "coder-1"}); code != 0 {
		t.Fatalf("next exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), itemID) || !strings.Contains(stdout.String(), "currently leased to coder-1") {
		t.Fatalf("unexpected next output: %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := runner.Run([]string{"--repo", repo, "inbox", "--agent", "coder-1"}); code != 0 {
		t.Fatalf("inbox exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "owned") {
		t.Fatalf("unexpected inbox output: %s", stdout.String())
	}
}

func TestRunnerLinkShowsDependency(t *testing.T) {
	repo := t.TempDir()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	runner := NewRunner(stdout, stderr)

	if code := runner.Run([]string{"--repo", repo, "init"}); code != 0 {
		t.Fatalf("init exit code = %d, stderr = %s", code, stderr.String())
	}
	if code := runner.Run([]string{"--repo", repo, "new", "--kind", "task", "--title", "Parent", "--goal", "Build parent", "--next", "Finish parent"}); code != 0 {
		t.Fatalf("new parent exit code = %d, stderr = %s", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := runner.Run([]string{"--repo", repo, "new", "--kind", "task", "--title", "Child", "--goal", "Build child", "--next", "Wait for parent"}); code != 0 {
		t.Fatalf("new child exit code = %d, stderr = %s", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := runner.Run([]string{"--repo", repo, "ls", "--format", "json"}); code != 0 {
		t.Fatalf("ls json exit code = %d, stderr = %s", code, stderr.String())
	}
	var items []map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &items); err != nil {
		t.Fatalf("unmarshal list json: %v", err)
	}

	var parentID, childID string
	for _, item := range items {
		title, _ := item["title"].(string)
		id, _ := item["id"].(string)
		switch title {
		case "Parent":
			parentID = id
		case "Child":
			childID = id
		}
	}
	if parentID == "" || childID == "" {
		t.Fatalf("expected both parent and child ids, got parent=%q child=%q", parentID, childID)
	}

	stdout.Reset()
	stderr.Reset()
	if code := runner.Run([]string{"--repo", repo, "link", childID, "--depends-on", parentID}); code != 0 {
		t.Fatalf("link exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "linked "+childID+" depends_on "+parentID) {
		t.Fatalf("unexpected link output: %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := runner.Run([]string{"--repo", repo, "show", childID}); code != 0 {
		t.Fatalf("show exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Depends On: "+parentID) {
		t.Fatalf("expected show output to include dependency, got %s", stdout.String())
	}
}

func TestRunnerChanges(t *testing.T) {
	repo := t.TempDir()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	runner := NewRunner(stdout, stderr)

	if code := runner.Run([]string{"--repo", repo, "init"}); code != 0 {
		t.Fatalf("init exit code = %d, stderr = %s", code, stderr.String())
	}
	if code := runner.Run([]string{"--repo", repo, "new", "--kind", "task", "--title", "History", "--goal", "Track changes", "--next", "Update it"}); code != 0 {
		t.Fatalf("new exit code = %d, stderr = %s", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := runner.Run([]string{"--repo", repo, "ls", "--format", "json"}); code != 0 {
		t.Fatalf("ls json exit code = %d, stderr = %s", code, stderr.String())
	}
	var items []map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &items); err != nil {
		t.Fatalf("unmarshal list json: %v", err)
	}
	itemID := items[0]["id"].(string)

	stdout.Reset()
	stderr.Reset()
	if code := runner.Run([]string{"--repo", repo, "update", itemID, "--summary", "history changed", "--next", "Review changes"}); code != 0 {
		t.Fatalf("update exit code = %d, stderr = %s", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := runner.Run([]string{"--repo", repo, "changes", "--item", itemID}); code != 0 {
		t.Fatalf("changes exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), itemID) || !strings.Contains(stdout.String(), "updated") {
		t.Fatalf("unexpected changes output: %s", stdout.String())
	}
}

func TestRunnerJiraPullPushAndTake(t *testing.T) {
	repo := t.TempDir()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	runner := NewRunner(stdout, stderr)

	if code := runner.Run([]string{"--repo", repo, "init"}); code != 0 {
		t.Fatalf("init exit code = %d, stderr = %s", code, stderr.String())
	}

	oldClient := jira.DefaultHTTPClient
	defer func() { jira.DefaultHTTPClient = oldClient }()
	jira.DefaultHTTPClient = &http.Client{Transport: runnerRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("agent@example.com:secret"))
		if r.Header.Get("Authorization") != wantAuth {
			return runnerJSONResponse(http.StatusUnauthorized, "unauthorized"), nil
		}
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/rest/api/3/issue/ABC-123/transitions":
			payload, _ := json.Marshal(map[string]any{
				"transitions": []any{
					map[string]any{
						"id":   "31",
						"name": "Start progress",
						"to":   map[string]any{"name": "In Progress"},
					},
				},
			})
			return runnerJSONResponse(http.StatusOK, string(payload)), nil
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/rest/api/3/issue/ABC-123"):
			payload, _ := json.Marshal(map[string]any{
				"key":  "ABC-123",
				"self": "https://example.atlassian.net/rest/api/3/issue/10000",
				"fields": map[string]any{
					"summary": "Imported Jira task",
					"description": map[string]any{
						"type":    "doc",
						"version": 1,
						"content": []any{
							map[string]any{
								"type": "paragraph",
								"content": []any{
									map[string]any{"type": "text", "text": "Bring Jira into aj."},
								},
							},
						},
					},
					"issuetype": map[string]any{"name": "Task"},
					"priority":  map[string]any{"name": "Medium"},
					"status":    map[string]any{"name": "To Do"},
					"updated":   "2026-03-13T12:00:00.000+0000",
				},
			})
			return runnerJSONResponse(http.StatusOK, string(payload)), nil
		case r.Method == http.MethodPost && r.URL.Path == "/rest/api/3/issue":
			payload, _ := json.Marshal(map[string]any{
				"key":  "ABC-456",
				"self": "https://example.atlassian.net/rest/api/3/issue/10001",
			})
			return runnerJSONResponse(http.StatusCreated, string(payload)), nil
		default:
			return runnerJSONResponse(http.StatusNotFound, "not found"), nil
		}
	})}

	writeRunnerJiraTestConfig(t, repo, "https://example.atlassian.net", "ABC")
	t.Setenv("AJ_JIRA_EMAIL", "agent@example.com")
	t.Setenv("AJ_JIRA_API_TOKEN", "secret")

	if code := runner.Run([]string{"--repo", repo, "jira", "pull", "ABC-123"}); code != 0 {
		t.Fatalf("jira pull exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "imported Jira ABC-123 as W-") {
		t.Fatalf("unexpected jira pull output: %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := runner.Run([]string{"--repo", repo, "take", "jira", "ABC-123", "--agent", "coder-1"}); code != 0 {
		t.Fatalf("take jira exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "using existing") || !strings.Contains(stdout.String(), "claimed") {
		t.Fatalf("unexpected take jira output: %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := runner.Run([]string{"--repo", repo, "new", "--kind", "task", "--title", "Push local item", "--goal", "Export into Jira", "--next", "Run jira push"}); code != 0 {
		t.Fatalf("new exit code = %d, stderr = %s", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := runner.Run([]string{"--repo", repo, "ls", "--format", "json"}); code != 0 {
		t.Fatalf("ls json exit code = %d, stderr = %s", code, stderr.String())
	}
	var items []map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &items); err != nil {
		t.Fatalf("unmarshal list json: %v", err)
	}

	var localID string
	for _, item := range items {
		if item["title"] == "Push local item" {
			localID = item["id"].(string)
			break
		}
	}
	if localID == "" {
		t.Fatalf("expected local item id in list output")
	}

	stdout.Reset()
	stderr.Reset()
	if code := runner.Run([]string{"--repo", repo, "jira", "push", localID}); code != 0 {
		t.Fatalf("jira push exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "exported "+localID+" to Jira ABC-456") {
		t.Fatalf("unexpected jira push output: %s", stdout.String())
	}
}

func TestRunnerJiraLinkSyncAndComment(t *testing.T) {
	repo := t.TempDir()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	runner := NewRunner(stdout, stderr)

	if code := runner.Run([]string{"--repo", repo, "init"}); code != 0 {
		t.Fatalf("init exit code = %d, stderr = %s", code, stderr.String())
	}

	oldClient := jira.DefaultHTTPClient
	defer func() { jira.DefaultHTTPClient = oldClient }()

	remoteUpdated := "2026-03-13T12:00:00.000+0000"
	remoteStatus := "To Do"
	transitionCalled := false
	commentCalled := false
	jira.DefaultHTTPClient = &http.Client{Transport: runnerRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("agent@example.com:secret"))
		if r.Header.Get("Authorization") != wantAuth {
			return runnerJSONResponse(http.StatusUnauthorized, "unauthorized"), nil
		}
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/rest/api/3/issue/ABC-123/transitions":
			payload, _ := json.Marshal(map[string]any{
				"transitions": []any{
					map[string]any{
						"id":   "31",
						"name": "Start progress",
						"to":   map[string]any{"name": "In Progress"},
					},
				},
			})
			return runnerJSONResponse(http.StatusOK, string(payload)), nil
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/rest/api/3/issue/ABC-123"):
			payload, _ := json.Marshal(map[string]any{
				"key":  "ABC-123",
				"self": "https://example.atlassian.net/rest/api/3/issue/10000",
				"fields": map[string]any{
					"summary":   "Linked issue",
					"issuetype": map[string]any{"name": "Task"},
					"priority":  map[string]any{"name": "Medium"},
					"status":    map[string]any{"name": remoteStatus},
					"updated":   remoteUpdated,
				},
			})
			return runnerJSONResponse(http.StatusOK, string(payload)), nil
		case r.Method == http.MethodPost && r.URL.Path == "/rest/api/3/issue/ABC-123/transitions":
			transitionCalled = true
			remoteStatus = "In Progress"
			return runnerJSONResponse(http.StatusNoContent, ""), nil
		case r.Method == http.MethodPut && r.URL.Path == "/rest/api/3/issue/ABC-123":
			remoteUpdated = "2026-03-13T13:00:00.000+0000"
			return runnerJSONResponse(http.StatusNoContent, ""), nil
		case r.Method == http.MethodPost && r.URL.Path == "/rest/api/3/issue/ABC-123/comment":
			commentCalled = true
			return runnerJSONResponse(http.StatusCreated, `{"id":"10001"}`), nil
		default:
			return runnerJSONResponse(http.StatusNotFound, "not found"), nil
		}
	})}

	writeRunnerJiraTestConfig(t, repo, "https://example.atlassian.net", "ABC")
	t.Setenv("AJ_JIRA_EMAIL", "agent@example.com")
	t.Setenv("AJ_JIRA_API_TOKEN", "secret")

	if code := runner.Run([]string{"--repo", repo, "new", "--kind", "task", "--title", "Sync me", "--goal", "Exercise jira link and sync", "--next", "Run jira link"}); code != 0 {
		t.Fatalf("new exit code = %d, stderr = %s", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := runner.Run([]string{"--repo", repo, "ls", "--format", "json"}); code != 0 {
		t.Fatalf("ls json exit code = %d, stderr = %s", code, stderr.String())
	}
	var items []map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &items); err != nil {
		t.Fatalf("unmarshal list json: %v", err)
	}
	itemID := items[0]["id"].(string)

	stdout.Reset()
	stderr.Reset()
	if code := runner.Run([]string{"--repo", repo, "jira", "link", itemID, "ABC-123"}); code != 0 {
		t.Fatalf("jira link exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "linked "+itemID+" to Jira ABC-123") {
		t.Fatalf("unexpected jira link output: %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := runner.Run([]string{"--repo", repo, "update", itemID, "--summary", "started linked work", "--status", "in_progress", "--next", "Keep local and Jira states aligned"}); code != 0 {
		t.Fatalf("update exit code = %d, stderr = %s", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := runner.Run([]string{"--repo", repo, "jira", "sync", itemID, "--dry-run"}); code != 0 {
		t.Fatalf("jira sync dry-run exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "direction=push") {
		t.Fatalf("unexpected jira sync dry-run output: %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := runner.Run([]string{"--repo", repo, "jira", "sync", itemID}); code != 0 {
		t.Fatalf("jira sync exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "synced "+itemID+" with Jira ABC-123 direction=push") {
		t.Fatalf("unexpected jira sync output: %s", stdout.String())
	}
	if !transitionCalled {
		t.Fatalf("expected jira transition API call during sync")
	}

	stdout.Reset()
	stderr.Reset()
	if code := runner.Run([]string{"--repo", repo, "jira", "comment", itemID, "--summary", "ready for review"}); code != 0 {
		t.Fatalf("jira comment exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "commented on Jira ABC-123 from "+itemID) {
		t.Fatalf("unexpected jira comment output: %s", stdout.String())
	}
	if !commentCalled {
		t.Fatalf("expected jira comment API call")
	}
}

func TestRunnerJiraStatusMapAndTransitions(t *testing.T) {
	repo := t.TempDir()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	runner := NewRunner(stdout, stderr)

	if code := runner.Run([]string{"--repo", repo, "init"}); code != 0 {
		t.Fatalf("init exit code = %d, stderr = %s", code, stderr.String())
	}

	oldClient := jira.DefaultHTTPClient
	defer func() { jira.DefaultHTTPClient = oldClient }()

	jira.DefaultHTTPClient = &http.Client{Transport: runnerRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("agent@example.com:secret"))
		if r.Header.Get("Authorization") != wantAuth {
			return runnerJSONResponse(http.StatusUnauthorized, "unauthorized"), nil
		}
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/rest/api/3/issue/ABC-123/transitions":
			payload, _ := json.Marshal(map[string]any{
				"transitions": []any{
					map[string]any{
						"id":   "31",
						"name": "In Progress",
						"to":   map[string]any{"name": "In Progress"},
					},
					map[string]any{
						"id":   "41",
						"name": "Done",
						"to":   map[string]any{"name": "Done"},
					},
				},
			})
			return runnerJSONResponse(http.StatusOK, string(payload)), nil
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/rest/api/3/issue/ABC-123"):
			payload, _ := json.Marshal(map[string]any{
				"key":  "ABC-123",
				"self": "https://example.atlassian.net/rest/api/3/issue/10000",
				"fields": map[string]any{
					"summary":   "Diagnostic issue",
					"issuetype": map[string]any{"name": "Task"},
					"priority":  map[string]any{"name": "Medium"},
					"status":    map[string]any{"name": "To Do"},
					"updated":   "2026-03-13T12:00:00.000+0000",
				},
			})
			return runnerJSONResponse(http.StatusOK, string(payload)), nil
		default:
			return runnerJSONResponse(http.StatusNotFound, "not found"), nil
		}
	})}

	writeRunnerJiraTestConfig(t, repo, "https://example.atlassian.net", "ABC")
	t.Setenv("AJ_JIRA_EMAIL", "agent@example.com")
	t.Setenv("AJ_JIRA_API_TOKEN", "secret")

	stdout.Reset()
	stderr.Reset()
	if code := runner.Run([]string{"--repo", repo, "jira", "status-map"}); code != 0 {
		t.Fatalf("jira status-map exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Blocked -> blocked") || !strings.Contains(stdout.String(), "Project: ABC") {
		t.Fatalf("unexpected jira status-map output: %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := runner.Run([]string{"--repo", repo, "new", "--kind", "task", "--title", "Diagnose transitions", "--goal", "Inspect jira transitions", "--next", "Link the item"}); code != 0 {
		t.Fatalf("new exit code = %d, stderr = %s", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := runner.Run([]string{"--repo", repo, "ls", "--format", "json"}); code != 0 {
		t.Fatalf("ls json exit code = %d, stderr = %s", code, stderr.String())
	}
	var items []map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &items); err != nil {
		t.Fatalf("unmarshal list json: %v", err)
	}
	itemID := items[0]["id"].(string)

	stdout.Reset()
	stderr.Reset()
	if code := runner.Run([]string{"--repo", repo, "jira", "link", itemID, "ABC-123"}); code != 0 {
		t.Fatalf("jira link exit code = %d, stderr = %s", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := runner.Run([]string{"--repo", repo, "update", itemID, "--summary", "started work", "--status", "in_progress", "--next", "Inspect live transitions"}); code != 0 {
		t.Fatalf("update exit code = %d, stderr = %s", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := runner.Run([]string{"--repo", repo, "jira", "transitions", itemID}); code != 0 {
		t.Fatalf("jira transitions exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Desired Jira Status: In Progress") || !strings.Contains(stdout.String(), "[matches desired]") {
		t.Fatalf("unexpected jira transitions output: %s", stdout.String())
	}
}

func TestRunnerJiraUnlinkAndReplace(t *testing.T) {
	repo := t.TempDir()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	runner := NewRunner(stdout, stderr)

	if code := runner.Run([]string{"--repo", repo, "init"}); code != 0 {
		t.Fatalf("init exit code = %d, stderr = %s", code, stderr.String())
	}

	oldClient := jira.DefaultHTTPClient
	defer func() { jira.DefaultHTTPClient = oldClient }()

	jira.DefaultHTTPClient = &http.Client{Transport: runnerRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("agent@example.com:secret"))
		if r.Header.Get("Authorization") != wantAuth {
			return runnerJSONResponse(http.StatusUnauthorized, "unauthorized"), nil
		}
		switch {
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/rest/api/3/issue/ABC-123"):
			payload, _ := json.Marshal(map[string]any{
				"key":  "ABC-123",
				"self": "https://example.atlassian.net/rest/api/3/issue/10000",
				"fields": map[string]any{
					"summary":   "First issue",
					"issuetype": map[string]any{"name": "Task"},
					"priority":  map[string]any{"name": "Medium"},
					"status":    map[string]any{"name": "To Do"},
					"updated":   "2026-03-13T12:00:00.000+0000",
				},
			})
			return runnerJSONResponse(http.StatusOK, string(payload)), nil
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/rest/api/3/issue/ABC-456"):
			payload, _ := json.Marshal(map[string]any{
				"key":  "ABC-456",
				"self": "https://example.atlassian.net/rest/api/3/issue/10001",
				"fields": map[string]any{
					"summary":   "Second issue",
					"issuetype": map[string]any{"name": "Task"},
					"priority":  map[string]any{"name": "Medium"},
					"status":    map[string]any{"name": "To Do"},
					"updated":   "2026-03-13T12:05:00.000+0000",
				},
			})
			return runnerJSONResponse(http.StatusOK, string(payload)), nil
		default:
			return runnerJSONResponse(http.StatusNotFound, "not found"), nil
		}
	})}

	writeRunnerJiraTestConfig(t, repo, "https://example.atlassian.net", "ABC")
	t.Setenv("AJ_JIRA_EMAIL", "agent@example.com")
	t.Setenv("AJ_JIRA_API_TOKEN", "secret")

	stdout.Reset()
	stderr.Reset()
	if code := runner.Run([]string{"--repo", repo, "new", "--kind", "task", "--title", "Relink me", "--goal", "Exercise unlink and replace flows", "--next", "Link the item"}); code != 0 {
		t.Fatalf("new exit code = %d, stderr = %s", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := runner.Run([]string{"--repo", repo, "ls", "--format", "json"}); code != 0 {
		t.Fatalf("ls json exit code = %d, stderr = %s", code, stderr.String())
	}
	var items []map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &items); err != nil {
		t.Fatalf("unmarshal list json: %v", err)
	}
	itemID := items[0]["id"].(string)

	stdout.Reset()
	stderr.Reset()
	if code := runner.Run([]string{"--repo", repo, "jira", "link", itemID, "ABC-123"}); code != 0 {
		t.Fatalf("jira link exit code = %d, stderr = %s", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := runner.Run([]string{"--repo", repo, "jira", "link", itemID, "ABC-456"}); code != 1 {
		t.Fatalf("jira relink without replace exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "--replace") {
		t.Fatalf("expected relink failure to mention --replace, got %s", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := runner.Run([]string{"--repo", repo, "jira", "unlink", itemID}); code != 1 {
		t.Fatalf("jira unlink without force exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "--force") {
		t.Fatalf("expected unlink failure to mention --force, got %s", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := runner.Run([]string{"--repo", repo, "jira", "unlink", itemID, "--force"}); code != 0 {
		t.Fatalf("jira unlink force exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "unlinked "+itemID+" from Jira") {
		t.Fatalf("unexpected jira unlink output: %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := runner.Run([]string{"--repo", repo, "jira", "link", itemID, "ABC-456", "--replace"}); code != 0 {
		t.Fatalf("jira link replace exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "linked "+itemID+" to Jira ABC-456") {
		t.Fatalf("unexpected jira link replace output: %s", stdout.String())
	}
}

func writeRunnerJiraTestConfig(t *testing.T, repo, baseURL, project string) {
	t.Helper()
	raw := `schema_version = 1
default_output = "brief"
default_lease_ttl = "4h"

[jira]
enabled = true
base_url = "` + baseURL + `"
project = "` + project + `"

[jira.status_map]
"To Do" = "todo"
"In Progress" = "in_progress"
"Blocked" = "blocked"
"In Review" = "in_review"
"Done" = "done"
`
	if err := os.WriteFile(filepath.Join(repo, ".aj", "config.toml"), []byte(raw), 0o644); err != nil {
		t.Fatalf("write jira test config: %v", err)
	}
}

type runnerRoundTripFunc func(*http.Request) (*http.Response, error)

func (fn runnerRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func runnerJSONResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestRunnerShowHistory(t *testing.T) {
	repo := t.TempDir()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	runner := NewRunner(stdout, stderr)

	if code := runner.Run([]string{"--repo", repo, "init"}); code != 0 {
		t.Fatalf("init exit code = %d, stderr = %s", code, stderr.String())
	}
	if code := runner.Run([]string{"--repo", repo, "new", "--kind", "task", "--title", "History view", "--goal", "Show recent item events", "--next", "Update it"}); code != 0 {
		t.Fatalf("new exit code = %d, stderr = %s", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := runner.Run([]string{"--repo", repo, "ls", "--format", "json"}); code != 0 {
		t.Fatalf("ls json exit code = %d, stderr = %s", code, stderr.String())
	}
	var items []map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &items); err != nil {
		t.Fatalf("unmarshal list json: %v", err)
	}
	itemID := items[0]["id"].(string)

	stdout.Reset()
	stderr.Reset()
	if code := runner.Run([]string{"--repo", repo, "update", itemID, "--summary", "history written", "--next", "Inspect history"}); code != 0 {
		t.Fatalf("update exit code = %d, stderr = %s", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := runner.Run([]string{"--repo", repo, "show", itemID, "--history", "--limit", "2"}); code != 0 {
		t.Fatalf("show history exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "History:") || !strings.Contains(stdout.String(), "updated") {
		t.Fatalf("unexpected show history output: %s", stdout.String())
	}
}

func TestRunnerReady(t *testing.T) {
	repo := t.TempDir()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	runner := NewRunner(stdout, stderr)

	if code := runner.Run([]string{"--repo", repo, "init"}); code != 0 {
		t.Fatalf("init exit code = %d, stderr = %s", code, stderr.String())
	}
	if code := runner.Run([]string{"--repo", repo, "new", "--kind", "task", "--title", "Ready task", "--goal", "Appear in ready output", "--next", "Start it"}); code != 0 {
		t.Fatalf("new ready exit code = %d, stderr = %s", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := runner.Run([]string{"--repo", repo, "ready"}); code != 0 {
		t.Fatalf("ready exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Ready task") || !strings.Contains(stdout.String(), "available") {
		t.Fatalf("unexpected ready output: %s", stdout.String())
	}
}
