package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ajentwork/internal/domain"
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
