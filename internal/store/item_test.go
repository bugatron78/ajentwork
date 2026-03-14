package store

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ajentwork/internal/domain"
	"ajentwork/internal/jira"
)

func TestCreateListAndGetItem(t *testing.T) {
	repo := t.TempDir()
	if _, err := InitRepo(InitOptions{RepoPath: repo}); err != nil {
		t.Fatalf("init repo: %v", err)
	}

	created, err := CreateItem(CreateItemOptions{
		RepoPath:   repo,
		Kind:       domain.KindTask,
		Title:      "Bootstrap CLI",
		Goal:       "Ship the first command set",
		NextAction: "Implement item storage",
		Priority:   1,
	})
	if err != nil {
		t.Fatalf("create item: %v", err)
	}

	if !strings.HasPrefix(created.ID, "W-") {
		t.Fatalf("expected item id prefix W-, got %s", created.ID)
	}

	itemPath := filepath.Join(repo, ".aj", "issues", created.ID, "meta.toml")
	if _, err := GetItem(repo, created.ID); err != nil {
		t.Fatalf("get item: %v", err)
	}
	if _, err := os.Stat(itemPath); err != nil {
		t.Fatalf("stat item metadata: %v", err)
	}

	items, err := ListItems(repo)
	if err != nil {
		t.Fatalf("list items: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one item, got %d", len(items))
	}
	if items[0].Title != "Bootstrap CLI" {
		t.Fatalf("unexpected listed item title %q", items[0].Title)
	}
}

func TestCreateItemPersistsStructuredContext(t *testing.T) {
	repo := t.TempDir()
	if _, err := InitRepo(InitOptions{RepoPath: repo}); err != nil {
		t.Fatalf("init repo: %v", err)
	}

	created, err := CreateItem(CreateItemOptions{
		RepoPath:      repo,
		Kind:          domain.KindFeature,
		Title:         "Structured authoring",
		Goal:          "Give agents richer ticket context",
		NextAction:    "Wire context fields through create and show",
		Acceptance:    []string{"agents can record success criteria", "show surfaces those criteria compactly"},
		Constraints:   []string{"keep storage git-friendly"},
		Risks:         []string{"too much verbosity can waste tokens, especially in prompt mode"},
		RelevantFiles: []string{"internal/store/item.go", "internal/render/item.go"},
		Verification:  []string{"run go test ./...", "inspect aj show output"},
		Priority:      1,
	})
	if err != nil {
		t.Fatalf("create item: %v", err)
	}

	loaded, err := GetItem(repo, created.ID)
	if err != nil {
		t.Fatalf("get item: %v", err)
	}

	if got, want := strings.Join(loaded.Acceptance, "|"), "agents can record success criteria|show surfaces those criteria compactly"; got != want {
		t.Fatalf("acceptance = %q, want %q", got, want)
	}
	if got, want := strings.Join(loaded.Constraints, "|"), "keep storage git-friendly"; got != want {
		t.Fatalf("constraints = %q, want %q", got, want)
	}
	if got, want := strings.Join(loaded.Risks, "|"), "too much verbosity can waste tokens, especially in prompt mode"; got != want {
		t.Fatalf("risks = %q, want %q", got, want)
	}
	if got, want := strings.Join(loaded.RelevantFiles, "|"), "internal/store/item.go|internal/render/item.go"; got != want {
		t.Fatalf("relevant files = %q, want %q", got, want)
	}
	if got, want := strings.Join(loaded.Verification, "|"), "run go test ./...|inspect aj show output"; got != want {
		t.Fatalf("verification = %q, want %q", got, want)
	}
}

func TestAttachArtifactAndRecordReceipt(t *testing.T) {
	repo := t.TempDir()
	if _, err := InitRepo(InitOptions{RepoPath: repo}); err != nil {
		t.Fatalf("init repo: %v", err)
	}

	item, err := CreateItem(CreateItemOptions{
		RepoPath:   repo,
		Kind:       domain.KindTask,
		Title:      "Capture evidence",
		Goal:       "Persist logs and receipts for another agent",
		NextAction: "Attach a log and record a test receipt",
		Priority:   1,
	})
	if err != nil {
		t.Fatalf("create item: %v", err)
	}

	logPath := filepath.Join(repo, "build.log")
	if err := os.WriteFile(logPath, []byte("build output"), 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}

	artifact, err := AttachArtifact(AttachArtifactOptions{
		RepoPath: repo,
		ItemID:   item.ID,
		Path:     logPath,
		Summary:  "build log before the fix",
		Label:    "pre-fix-log",
	})
	if err != nil {
		t.Fatalf("attach artifact: %v", err)
	}
	if artifact.StoredPath == "" {
		t.Fatalf("expected stored path to be set")
	}
	if _, err := os.Stat(artifact.StoredPath); err != nil {
		t.Fatalf("stat copied artifact: %v", err)
	}

	receiptOutputPath := filepath.Join(repo, "test.log")
	if err := os.WriteFile(receiptOutputPath, []byte("test output"), 0o644); err != nil {
		t.Fatalf("write receipt output: %v", err)
	}

	receipt, err := RecordReceipt(RecordReceiptOptions{
		RepoPath: repo,
		ItemID:   item.ID,
		Summary:  "go test failed before the patch",
		Command:  "go test ./...",
		ExitCode: 1,
		Output:   receiptOutputPath,
		Label:    "test-failure",
	})
	if err != nil {
		t.Fatalf("record receipt: %v", err)
	}
	if receipt.ExitCode == nil || *receipt.ExitCode != 1 {
		t.Fatalf("expected exit code 1, got %#v", receipt.ExitCode)
	}

	artifacts, err := ListArtifacts(repo, item.ID, 10)
	if err != nil {
		t.Fatalf("list artifacts: %v", err)
	}
	if len(artifacts) != 2 {
		t.Fatalf("expected two artifacts, got %d", len(artifacts))
	}

	updated, err := GetItem(repo, item.ID)
	if err != nil {
		t.Fatalf("get updated item: %v", err)
	}
	if !updated.UpdatedAt.After(item.UpdatedAt) {
		t.Fatalf("expected item updated_at to advance after artifact activity")
	}
}

func TestCheckpointItem(t *testing.T) {
	repo := t.TempDir()
	if _, err := InitRepo(InitOptions{RepoPath: repo}); err != nil {
		t.Fatalf("init repo: %v", err)
	}

	item, err := CreateItem(CreateItemOptions{
		RepoPath:   repo,
		Kind:       domain.KindTask,
		Title:      "Checkpoint work",
		Goal:       "Capture a resume point",
		NextAction: "Implement checkpoint support",
		Priority:   1,
	})
	if err != nil {
		t.Fatalf("create item: %v", err)
	}

	next := "Dogfood the checkpoint flow"
	checkpointed, err := CheckpointItem(CheckpointItemOptions{
		RepoPath:   repo,
		ItemID:     item.ID,
		Summary:    "checkpoint support is implemented; only smoke-testing remains",
		NextAction: &next,
		Risks:      []string{"rendering could be too verbose"},
		Verify:     []string{"run go test ./...", "inspect aj show output"},
	})
	if err != nil {
		t.Fatalf("checkpoint item: %v", err)
	}
	if checkpointed.Checkpoint == nil {
		t.Fatalf("expected checkpoint to be recorded")
	}
	if checkpointed.Checkpoint.Summary != "checkpoint support is implemented; only smoke-testing remains" {
		t.Fatalf("unexpected checkpoint summary %q", checkpointed.Checkpoint.Summary)
	}
	if got, want := strings.Join(checkpointed.Checkpoint.Risks, "|"), "rendering could be too verbose"; got != want {
		t.Fatalf("risks = %q, want %q", got, want)
	}
	if got, want := strings.Join(checkpointed.Checkpoint.Verify, "|"), "run go test ./...|inspect aj show output"; got != want {
		t.Fatalf("verify = %q, want %q", got, want)
	}
	if checkpointed.NextAction != next {
		t.Fatalf("next action = %q, want %q", checkpointed.NextAction, next)
	}

	loaded, err := GetItem(repo, item.ID)
	if err != nil {
		t.Fatalf("reload item: %v", err)
	}
	if loaded.Checkpoint == nil || loaded.Checkpoint.Summary != checkpointed.Checkpoint.Summary {
		t.Fatalf("expected persisted checkpoint, got %#v", loaded.Checkpoint)
	}
}

func TestUpdateAndCompleteItem(t *testing.T) {
	repo := t.TempDir()
	if _, err := InitRepo(InitOptions{RepoPath: repo}); err != nil {
		t.Fatalf("init repo: %v", err)
	}

	created, err := CreateItem(CreateItemOptions{
		RepoPath:   repo,
		Kind:       domain.KindTask,
		Title:      "Ship update flow",
		Goal:       "Allow items to record progress",
		NextAction: "Implement update command",
		Priority:   0,
	})
	if err != nil {
		t.Fatalf("create item: %v", err)
	}

	status := domain.StatusInProgress
	next := "Write lifecycle tests"
	updated, err := UpdateItem(UpdateItemOptions{
		RepoPath:   repo,
		ItemID:     created.ID,
		Summary:    "implementation started",
		NextAction: &next,
		Status:     &status,
	})
	if err != nil {
		t.Fatalf("update item: %v", err)
	}
	if updated.Status != domain.StatusInProgress {
		t.Fatalf("expected in_progress, got %s", updated.Status)
	}
	if updated.NextAction != next {
		t.Fatalf("expected next action %q, got %q", next, updated.NextAction)
	}

	done, err := CompleteItem(CompleteItemOptions{
		RepoPath: repo,
		ItemID:   created.ID,
		Summary:  "update command shipped",
	})
	if err != nil {
		t.Fatalf("complete item: %v", err)
	}
	if done.Status != domain.StatusDone {
		t.Fatalf("expected done, got %s", done.Status)
	}
	if done.NextAction != "" {
		t.Fatalf("expected empty next action for done item, got %q", done.NextAction)
	}
	if done.Lease != nil {
		t.Fatalf("expected done item lease to be cleared, got %#v", done.Lease)
	}
}

func TestTakeAndReleaseItem(t *testing.T) {
	repo := t.TempDir()
	if _, err := InitRepo(InitOptions{RepoPath: repo}); err != nil {
		t.Fatalf("init repo: %v", err)
	}

	created, err := CreateItem(CreateItemOptions{
		RepoPath:   repo,
		Kind:       domain.KindTask,
		Title:      "Claim work",
		Goal:       "Add lease support",
		NextAction: "Implement take command",
		Priority:   1,
	})
	if err != nil {
		t.Fatalf("create item: %v", err)
	}

	claimed, err := TakeItem(TakeItemOptions{
		RepoPath: repo,
		ItemID:   created.ID,
		Agent:    "coder-1",
		TTL:      2 * time.Hour,
	})
	if err != nil {
		t.Fatalf("take item: %v", err)
	}
	if claimed.Lease == nil || claimed.Lease.Owner != "coder-1" {
		t.Fatalf("expected active lease for coder-1, got %#v", claimed.Lease)
	}

	released, err := ReleaseItem(ReleaseItemOptions{
		RepoPath: repo,
		ItemID:   created.ID,
	})
	if err != nil {
		t.Fatalf("release item: %v", err)
	}
	if released.Lease != nil {
		t.Fatalf("expected lease to be cleared, got %#v", released.Lease)
	}
}

func TestBlockUnblockHandoffAndReopenItem(t *testing.T) {
	repo := t.TempDir()
	if _, err := InitRepo(InitOptions{RepoPath: repo}); err != nil {
		t.Fatalf("init repo: %v", err)
	}

	dependency, err := CreateItem(CreateItemOptions{
		RepoPath:   repo,
		Kind:       domain.KindTask,
		Title:      "Dependency",
		Goal:       "Unblock other work",
		NextAction: "Ship dependency",
		Priority:   0,
	})
	if err != nil {
		t.Fatalf("create dependency: %v", err)
	}

	item, err := CreateItem(CreateItemOptions{
		RepoPath:   repo,
		Kind:       domain.KindTask,
		Title:      "Coordination lifecycle",
		Goal:       "Exercise block handoff and reopen flows",
		NextAction: "Start work",
		Priority:   1,
	})
	if err != nil {
		t.Fatalf("create item: %v", err)
	}

	blocked, err := BlockItem(BlockItemOptions{
		RepoPath: repo,
		ItemID:   item.ID,
		Summary:  "waiting on dependency",
		OnID:     dependency.ID,
	})
	if err != nil {
		t.Fatalf("block item: %v", err)
	}
	if blocked.Status != domain.StatusBlocked {
		t.Fatalf("expected blocked status, got %s", blocked.Status)
	}
	if blocked.NextAction != "Wait for "+dependency.ID {
		t.Fatalf("expected wait next action, got %q", blocked.NextAction)
	}
	if len(blocked.DependsOn) != 1 || blocked.DependsOn[0] != dependency.ID {
		t.Fatalf("expected dependency on %s, got %#v", dependency.ID, blocked.DependsOn)
	}

	next := "Resume implementation"
	unblocked, err := UnblockItem(UnblockItemOptions{
		RepoPath:   repo,
		ItemID:     item.ID,
		Summary:    "dependency shipped",
		NextAction: &next,
	})
	if err != nil {
		t.Fatalf("unblock item: %v", err)
	}
	if unblocked.Status != domain.StatusTodo {
		t.Fatalf("expected todo status after unblock, got %s", unblocked.Status)
	}
	if unblocked.NextAction != next {
		t.Fatalf("expected updated next action %q, got %q", next, unblocked.NextAction)
	}

	handedOff, err := HandoffItem(HandoffItemOptions{
		RepoPath: repo,
		ItemID:   item.ID,
		ToAgent:  "reviewer-1",
		Summary:  "implementation ready for review",
		TTL:      time.Hour,
	})
	if err != nil {
		t.Fatalf("handoff item: %v", err)
	}
	if handedOff.Lease == nil || handedOff.Lease.Owner != "reviewer-1" {
		t.Fatalf("expected reviewer lease, got %#v", handedOff.Lease)
	}

	done, err := CompleteItem(CompleteItemOptions{
		RepoPath: repo,
		ItemID:   item.ID,
		Summary:  "review completed",
	})
	if err != nil {
		t.Fatalf("complete item: %v", err)
	}
	if done.Status != domain.StatusDone {
		t.Fatalf("expected done status, got %s", done.Status)
	}

	reopened, err := ReopenItem(ReopenItemOptions{
		RepoPath:   repo,
		ItemID:     item.ID,
		Summary:    "follow-up regression found",
		NextAction: "Add a failing regression test",
	})
	if err != nil {
		t.Fatalf("reopen item: %v", err)
	}
	if reopened.Status != domain.StatusTodo {
		t.Fatalf("expected todo status after reopen, got %s", reopened.Status)
	}
	if reopened.NextAction != "Add a failing regression test" {
		t.Fatalf("unexpected reopen next action %q", reopened.NextAction)
	}
}

func TestCompleteItemClearsExistingLease(t *testing.T) {
	repo := t.TempDir()
	if _, err := InitRepo(InitOptions{RepoPath: repo}); err != nil {
		t.Fatalf("init repo: %v", err)
	}

	created, err := CreateItem(CreateItemOptions{
		RepoPath:   repo,
		Kind:       domain.KindTask,
		Title:      "Done clears lease",
		Goal:       "Avoid stale ownership on completed work",
		NextAction: "Complete the task",
		Priority:   1,
	})
	if err != nil {
		t.Fatalf("create item: %v", err)
	}

	if _, err := TakeItem(TakeItemOptions{
		RepoPath: repo,
		ItemID:   created.ID,
		Agent:    "coder-1",
		TTL:      time.Hour,
	}); err != nil {
		t.Fatalf("take item: %v", err)
	}

	done, err := CompleteItem(CompleteItemOptions{
		RepoPath: repo,
		ItemID:   created.ID,
		Summary:  "completed with lease cleanup",
	})
	if err != nil {
		t.Fatalf("complete item: %v", err)
	}
	if done.Lease != nil {
		t.Fatalf("expected done item lease to be nil, got %#v", done.Lease)
	}
}

func TestRecommendNextAndInbox(t *testing.T) {
	repo := t.TempDir()
	if _, err := InitRepo(InitOptions{RepoPath: repo}); err != nil {
		t.Fatalf("init repo: %v", err)
	}

	owned, err := CreateItem(CreateItemOptions{
		RepoPath:   repo,
		Kind:       domain.KindTask,
		Title:      "Owned work",
		Goal:       "Keep working on owned task",
		NextAction: "Continue implementation",
		Priority:   2,
	})
	if err != nil {
		t.Fatalf("create owned item: %v", err)
	}
	if _, err := TakeItem(TakeItemOptions{
		RepoPath: repo,
		ItemID:   owned.ID,
		Agent:    "coder-1",
		TTL:      2 * time.Hour,
	}); err != nil {
		t.Fatalf("take owned item: %v", err)
	}

	available, err := CreateItem(CreateItemOptions{
		RepoPath:   repo,
		Kind:       domain.KindTask,
		Title:      "Available work",
		Goal:       "Should show as available",
		NextAction: "Claim this task",
		Priority:   0,
	})
	if err != nil {
		t.Fatalf("create available item: %v", err)
	}

	nextOwned, err := RecommendNext(repo, "coder-1")
	if err != nil {
		t.Fatalf("recommend next for owner: %v", err)
	}
	if nextOwned.Item.ID != owned.ID {
		t.Fatalf("expected owned item recommendation, got %s", nextOwned.Item.ID)
	}

	nextAvailable, err := RecommendNext(repo, "")
	if err != nil {
		t.Fatalf("recommend next available: %v", err)
	}
	if nextAvailable.Item.ID != available.ID {
		t.Fatalf("expected available item recommendation, got %s", nextAvailable.Item.ID)
	}

	inbox, err := Inbox(repo, "coder-1")
	if err != nil {
		t.Fatalf("inbox: %v", err)
	}
	if len(inbox) < 2 {
		t.Fatalf("expected at least two inbox entries, got %d", len(inbox))
	}
	if inbox[0].Reason != "owned" {
		t.Fatalf("expected first inbox entry to be owned, got %s", inbox[0].Reason)
	}
}

func TestLinkDependencyAffectsReadiness(t *testing.T) {
	repo := t.TempDir()
	if _, err := InitRepo(InitOptions{RepoPath: repo}); err != nil {
		t.Fatalf("init repo: %v", err)
	}

	parent, err := CreateItem(CreateItemOptions{
		RepoPath:   repo,
		Kind:       domain.KindTask,
		Title:      "Foundation",
		Goal:       "Build base feature",
		NextAction: "Finish foundation",
		Priority:   1,
	})
	if err != nil {
		t.Fatalf("create parent: %v", err)
	}

	child, err := CreateItem(CreateItemOptions{
		RepoPath:   repo,
		Kind:       domain.KindTask,
		Title:      "Dependent work",
		Goal:       "Build on foundation",
		NextAction: "Wait for parent",
		Priority:   0,
	})
	if err != nil {
		t.Fatalf("create child: %v", err)
	}

	linked, err := LinkDependency(LinkDependencyOptions{
		RepoPath:    repo,
		ItemID:      child.ID,
		DependsOnID: parent.ID,
	})
	if err != nil {
		t.Fatalf("link dependency: %v", err)
	}
	if len(linked.DependsOn) != 1 || linked.DependsOn[0] != parent.ID {
		t.Fatalf("expected dependency on %s, got %#v", parent.ID, linked.DependsOn)
	}

	next, err := RecommendNext(repo, "")
	if err != nil {
		t.Fatalf("recommend next: %v", err)
	}
	if next.Item.ID != parent.ID {
		t.Fatalf("expected dependency parent to be recommended, got %s", next.Item.ID)
	}

	inbox, err := Inbox(repo, "")
	if err != nil {
		t.Fatalf("inbox: %v", err)
	}
	foundWaiting := false
	for _, entry := range inbox {
		if entry.Item.ID == child.ID {
			foundWaiting = true
			if entry.Reason != "waiting" {
				t.Fatalf("expected waiting reason for child, got %s", entry.Reason)
			}
			if len(entry.WaitingOn) != 1 || entry.WaitingOn[0] != parent.ID {
				t.Fatalf("expected waiting on %s, got %#v", parent.ID, entry.WaitingOn)
			}
		}
	}
	if !foundWaiting {
		t.Fatalf("expected inbox to include child waiting on dependency")
	}
}

func TestListChanges(t *testing.T) {
	repo := t.TempDir()
	if _, err := InitRepo(InitOptions{RepoPath: repo}); err != nil {
		t.Fatalf("init repo: %v", err)
	}

	item, err := CreateItem(CreateItemOptions{
		RepoPath:   repo,
		Kind:       domain.KindTask,
		Title:      "History item",
		Goal:       "Generate changes",
		NextAction: "Update the item",
		Priority:   1,
	})
	if err != nil {
		t.Fatalf("create item: %v", err)
	}

	next := "Finish history work"
	if _, err := UpdateItem(UpdateItemOptions{
		RepoPath:   repo,
		ItemID:     item.ID,
		Summary:    "history updated",
		NextAction: &next,
	}); err != nil {
		t.Fatalf("update item: %v", err)
	}

	events, err := ListChanges(ChangesOptions{
		RepoPath: repo,
		ItemID:   item.ID,
		Limit:    10,
	})
	if err != nil {
		t.Fatalf("list changes: %v", err)
	}
	if len(events) < 2 {
		t.Fatalf("expected at least two events, got %d", len(events))
	}
	if events[0].ItemID != item.ID {
		t.Fatalf("expected item id %s, got %s", item.ID, events[0].ItemID)
	}
}

func TestReadyFiltersBlockedAndForeignLeasedItems(t *testing.T) {
	repo := t.TempDir()
	if _, err := InitRepo(InitOptions{RepoPath: repo}); err != nil {
		t.Fatalf("init repo: %v", err)
	}

	readyItem, err := CreateItem(CreateItemOptions{
		RepoPath:   repo,
		Kind:       domain.KindTask,
		Title:      "Ready item",
		Goal:       "Should appear in ready view",
		NextAction: "Start work",
		Priority:   1,
	})
	if err != nil {
		t.Fatalf("create ready item: %v", err)
	}

	foreign, err := CreateItem(CreateItemOptions{
		RepoPath:   repo,
		Kind:       domain.KindTask,
		Title:      "Foreign leased item",
		Goal:       "Should be hidden from other agents",
		NextAction: "Wait",
		Priority:   0,
	})
	if err != nil {
		t.Fatalf("create foreign item: %v", err)
	}
	if _, err := TakeItem(TakeItemOptions{
		RepoPath: repo,
		ItemID:   foreign.ID,
		Agent:    "reviewer-1",
		TTL:      time.Hour,
	}); err != nil {
		t.Fatalf("take foreign item: %v", err)
	}

	waitingParent, err := CreateItem(CreateItemOptions{
		RepoPath:   repo,
		Kind:       domain.KindTask,
		Title:      "Dependency parent",
		Goal:       "Needed first",
		NextAction: "Finish parent",
		Priority:   0,
	})
	if err != nil {
		t.Fatalf("create parent: %v", err)
	}
	waitingChild, err := CreateItem(CreateItemOptions{
		RepoPath:   repo,
		Kind:       domain.KindTask,
		Title:      "Waiting child",
		Goal:       "Blocked by parent",
		NextAction: "Wait for parent",
		Priority:   0,
	})
	if err != nil {
		t.Fatalf("create child: %v", err)
	}
	if _, err := LinkDependency(LinkDependencyOptions{
		RepoPath:    repo,
		ItemID:      waitingChild.ID,
		DependsOnID: waitingParent.ID,
	}); err != nil {
		t.Fatalf("link dependency: %v", err)
	}

	results, err := Ready(ReadyOptions{
		RepoPath: repo,
		Agent:    "coder-1",
	})
	if err != nil {
		t.Fatalf("ready: %v", err)
	}

	foundReady := false
	for _, entry := range results {
		if entry.Item.ID == readyItem.ID {
			foundReady = true
		}
		if entry.Item.ID == foreign.ID || entry.Item.ID == waitingChild.ID {
			t.Fatalf("unexpected item %s in ready results", entry.Item.ID)
		}
	}
	if !foundReady {
		t.Fatalf("expected ready item %s to appear", readyItem.ID)
	}
}

func TestImportAndExportJiraIssue(t *testing.T) {
	repo := t.TempDir()
	if _, err := InitRepo(InitOptions{RepoPath: repo}); err != nil {
		t.Fatalf("init repo: %v", err)
	}

	oldClient := jira.DefaultHTTPClient
	defer func() { jira.DefaultHTTPClient = oldClient }()
	remoteExportStatus := "To Do"
	remoteExportUpdated := "2026-03-13T12:05:00.000+0000"
	jira.DefaultHTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("agent@example.com:secret"))
		if r.Header.Get("Authorization") != wantAuth {
			return testJSONResponse(http.StatusUnauthorized, "unauthorized"), nil
		}
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/rest/api/3/issue/ABC-456/transitions":
			payload, _ := json.Marshal(map[string]any{
				"transitions": []any{
					map[string]any{
						"id":   "31",
						"name": "Done",
						"to":   map[string]any{"name": "Done"},
					},
				},
			})
			return testJSONResponse(http.StatusOK, string(payload)), nil
		case r.Method == http.MethodGet && r.URL.Path == "/rest/api/3/issue/ABC-123/transitions":
			payload, _ := json.Marshal(map[string]any{
				"transitions": []any{
					map[string]any{
						"id":   "31",
						"name": "Start progress",
						"to":   map[string]any{"name": "In Progress"},
					},
					map[string]any{
						"id":   "41",
						"name": "Done",
						"to":   map[string]any{"name": "Done"},
					},
				},
			})
			return testJSONResponse(http.StatusOK, string(payload)), nil
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/rest/api/3/issue/ABC-123"):
			payload, _ := json.Marshal(map[string]any{
				"key":  "ABC-123",
				"self": "https://example.atlassian.net/rest/api/3/issue/10000",
				"fields": map[string]any{
					"summary": "Imported bug",
					"description": map[string]any{
						"type":    "doc",
						"version": 1,
						"content": []any{
							map[string]any{
								"type": "paragraph",
								"content": []any{
									map[string]any{"type": "text", "text": "Investigate the failing sync path."},
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
			return testJSONResponse(http.StatusOK, string(payload)), nil
		case r.Method == http.MethodPost && r.URL.Path == "/rest/api/3/issue":
			payload, _ := json.Marshal(map[string]any{
				"key":  "ABC-456",
				"self": "https://example.atlassian.net/rest/api/3/issue/10001",
			})
			return testJSONResponse(http.StatusCreated, string(payload)), nil
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/rest/api/3/issue/ABC-456"):
			payload, _ := json.Marshal(map[string]any{
				"key":  "ABC-456",
				"self": "https://example.atlassian.net/rest/api/3/issue/10001",
				"fields": map[string]any{
					"summary":   "Export me",
					"issuetype": map[string]any{"name": "Task"},
					"priority":  map[string]any{"name": "Medium"},
					"status":    map[string]any{"name": remoteExportStatus},
					"updated":   remoteExportUpdated,
				},
			})
			return testJSONResponse(http.StatusOK, string(payload)), nil
		default:
			return testJSONResponse(http.StatusNotFound, "not found"), nil
		}
	})}

	writeJiraTestConfig(t, repo, "https://example.atlassian.net", "ABC")
	t.Setenv("AJ_JIRA_EMAIL", "agent@example.com")
	t.Setenv("AJ_JIRA_API_TOKEN", "secret")

	imported, err := ImportJiraIssue(ImportJiraIssueOptions{
		RepoPath: repo,
		IssueKey: "ABC-123",
	})
	if err != nil {
		t.Fatalf("import jira issue: %v", err)
	}
	if imported.AlreadyLinked {
		t.Fatalf("expected first import to create a new local item")
	}
	if imported.Item.Jira == nil || imported.Item.Jira.Key != "ABC-123" {
		t.Fatalf("expected jira metadata on imported item, got %#v", imported.Item.Jira)
	}
	if imported.Item.Kind != domain.KindBug || imported.Item.Status != domain.StatusInProgress {
		t.Fatalf("unexpected imported item mapping: %#v", imported.Item)
	}

	reused, err := ImportJiraIssue(ImportJiraIssueOptions{
		RepoPath: repo,
		IssueKey: "ABC-123",
	})
	if err != nil {
		t.Fatalf("re-import jira issue: %v", err)
	}
	if !reused.AlreadyLinked || reused.Item.ID != imported.Item.ID {
		t.Fatalf("expected import reuse, got %#v", reused)
	}

	local, err := CreateItem(CreateItemOptions{
		RepoPath:   repo,
		Kind:       domain.KindTask,
		Title:      "Export me",
		Goal:       "Create a Jira issue from local work",
		NextAction: "Push to Jira",
		Priority:   1,
	})
	if err != nil {
		t.Fatalf("create local item: %v", err)
	}

	exported, err := ExportJiraIssue(ExportJiraIssueOptions{
		RepoPath: repo,
		ItemID:   local.ID,
	})
	if err != nil {
		t.Fatalf("export jira issue: %v", err)
	}
	if exported.Item.Jira == nil || exported.Item.Jira.Key != "ABC-456" {
		t.Fatalf("expected exported jira key, got %#v", exported.Item.Jira)
	}
	if exported.Item.Jira.LastRemoteVersion != remoteExportUpdated {
		t.Fatalf("expected exported jira remote version to be recorded, got %#v", exported.Item.Jira)
	}
}

func TestExportJiraIssueAlignsRemoteStatus(t *testing.T) {
	repo := t.TempDir()
	if _, err := InitRepo(InitOptions{RepoPath: repo}); err != nil {
		t.Fatalf("init repo: %v", err)
	}

	oldClient := jira.DefaultHTTPClient
	defer func() { jira.DefaultHTTPClient = oldClient }()

	remoteStatus := "To Do"
	remoteUpdated := "2026-03-13T12:00:00.000+0000"
	transitionCalled := false
	jira.DefaultHTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("agent@example.com:secret"))
		if r.Header.Get("Authorization") != wantAuth {
			return testJSONResponse(http.StatusUnauthorized, "unauthorized"), nil
		}
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/rest/api/3/issue":
			payload, _ := json.Marshal(map[string]any{
				"key":  "ABC-456",
				"self": "https://example.atlassian.net/rest/api/3/issue/10001",
			})
			return testJSONResponse(http.StatusCreated, string(payload)), nil
		case r.Method == http.MethodGet && r.URL.Path == "/rest/api/3/issue/ABC-456/transitions":
			payload, _ := json.Marshal(map[string]any{
				"transitions": []any{
					map[string]any{
						"id":   "41",
						"name": "Done",
						"to":   map[string]any{"name": "Done"},
					},
				},
			})
			return testJSONResponse(http.StatusOK, string(payload)), nil
		case r.Method == http.MethodPost && r.URL.Path == "/rest/api/3/issue/ABC-456/transitions":
			transitionCalled = true
			remoteStatus = "Done"
			remoteUpdated = "2026-03-13T12:02:00.000+0000"
			return testJSONResponse(http.StatusNoContent, ""), nil
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/rest/api/3/issue/ABC-456"):
			payload, _ := json.Marshal(map[string]any{
				"key":  "ABC-456",
				"self": "https://example.atlassian.net/rest/api/3/issue/10001",
				"fields": map[string]any{
					"summary":   "Ship export status alignment",
					"issuetype": map[string]any{"name": "Task"},
					"priority":  map[string]any{"name": "Medium"},
					"status":    map[string]any{"name": remoteStatus},
					"updated":   remoteUpdated,
				},
			})
			return testJSONResponse(http.StatusOK, string(payload)), nil
		default:
			return testJSONResponse(http.StatusNotFound, "not found"), nil
		}
	})}

	writeJiraTestConfig(t, repo, "https://example.atlassian.net", "ABC")
	t.Setenv("AJ_JIRA_EMAIL", "agent@example.com")
	t.Setenv("AJ_JIRA_API_TOKEN", "secret")

	item, err := CreateItem(CreateItemOptions{
		RepoPath:   repo,
		Kind:       domain.KindTask,
		Title:      "Ship export status alignment",
		Goal:       "ensure exported done items land in Done",
		NextAction: "export to Jira",
		Priority:   1,
	})
	if err != nil {
		t.Fatalf("create item: %v", err)
	}
	if _, err := CompleteItem(CompleteItemOptions{
		RepoPath: repo,
		ItemID:   item.ID,
		Summary:  "implemented",
	}); err != nil {
		t.Fatalf("complete item: %v", err)
	}

	exported, err := ExportJiraIssue(ExportJiraIssueOptions{
		RepoPath: repo,
		ItemID:   item.ID,
	})
	if err != nil {
		t.Fatalf("export jira issue: %v", err)
	}
	if !transitionCalled {
		t.Fatalf("expected export to align remote Jira status")
	}
	if exported.Item.Jira == nil || exported.Item.Jira.LastRemoteVersion != remoteUpdated {
		t.Fatalf("expected export to refresh jira metadata, got %#v", exported.Item.Jira)
	}
}

func TestShowJiraStatusMap(t *testing.T) {
	repo := t.TempDir()
	if _, err := InitRepo(InitOptions{RepoPath: repo}); err != nil {
		t.Fatalf("init repo: %v", err)
	}

	writeJiraTestConfig(t, repo, "https://example.atlassian.net", "ABC")

	result, err := ShowJiraStatusMap(repo)
	if err != nil {
		t.Fatalf("show jira status map: %v", err)
	}
	if !result.Enabled || result.BaseURL != "https://example.atlassian.net" || result.Project != "ABC" {
		t.Fatalf("unexpected jira status map header: %#v", result)
	}
	if len(result.Entries) == 0 {
		t.Fatalf("expected jira status mappings")
	}
	if result.Entries[0].JiraStatus != "Blocked" || result.Entries[0].LocalStatus != domain.StatusBlocked {
		t.Fatalf("expected sorted jira status map entries, got %#v", result.Entries[0])
	}
}

func TestLinkAndSyncJiraIssue(t *testing.T) {
	repo := t.TempDir()
	if _, err := InitRepo(InitOptions{RepoPath: repo}); err != nil {
		t.Fatalf("init repo: %v", err)
	}

	oldClient := jira.DefaultHTTPClient
	defer func() { jira.DefaultHTTPClient = oldClient }()

	remoteUpdated := "2026-03-13T12:00:00.000+0000"
	remoteSummary := "Linked Jira summary"
	remoteDescription := "Remote Jira description"
	remoteStatus := "To Do"
	transitionCalled := false
	jira.DefaultHTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("agent@example.com:secret"))
		if r.Header.Get("Authorization") != wantAuth {
			return testJSONResponse(http.StatusUnauthorized, "unauthorized"), nil
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
					map[string]any{
						"id":   "41",
						"name": "Done",
						"to":   map[string]any{"name": "Done"},
					},
				},
			})
			return testJSONResponse(http.StatusOK, string(payload)), nil
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/rest/api/3/issue/ABC-123"):
			payload, _ := json.Marshal(map[string]any{
				"key":  "ABC-123",
				"self": "https://example.atlassian.net/rest/api/3/issue/10000",
				"fields": map[string]any{
					"summary": remoteSummary,
					"description": map[string]any{
						"type":    "doc",
						"version": 1,
						"content": []any{
							map[string]any{
								"type": "paragraph",
								"content": []any{
									map[string]any{"type": "text", "text": remoteDescription},
								},
							},
						},
					},
					"issuetype": map[string]any{"name": "Task"},
					"priority":  map[string]any{"name": "Medium"},
					"status":    map[string]any{"name": remoteStatus},
					"updated":   remoteUpdated,
				},
			})
			return testJSONResponse(http.StatusOK, string(payload)), nil
		case r.Method == http.MethodPost && r.URL.Path == "/rest/api/3/issue/ABC-123/transitions":
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode transition payload: %v", err)
			}
			transition := payload["transition"].(map[string]any)
			if transition["id"] != "31" {
				t.Fatalf("unexpected transition payload: %#v", payload)
			}
			transitionCalled = true
			remoteStatus = "In Progress"
			return testJSONResponse(http.StatusNoContent, ""), nil
		case r.Method == http.MethodPut && r.URL.Path == "/rest/api/3/issue/ABC-123":
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode sync payload: %v", err)
			}
			fields := payload["fields"].(map[string]any)
			if fields["summary"] != "Local linked item" {
				t.Fatalf("unexpected synced summary: %#v", fields["summary"])
			}
			remoteUpdated = "2026-03-13T13:00:00.000+0000"
			return testJSONResponse(http.StatusNoContent, ""), nil
		default:
			return testJSONResponse(http.StatusNotFound, "not found"), nil
		}
	})}

	writeJiraTestConfig(t, repo, "https://example.atlassian.net", "ABC")
	t.Setenv("AJ_JIRA_EMAIL", "agent@example.com")
	t.Setenv("AJ_JIRA_API_TOKEN", "secret")

	item, err := CreateItem(CreateItemOptions{
		RepoPath:   repo,
		Kind:       domain.KindTask,
		Title:      "Local linked item",
		Goal:       "Sync local work with Jira",
		NextAction: "Link to Jira",
		Priority:   1,
	})
	if err != nil {
		t.Fatalf("create item: %v", err)
	}

	linked, err := LinkJiraIssue(LinkJiraIssueOptions{
		RepoPath: repo,
		ItemID:   item.ID,
		IssueKey: "ABC-123",
	})
	if err != nil {
		t.Fatalf("link jira issue: %v", err)
	}
	if linked.Item.Jira == nil || linked.Item.Jira.SyncState != "dirty_local" {
		t.Fatalf("expected dirty_local linked jira metadata, got %#v", linked.Item.Jira)
	}

	status := domain.StatusInProgress
	next := "Keep the local and Jira states aligned"
	if _, err := UpdateItem(UpdateItemOptions{
		RepoPath:   repo,
		ItemID:     item.ID,
		Summary:    "started linked work",
		NextAction: &next,
		Status:     &status,
	}); err != nil {
		t.Fatalf("update local item before sync: %v", err)
	}

	dryRun, err := SyncJiraIssue(SyncJiraIssueOptions{
		RepoPath: repo,
		ItemID:   item.ID,
		DryRun:   true,
	})
	if err != nil {
		t.Fatalf("dry-run sync: %v", err)
	}
	if dryRun.Direction != "push" {
		t.Fatalf("expected dry-run push direction, got %q", dryRun.Direction)
	}

	synced, err := SyncJiraIssue(SyncJiraIssueOptions{
		RepoPath: repo,
		ItemID:   item.ID,
	})
	if err != nil {
		t.Fatalf("sync item: %v", err)
	}
	if synced.Direction != "push" || synced.Item.Jira.SyncState != "clean" {
		t.Fatalf("unexpected sync result: %#v", synced)
	}
	if !transitionCalled {
		t.Fatalf("expected jira status transition during sync push")
	}

	remoteSummary = "Remote changed summary"
	remoteDescription = "Remote changed description"
	remoteUpdated = "2026-03-13T14:00:00.000+0000"
	next = "Local follow-up"
	if _, err := UpdateItem(UpdateItemOptions{
		RepoPath:   repo,
		ItemID:     item.ID,
		Summary:    "local change after sync",
		NextAction: &next,
	}); err != nil {
		t.Fatalf("update local item: %v", err)
	}

	if _, err := SyncJiraIssue(SyncJiraIssueOptions{
		RepoPath: repo,
		ItemID:   item.ID,
	}); err == nil {
		t.Fatalf("expected conflict without resolution")
	}

	resolved, err := SyncJiraIssue(SyncJiraIssueOptions{
		RepoPath: repo,
		ItemID:   item.ID,
		Resolve:  "keep-remote",
	})
	if err != nil {
		t.Fatalf("resolve keep-remote sync: %v", err)
	}
	if resolved.Direction != "pull" || resolved.Item.Title != "Remote changed summary" {
		t.Fatalf("unexpected resolved sync result: %#v", resolved)
	}
}

func TestShowJiraTransitions(t *testing.T) {
	repo := t.TempDir()
	if _, err := InitRepo(InitOptions{RepoPath: repo}); err != nil {
		t.Fatalf("init repo: %v", err)
	}

	oldClient := jira.DefaultHTTPClient
	defer func() { jira.DefaultHTTPClient = oldClient }()

	jira.DefaultHTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("agent@example.com:secret"))
		if r.Header.Get("Authorization") != wantAuth {
			return testJSONResponse(http.StatusUnauthorized, "unauthorized"), nil
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
					map[string]any{
						"id":   "41",
						"name": "Done",
						"to":   map[string]any{"name": "Done"},
					},
				},
			})
			return testJSONResponse(http.StatusOK, string(payload)), nil
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/rest/api/3/issue/ABC-123"):
			payload, _ := json.Marshal(map[string]any{
				"key":  "ABC-123",
				"self": "https://example.atlassian.net/rest/api/3/issue/10000",
				"fields": map[string]any{
					"summary":   "Linked Jira summary",
					"issuetype": map[string]any{"name": "Task"},
					"priority":  map[string]any{"name": "Medium"},
					"status":    map[string]any{"name": "To Do"},
					"updated":   "2026-03-13T12:00:00.000+0000",
				},
			})
			return testJSONResponse(http.StatusOK, string(payload)), nil
		default:
			return testJSONResponse(http.StatusNotFound, "not found"), nil
		}
	})}

	writeJiraTestConfig(t, repo, "https://example.atlassian.net", "ABC")
	t.Setenv("AJ_JIRA_EMAIL", "agent@example.com")
	t.Setenv("AJ_JIRA_API_TOKEN", "secret")

	item, err := CreateItem(CreateItemOptions{
		RepoPath:   repo,
		Kind:       domain.KindTask,
		Title:      "Transition diagnostics",
		Goal:       "Inspect live Jira workflow options",
		NextAction: "Link the item",
		Priority:   1,
	})
	if err != nil {
		t.Fatalf("create item: %v", err)
	}

	if _, err := LinkJiraIssue(LinkJiraIssueOptions{
		RepoPath: repo,
		ItemID:   item.ID,
		IssueKey: "ABC-123",
	}); err != nil {
		t.Fatalf("link jira issue: %v", err)
	}

	status := domain.StatusInProgress
	next := "Run the transitions command"
	if _, err := UpdateItem(UpdateItemOptions{
		RepoPath:   repo,
		ItemID:     item.ID,
		Summary:    "local status changed",
		NextAction: &next,
		Status:     &status,
	}); err != nil {
		t.Fatalf("update local item: %v", err)
	}

	result, err := ShowJiraTransitions(repo, item.ID)
	if err != nil {
		t.Fatalf("show jira transitions: %v", err)
	}
	if result.RemoteStatus != "To Do" || result.DesiredStatus != "In Progress" || !result.CanTransition {
		t.Fatalf("unexpected transitions result header: %#v", result)
	}
	if len(result.Available) != 2 || !result.Available[0].MatchesDesired {
		t.Fatalf("unexpected transitions result entries: %#v", result.Available)
	}
}

func TestSearchJiraIssues(t *testing.T) {
	repo := t.TempDir()
	if _, err := InitRepo(InitOptions{RepoPath: repo}); err != nil {
		t.Fatalf("init repo: %v", err)
	}

	oldClient := jira.DefaultHTTPClient
	defer func() { jira.DefaultHTTPClient = oldClient }()

	var capturedJQL string
	jira.DefaultHTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("agent@example.com:secret"))
		if r.Header.Get("Authorization") != wantAuth {
			return testJSONResponse(http.StatusUnauthorized, "unauthorized"), nil
		}
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/rest/api/3/search/jql":
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode search payload: %v", err)
			}
			capturedJQL, _ = payload["jql"].(string)
			response, _ := json.Marshal(map[string]any{
				"issues": []any{
					map[string]any{
						"key":  "ABC-123",
						"self": "https://example.atlassian.net/rest/api/3/issue/10000",
						"fields": map[string]any{
							"summary":   "Cache invalidation bug",
							"issuetype": map[string]any{"name": "Bug"},
							"priority":  map[string]any{"name": "High"},
							"status":    map[string]any{"name": "To Do"},
							"updated":   "2026-03-13T12:00:00.000+0000",
						},
					},
				},
			})
			return testJSONResponse(http.StatusOK, string(response)), nil
		default:
			return testJSONResponse(http.StatusNotFound, "not found"), nil
		}
	})}

	writeJiraTestConfig(t, repo, "https://example.atlassian.net", "ABC")
	t.Setenv("AJ_JIRA_EMAIL", "agent@example.com")
	t.Setenv("AJ_JIRA_API_TOKEN", "secret")

	result, err := SearchJiraIssues(SearchJiraIssuesOptions{
		RepoPath: repo,
		Query:    "cache invalidation",
		Limit:    5,
	})
	if err != nil {
		t.Fatalf("search jira issues: %v", err)
	}
	if !strings.Contains(capturedJQL, `project = "ABC"`) || !strings.Contains(capturedJQL, `text ~ "cache" AND text ~ "invalidation"`) {
		t.Fatalf("unexpected captured jql: %s", capturedJQL)
	}
	if result.Project != "ABC" || result.Query != "cache invalidation" || len(result.Issues) != 1 {
		t.Fatalf("unexpected search result: %#v", result)
	}
}

func TestLinkJiraIssueRequiresReplaceAndUnlinkHonorsForce(t *testing.T) {
	repo := t.TempDir()
	if _, err := InitRepo(InitOptions{RepoPath: repo}); err != nil {
		t.Fatalf("init repo: %v", err)
	}

	oldClient := jira.DefaultHTTPClient
	defer func() { jira.DefaultHTTPClient = oldClient }()

	jira.DefaultHTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("agent@example.com:secret"))
		if r.Header.Get("Authorization") != wantAuth {
			return testJSONResponse(http.StatusUnauthorized, "unauthorized"), nil
		}
		switch {
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/rest/api/3/issue/ABC-123"):
			payload, _ := json.Marshal(map[string]any{
				"key":  "ABC-123",
				"self": "https://example.atlassian.net/rest/api/3/issue/10000",
				"fields": map[string]any{
					"summary":   "First linked issue",
					"issuetype": map[string]any{"name": "Task"},
					"priority":  map[string]any{"name": "Medium"},
					"status":    map[string]any{"name": "To Do"},
					"updated":   "2026-03-13T12:00:00.000+0000",
				},
			})
			return testJSONResponse(http.StatusOK, string(payload)), nil
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/rest/api/3/issue/ABC-456"):
			payload, _ := json.Marshal(map[string]any{
				"key":  "ABC-456",
				"self": "https://example.atlassian.net/rest/api/3/issue/10001",
				"fields": map[string]any{
					"summary":   "Second linked issue",
					"issuetype": map[string]any{"name": "Task"},
					"priority":  map[string]any{"name": "Medium"},
					"status":    map[string]any{"name": "To Do"},
					"updated":   "2026-03-13T12:05:00.000+0000",
				},
			})
			return testJSONResponse(http.StatusOK, string(payload)), nil
		default:
			return testJSONResponse(http.StatusNotFound, "not found"), nil
		}
	})}

	writeJiraTestConfig(t, repo, "https://example.atlassian.net", "ABC")
	t.Setenv("AJ_JIRA_EMAIL", "agent@example.com")
	t.Setenv("AJ_JIRA_API_TOKEN", "secret")

	item, err := CreateItem(CreateItemOptions{
		RepoPath:   repo,
		Kind:       domain.KindTask,
		Title:      "Relink safety",
		Goal:       "Protect existing Jira links",
		NextAction: "Link to the first Jira issue",
		Priority:   1,
	})
	if err != nil {
		t.Fatalf("create item: %v", err)
	}

	if _, err := LinkJiraIssue(LinkJiraIssueOptions{
		RepoPath: repo,
		ItemID:   item.ID,
		IssueKey: "ABC-123",
	}); err != nil {
		t.Fatalf("link jira issue: %v", err)
	}

	if _, err := LinkJiraIssue(LinkJiraIssueOptions{
		RepoPath: repo,
		ItemID:   item.ID,
		IssueKey: "ABC-456",
	}); err == nil || !strings.Contains(err.Error(), "--replace") {
		t.Fatalf("expected relink without --replace to fail, got %v", err)
	}

	replaced, err := LinkJiraIssue(LinkJiraIssueOptions{
		RepoPath: repo,
		ItemID:   item.ID,
		IssueKey: "ABC-456",
		Replace:  true,
	})
	if err != nil {
		t.Fatalf("relink jira issue with --replace: %v", err)
	}
	if replaced.Item.Jira == nil || replaced.Item.Jira.Key != "ABC-456" {
		t.Fatalf("expected replaced jira key ABC-456, got %#v", replaced.Item.Jira)
	}

	if _, err := UnlinkJiraIssue(UnlinkJiraIssueOptions{
		RepoPath: repo,
		ItemID:   replaced.Item.ID,
	}); err == nil || !strings.Contains(err.Error(), "sync_state=dirty_local") {
		t.Fatalf("expected unlink without force to fail on dirty link, got %v", err)
	}

	unlinked, err := UnlinkJiraIssue(UnlinkJiraIssueOptions{
		RepoPath: repo,
		ItemID:   replaced.Item.ID,
		Force:    true,
	})
	if err != nil {
		t.Fatalf("force unlink jira issue: %v", err)
	}
	if unlinked.Jira != nil {
		t.Fatalf("expected jira link to be cleared, got %#v", unlinked.Jira)
	}

	relinked, err := LinkJiraIssue(LinkJiraIssueOptions{
		RepoPath: repo,
		ItemID:   item.ID,
		IssueKey: "ABC-456",
	})
	if err != nil {
		t.Fatalf("link second jira issue after unlink: %v", err)
	}
	if relinked.Item.Jira == nil || relinked.Item.Jira.Key != "ABC-456" {
		t.Fatalf("expected relinked jira key ABC-456, got %#v", relinked.Item.Jira)
	}
}

func TestCommentJiraIssue(t *testing.T) {
	repo := t.TempDir()
	if _, err := InitRepo(InitOptions{RepoPath: repo}); err != nil {
		t.Fatalf("init repo: %v", err)
	}

	oldClient := jira.DefaultHTTPClient
	defer func() { jira.DefaultHTTPClient = oldClient }()

	commentCalled := false
	jira.DefaultHTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("agent@example.com:secret"))
		if r.Header.Get("Authorization") != wantAuth {
			return testJSONResponse(http.StatusUnauthorized, "unauthorized"), nil
		}
		switch {
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/rest/api/3/issue/ABC-123"):
			payload, _ := json.Marshal(map[string]any{
				"key":  "ABC-123",
				"self": "https://example.atlassian.net/rest/api/3/issue/10000",
				"fields": map[string]any{
					"summary":   "Linked Jira summary",
					"issuetype": map[string]any{"name": "Task"},
					"priority":  map[string]any{"name": "Medium"},
					"status":    map[string]any{"name": "To Do"},
					"updated":   "2026-03-13T12:00:00.000+0000",
				},
			})
			return testJSONResponse(http.StatusOK, string(payload)), nil
		case r.Method == http.MethodPost && r.URL.Path == "/rest/api/3/issue/ABC-123/comment":
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode comment payload: %v", err)
			}
			commentBody := payload["body"].(map[string]any)
			if commentBody["type"] != "doc" {
				t.Fatalf("unexpected comment body: %#v", payload)
			}
			commentCalled = true
			return testJSONResponse(http.StatusCreated, `{"id":"10001"}`), nil
		default:
			return testJSONResponse(http.StatusNotFound, "not found"), nil
		}
	})}

	writeJiraTestConfig(t, repo, "https://example.atlassian.net", "ABC")
	t.Setenv("AJ_JIRA_EMAIL", "agent@example.com")
	t.Setenv("AJ_JIRA_API_TOKEN", "secret")

	item, err := CreateItem(CreateItemOptions{
		RepoPath:   repo,
		Kind:       domain.KindTask,
		Title:      "Comment me",
		Goal:       "Send a Jira milestone comment",
		NextAction: "Link the item",
		Priority:   1,
	})
	if err != nil {
		t.Fatalf("create item: %v", err)
	}

	linked, err := LinkJiraIssue(LinkJiraIssueOptions{
		RepoPath: repo,
		ItemID:   item.ID,
		IssueKey: "ABC-123",
	})
	if err != nil {
		t.Fatalf("link jira issue: %v", err)
	}

	commented, err := CommentJiraIssue(CommentJiraIssueOptions{
		RepoPath: repo,
		ItemID:   linked.Item.ID,
		Summary:  "Ready for review",
	})
	if err != nil {
		t.Fatalf("comment jira issue: %v", err)
	}
	if !commentCalled {
		t.Fatalf("expected jira comment API call")
	}
	if commented.Jira == nil || commented.Jira.SyncState != "clean" {
		t.Fatalf("expected clean jira state after comment, got %#v", commented.Jira)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func testJSONResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func writeJiraTestConfig(t *testing.T, repo, baseURL, project string) {
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
