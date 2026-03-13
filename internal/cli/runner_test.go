package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
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
