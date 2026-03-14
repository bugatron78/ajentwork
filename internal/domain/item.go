package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type ItemKind string

const (
	KindBug     ItemKind = "bug"
	KindFeature ItemKind = "feature"
	KindTask    ItemKind = "task"
	KindSpike   ItemKind = "spike"
	KindEpic    ItemKind = "epic"
)

type Status string

const (
	StatusTodo       Status = "todo"
	StatusInProgress Status = "in_progress"
	StatusBlocked    Status = "blocked"
	StatusInReview   Status = "in_review"
	StatusDone       Status = "done"
	StatusCanceled   Status = "canceled"
)

func ParseStatus(value string) (Status, error) {
	switch Status(strings.TrimSpace(value)) {
	case StatusTodo, StatusInProgress, StatusBlocked, StatusInReview, StatusDone, StatusCanceled:
		return Status(value), nil
	default:
		return "", fmt.Errorf("unsupported status %q (expected todo, in_progress, blocked, in_review, done, or canceled)", value)
	}
}

type Item struct {
	ID            string      `json:"id"`
	Kind          ItemKind    `json:"kind"`
	Title         string      `json:"title"`
	Status        Status      `json:"status"`
	Priority      int         `json:"priority"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
	Goal          string      `json:"goal"`
	Summary       string      `json:"summary"`
	NextAction    string      `json:"next_action"`
	Acceptance    []string    `json:"acceptance,omitempty"`
	Constraints   []string    `json:"constraints,omitempty"`
	Risks         []string    `json:"risks,omitempty"`
	RelevantFiles []string    `json:"relevant_files,omitempty"`
	Verification  []string    `json:"verification,omitempty"`
	Checkpoint    *Checkpoint `json:"checkpoint,omitempty"`
	DependsOn     []string    `json:"depends_on,omitempty"`
	Lease         *Lease      `json:"lease,omitempty"`
	Jira          *JiraLink   `json:"jira,omitempty"`
}

type Checkpoint struct {
	Summary   string    `json:"summary"`
	Risks     []string  `json:"risks,omitempty"`
	Verify    []string  `json:"verify,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	Actor     string    `json:"actor"`
}

type Lease struct {
	Owner     string    `json:"owner"`
	ClaimedAt time.Time `json:"claimed_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

type JiraLink struct {
	Key               string     `json:"key"`
	URL               string     `json:"url"`
	SyncMode          string     `json:"sync_mode"`
	SyncState         string     `json:"sync_state"`
	LastSyncedAt      *time.Time `json:"last_synced_at,omitempty"`
	LastRemoteVersion string     `json:"last_remote_version,omitempty"`
}

func ParseItemKind(value string) (ItemKind, error) {
	switch ItemKind(strings.TrimSpace(value)) {
	case KindBug, KindFeature, KindTask, KindSpike, KindEpic:
		return ItemKind(value), nil
	default:
		return "", fmt.Errorf("unsupported kind %q (expected bug, feature, task, spike, or epic)", value)
	}
}

func ValidateNewItemFields(kind ItemKind, title, goal, nextAction string, priority int) error {
	if kind == "" {
		return errors.New("kind is required")
	}
	if strings.TrimSpace(title) == "" {
		return errors.New("title is required")
	}
	if strings.TrimSpace(goal) == "" {
		return errors.New("goal is required")
	}
	if strings.TrimSpace(nextAction) == "" {
		return errors.New("next action is required")
	}
	if priority < 0 || priority > 4 {
		return fmt.Errorf("priority %d is out of range (expected 0 through 4)", priority)
	}
	return nil
}

func StatusRequiresNextAction(status Status) bool {
	switch status {
	case StatusTodo, StatusInProgress, StatusBlocked:
		return true
	default:
		return false
	}
}

func StatusActionable(status Status) bool {
	switch status {
	case StatusTodo, StatusInProgress:
		return true
	default:
		return false
	}
}

func (l Lease) Expired(now time.Time) bool {
	return !l.ExpiresAt.After(now)
}
