package store

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"ajentwork/internal/config"
	"ajentwork/internal/domain"
	"ajentwork/internal/idgen"
	"ajentwork/internal/jira"
)

type CreateItemOptions struct {
	RepoPath    string
	Kind        domain.ItemKind
	Title       string
	Goal        string
	NextAction  string
	Priority    int
}

type UpdateItemOptions struct {
	RepoPath   string
	ItemID     string
	Summary    string
	NextAction *string
	Status     *domain.Status
}

type CompleteItemOptions struct {
	RepoPath string
	ItemID   string
	Summary  string
}

type BlockItemOptions struct {
	RepoPath   string
	ItemID     string
	Summary    string
	OnID       string
	NextAction *string
}

type UnblockItemOptions struct {
	RepoPath   string
	ItemID     string
	Summary    string
	NextAction *string
	Status     *domain.Status
}

type TakeItemOptions struct {
	RepoPath string
	ItemID   string
	Agent    string
	TTL      time.Duration
	Force    bool
}

type ReleaseItemOptions struct {
	RepoPath string
	ItemID   string
}

type HandoffItemOptions struct {
	RepoPath   string
	ItemID     string
	ToAgent    string
	Summary    string
	NextAction *string
	TTL        time.Duration
}

type ReopenItemOptions struct {
	RepoPath   string
	ItemID     string
	Summary    string
	NextAction string
	Status     *domain.Status
}

type LinkDependencyOptions struct {
	RepoPath    string
	ItemID      string
	DependsOnID string
}

type ImportJiraIssueOptions struct {
	RepoPath string
	IssueKey string
}

type ImportJiraIssueResult struct {
	Item          domain.Item `json:"item"`
	AlreadyLinked bool        `json:"already_linked"`
}

type ExportJiraIssueOptions struct {
	RepoPath   string
	ItemID     string
	ProjectKey string
	IssueType  string
}

type ExportJiraIssueResult struct {
	Item          domain.Item `json:"item"`
	AlreadyLinked bool        `json:"already_linked"`
}

type LinkJiraIssueOptions struct {
	RepoPath string
	ItemID   string
	IssueKey string
}

type LinkJiraIssueResult struct {
	Item          domain.Item `json:"item"`
	AlreadyLinked bool        `json:"already_linked"`
}

type SyncJiraIssueOptions struct {
	RepoPath string
	ItemID   string
	DryRun   bool
	Resolve  string
}

type SyncJiraIssueResult struct {
	Item      domain.Item `json:"item"`
	Direction string      `json:"direction"`
	DryRun    bool        `json:"dry_run"`
}

type NextItemResult struct {
	Item      domain.Item `json:"item"`
	Reason    string      `json:"reason"`
	WaitingOn []string    `json:"waiting_on,omitempty"`
}

type InboxEntry struct {
	Item      domain.Item `json:"item"`
	Reason    string      `json:"reason"`
	WaitingOn []string    `json:"waiting_on,omitempty"`
}

type ReadyEntry struct {
	Item   domain.Item `json:"item"`
	Reason string      `json:"reason"`
}

type ChangesOptions struct {
	RepoPath string
	ItemID   string
	Since    *time.Time
	Limit    int
}

type ReadyOptions struct {
	RepoPath string
	Agent    string
}

func CreateItem(opts CreateItemOptions) (domain.Item, error) {
	if err := domain.ValidateNewItemFields(opts.Kind, opts.Title, opts.Goal, opts.NextAction, opts.Priority); err != nil {
		return domain.Item{}, err
	}

	ajDir, err := ensureAJRepo(opts.RepoPath)
	if err != nil {
		return domain.Item{}, err
	}

	var itemID string
	for attempt := 0; attempt < 5; attempt++ {
		itemID, err = idgen.NewItemID()
		if err != nil {
			return domain.Item{}, err
		}
		itemDir := filepath.Join(ajDir, "issues", itemID)
		_, statErr := os.Stat(itemDir)
		if errors.Is(statErr, os.ErrNotExist) {
			break
		}
		if statErr != nil {
			return domain.Item{}, fmt.Errorf("check item directory %s: %w", itemDir, statErr)
		}
		itemID = ""
	}
	if itemID == "" {
		return domain.Item{}, errors.New("failed to allocate a unique item id")
	}

	now := time.Now().UTC().Truncate(time.Second)
	item := domain.Item{
		ID:         itemID,
		Kind:       opts.Kind,
		Title:      strings.TrimSpace(opts.Title),
		Status:     domain.StatusTodo,
		Priority:   opts.Priority,
		CreatedAt:  now,
		UpdatedAt:  now,
		Goal:       strings.TrimSpace(opts.Goal),
		Summary:    "created",
		NextAction: strings.TrimSpace(opts.NextAction),
	}

	itemDir := filepath.Join(ajDir, "issues", item.ID)
	eventsDir := filepath.Join(itemDir, "events")
	if err := os.MkdirAll(eventsDir, 0o755); err != nil {
		return domain.Item{}, fmt.Errorf("create item directories: %w", err)
	}

	metaPath := filepath.Join(itemDir, "meta.toml")
	if err := os.WriteFile(metaPath, []byte(marshalItem(item)), 0o644); err != nil {
		return domain.Item{}, fmt.Errorf("write item metadata: %w", err)
	}

	if err := writeCreatedEvent(eventsDir, item); err != nil {
		return domain.Item{}, err
	}

	return item, nil
}

func GetItem(repoPath, itemID string) (domain.Item, error) {
	ajDir, err := ensureAJRepo(repoPath)
	if err != nil {
		return domain.Item{}, err
	}

	metaPath := filepath.Join(ajDir, "issues", itemID, "meta.toml")
	bytes, err := os.ReadFile(metaPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return domain.Item{}, fmt.Errorf("item %s not found", itemID)
		}
		return domain.Item{}, fmt.Errorf("read item metadata: %w", err)
	}

	item, err := parseItem(string(bytes))
	if err != nil {
		return domain.Item{}, fmt.Errorf("parse item metadata: %w", err)
	}
	return item, nil
}

func UpdateItem(opts UpdateItemOptions) (domain.Item, error) {
	if strings.TrimSpace(opts.ItemID) == "" {
		return domain.Item{}, errors.New("item id is required")
	}
	if strings.TrimSpace(opts.Summary) == "" {
		return domain.Item{}, errors.New("summary is required")
	}

	item, itemDir, err := loadItemForMutation(opts.RepoPath, opts.ItemID)
	if err != nil {
		return domain.Item{}, err
	}
	if item.Status == domain.StatusDone {
		return domain.Item{}, fmt.Errorf("item %s is already done", item.ID)
	}

	item.Summary = strings.TrimSpace(opts.Summary)
	if opts.NextAction != nil {
		item.NextAction = strings.TrimSpace(*opts.NextAction)
	}
	if opts.Status != nil {
		switch *opts.Status {
		case domain.StatusDone:
			return domain.Item{}, errors.New("use `aj done <id> --summary ...` to complete an item")
		case domain.StatusCanceled:
			return domain.Item{}, errors.New("status canceled is not supported yet")
		default:
			item.Status = *opts.Status
		}
	}
	if domain.StatusRequiresNextAction(item.Status) && strings.TrimSpace(item.NextAction) == "" {
		return domain.Item{}, fmt.Errorf("status %s requires a next action", item.Status)
	}
	markJiraDirtyLocal(&item)

	item.UpdatedAt = time.Now().UTC().Truncate(time.Second)
	if err := persistItemMutation(itemDir, item, "updated", "system", item.Summary); err != nil {
		return domain.Item{}, err
	}

	return item, nil
}

func CompleteItem(opts CompleteItemOptions) (domain.Item, error) {
	if strings.TrimSpace(opts.ItemID) == "" {
		return domain.Item{}, errors.New("item id is required")
	}
	if strings.TrimSpace(opts.Summary) == "" {
		return domain.Item{}, errors.New("summary is required")
	}

	item, itemDir, err := loadItemForMutation(opts.RepoPath, opts.ItemID)
	if err != nil {
		return domain.Item{}, err
	}

	item.Status = domain.StatusDone
	item.Summary = strings.TrimSpace(opts.Summary)
	item.NextAction = ""
	item.Lease = nil
	markJiraDirtyLocal(&item)
	item.UpdatedAt = time.Now().UTC().Truncate(time.Second)

	if err := persistItemMutation(itemDir, item, "done", "system", item.Summary); err != nil {
		return domain.Item{}, err
	}

	return item, nil
}

func BlockItem(opts BlockItemOptions) (domain.Item, error) {
	if strings.TrimSpace(opts.ItemID) == "" {
		return domain.Item{}, errors.New("item id is required")
	}
	if strings.TrimSpace(opts.Summary) == "" {
		return domain.Item{}, errors.New("summary is required")
	}
	if strings.TrimSpace(opts.OnID) == opts.ItemID {
		return domain.Item{}, errors.New("an item cannot depend on itself")
	}

	if dependencyID := strings.TrimSpace(opts.OnID); dependencyID != "" {
		if _, err := GetItem(opts.RepoPath, dependencyID); err != nil {
			return domain.Item{}, fmt.Errorf("dependency item %s not found", dependencyID)
		}
	}

	item, itemDir, err := loadItemForMutation(opts.RepoPath, opts.ItemID)
	if err != nil {
		return domain.Item{}, err
	}
	if item.Status == domain.StatusDone || item.Status == domain.StatusCanceled {
		return domain.Item{}, fmt.Errorf("item %s is already complete; use `aj reopen` first", item.ID)
	}

	item.Status = domain.StatusBlocked
	item.Summary = strings.TrimSpace(opts.Summary)
	if opts.NextAction != nil {
		item.NextAction = strings.TrimSpace(*opts.NextAction)
	} else if dependencyID := strings.TrimSpace(opts.OnID); dependencyID != "" {
		item.NextAction = fmt.Sprintf("Wait for %s", dependencyID)
	}

	if dependencyID := strings.TrimSpace(opts.OnID); dependencyID != "" {
		alreadyLinked := false
		for _, depID := range item.DependsOn {
			if depID == dependencyID {
				alreadyLinked = true
				break
			}
		}
		if !alreadyLinked {
			item.DependsOn = append(item.DependsOn, dependencyID)
			sort.Strings(item.DependsOn)
		}
	}

	if domain.StatusRequiresNextAction(item.Status) && strings.TrimSpace(item.NextAction) == "" {
		return domain.Item{}, fmt.Errorf("status %s requires a next action", item.Status)
	}
	markJiraDirtyLocal(&item)

	item.UpdatedAt = time.Now().UTC().Truncate(time.Second)
	if err := persistItemMutation(itemDir, item, "blocked", "system", item.Summary); err != nil {
		return domain.Item{}, err
	}

	return item, nil
}

func UnblockItem(opts UnblockItemOptions) (domain.Item, error) {
	if strings.TrimSpace(opts.ItemID) == "" {
		return domain.Item{}, errors.New("item id is required")
	}
	if strings.TrimSpace(opts.Summary) == "" {
		return domain.Item{}, errors.New("summary is required")
	}

	item, itemDir, err := loadItemForMutation(opts.RepoPath, opts.ItemID)
	if err != nil {
		return domain.Item{}, err
	}
	if item.Status != domain.StatusBlocked {
		return domain.Item{}, fmt.Errorf("item %s is not blocked", item.ID)
	}

	targetStatus := domain.StatusTodo
	if opts.Status != nil {
		switch *opts.Status {
		case domain.StatusBlocked:
			return domain.Item{}, errors.New("use `aj block <id> ...` to keep an item blocked")
		case domain.StatusDone:
			return domain.Item{}, errors.New("use `aj done <id> --summary ...` to complete an item")
		case domain.StatusCanceled:
			return domain.Item{}, errors.New("status canceled is not supported yet")
		default:
			targetStatus = *opts.Status
		}
	}

	item.Status = targetStatus
	item.Summary = strings.TrimSpace(opts.Summary)
	if opts.NextAction != nil {
		item.NextAction = strings.TrimSpace(*opts.NextAction)
	}
	if domain.StatusRequiresNextAction(item.Status) && strings.TrimSpace(item.NextAction) == "" {
		return domain.Item{}, fmt.Errorf("status %s requires a next action", item.Status)
	}
	markJiraDirtyLocal(&item)

	item.UpdatedAt = time.Now().UTC().Truncate(time.Second)
	if err := persistItemMutation(itemDir, item, "unblocked", "system", item.Summary); err != nil {
		return domain.Item{}, err
	}

	return item, nil
}

func TakeItem(opts TakeItemOptions) (domain.Item, error) {
	if strings.TrimSpace(opts.ItemID) == "" {
		return domain.Item{}, errors.New("item id is required")
	}
	if strings.TrimSpace(opts.Agent) == "" {
		return domain.Item{}, errors.New("agent is required")
	}
	if opts.TTL <= 0 {
		return domain.Item{}, errors.New("ttl must be greater than zero")
	}

	item, itemDir, err := loadItemForMutation(opts.RepoPath, opts.ItemID)
	if err != nil {
		return domain.Item{}, err
	}

	now := time.Now().UTC().Truncate(time.Second)
	if item.Lease != nil && !item.Lease.Expired(now) && item.Lease.Owner != strings.TrimSpace(opts.Agent) && !opts.Force {
		return domain.Item{}, fmt.Errorf("item %s is currently leased by %s until %s", item.ID, item.Lease.Owner, item.Lease.ExpiresAt.Format(time.RFC3339))
	}

	item.Lease = &domain.Lease{
		Owner:     strings.TrimSpace(opts.Agent),
		ClaimedAt: now,
		ExpiresAt: now.Add(opts.TTL).Truncate(time.Second),
	}
	item.UpdatedAt = now
	item.Summary = fmt.Sprintf("claimed by %s", item.Lease.Owner)

	if err := persistItemMutation(itemDir, item, "claimed", item.Lease.Owner, item.Summary); err != nil {
		return domain.Item{}, err
	}
	return item, nil
}

func ReleaseItem(opts ReleaseItemOptions) (domain.Item, error) {
	if strings.TrimSpace(opts.ItemID) == "" {
		return domain.Item{}, errors.New("item id is required")
	}

	item, itemDir, err := loadItemForMutation(opts.RepoPath, opts.ItemID)
	if err != nil {
		return domain.Item{}, err
	}
	if item.Lease == nil {
		return domain.Item{}, fmt.Errorf("item %s does not have an active lease", item.ID)
	}

	owner := item.Lease.Owner
	item.Lease = nil
	item.UpdatedAt = time.Now().UTC().Truncate(time.Second)
	item.Summary = fmt.Sprintf("released by %s", owner)

	if err := persistItemMutation(itemDir, item, "released", owner, item.Summary); err != nil {
		return domain.Item{}, err
	}
	return item, nil
}

func HandoffItem(opts HandoffItemOptions) (domain.Item, error) {
	if strings.TrimSpace(opts.ItemID) == "" {
		return domain.Item{}, errors.New("item id is required")
	}
	if strings.TrimSpace(opts.ToAgent) == "" {
		return domain.Item{}, errors.New("destination agent is required")
	}
	if strings.TrimSpace(opts.Summary) == "" {
		return domain.Item{}, errors.New("summary is required")
	}
	if opts.TTL <= 0 {
		return domain.Item{}, errors.New("ttl must be greater than zero")
	}

	item, itemDir, err := loadItemForMutation(opts.RepoPath, opts.ItemID)
	if err != nil {
		return domain.Item{}, err
	}
	if item.Status == domain.StatusDone || item.Status == domain.StatusCanceled {
		return domain.Item{}, fmt.Errorf("item %s is already complete", item.ID)
	}

	now := time.Now().UTC().Truncate(time.Second)
	item.Lease = &domain.Lease{
		Owner:     strings.TrimSpace(opts.ToAgent),
		ClaimedAt: now,
		ExpiresAt: now.Add(opts.TTL).Truncate(time.Second),
	}
	item.Summary = strings.TrimSpace(opts.Summary)
	if opts.NextAction != nil {
		item.NextAction = strings.TrimSpace(*opts.NextAction)
	}
	if domain.StatusRequiresNextAction(item.Status) && strings.TrimSpace(item.NextAction) == "" {
		return domain.Item{}, fmt.Errorf("status %s requires a next action", item.Status)
	}
	markJiraDirtyLocal(&item)

	item.UpdatedAt = now
	if err := persistItemMutation(itemDir, item, "handoff", "system", item.Summary); err != nil {
		return domain.Item{}, err
	}
	return item, nil
}

func ReopenItem(opts ReopenItemOptions) (domain.Item, error) {
	if strings.TrimSpace(opts.ItemID) == "" {
		return domain.Item{}, errors.New("item id is required")
	}
	if strings.TrimSpace(opts.Summary) == "" {
		return domain.Item{}, errors.New("summary is required")
	}
	if strings.TrimSpace(opts.NextAction) == "" {
		return domain.Item{}, errors.New("next action is required")
	}

	item, itemDir, err := loadItemForMutation(opts.RepoPath, opts.ItemID)
	if err != nil {
		return domain.Item{}, err
	}
	if item.Status != domain.StatusDone && item.Status != domain.StatusCanceled {
		return domain.Item{}, fmt.Errorf("item %s is not done or canceled", item.ID)
	}

	targetStatus := domain.StatusTodo
	if opts.Status != nil {
		switch *opts.Status {
		case domain.StatusDone:
			return domain.Item{}, errors.New("use `aj done <id> --summary ...` to complete an item")
		case domain.StatusCanceled:
			return domain.Item{}, errors.New("status canceled is not supported yet")
		default:
			targetStatus = *opts.Status
		}
	}

	item.Status = targetStatus
	item.Summary = strings.TrimSpace(opts.Summary)
	item.NextAction = strings.TrimSpace(opts.NextAction)
	markJiraDirtyLocal(&item)
	item.UpdatedAt = time.Now().UTC().Truncate(time.Second)
	if err := persistItemMutation(itemDir, item, "reopened", "system", item.Summary); err != nil {
		return domain.Item{}, err
	}
	return item, nil
}

func LinkDependency(opts LinkDependencyOptions) (domain.Item, error) {
	if strings.TrimSpace(opts.ItemID) == "" {
		return domain.Item{}, errors.New("item id is required")
	}
	if strings.TrimSpace(opts.DependsOnID) == "" {
		return domain.Item{}, errors.New("dependency id is required")
	}
	if opts.ItemID == opts.DependsOnID {
		return domain.Item{}, errors.New("an item cannot depend on itself")
	}

	if _, err := GetItem(opts.RepoPath, opts.DependsOnID); err != nil {
		return domain.Item{}, fmt.Errorf("dependency item %s not found", opts.DependsOnID)
	}

	item, itemDir, err := loadItemForMutation(opts.RepoPath, opts.ItemID)
	if err != nil {
		return domain.Item{}, err
	}

	for _, depID := range item.DependsOn {
		if depID == opts.DependsOnID {
			return item, nil
		}
	}

	item.DependsOn = append(item.DependsOn, opts.DependsOnID)
	sort.Strings(item.DependsOn)
	item.UpdatedAt = time.Now().UTC().Truncate(time.Second)
	item.Summary = fmt.Sprintf("linked dependency on %s", opts.DependsOnID)

	if err := persistItemMutation(itemDir, item, "linked_dependency", "system", item.Summary); err != nil {
		return domain.Item{}, err
	}
	return item, nil
}

func ImportJiraIssue(opts ImportJiraIssueOptions) (ImportJiraIssueResult, error) {
	if strings.TrimSpace(opts.IssueKey) == "" {
		return ImportJiraIssueResult{}, errors.New("jira issue key is required")
	}

	if existing, ok, err := findItemByJiraKey(opts.RepoPath, opts.IssueKey); err != nil {
		return ImportJiraIssueResult{}, err
	} else if ok {
		return ImportJiraIssueResult{Item: existing, AlreadyLinked: true}, nil
	}

	settings, err := config.ResolveJiraSettings(opts.RepoPath)
	if err != nil {
		return ImportJiraIssueResult{}, err
	}
	client := jira.Client{
		BaseURL:    settings.BaseURL,
		Email:      settings.Email,
		APIToken:   settings.APIToken,
		HTTPClient: jira.DefaultHTTPClient,
	}

	remoteIssue, err := client.GetIssue(context.Background(), opts.IssueKey)
	if err != nil {
		return ImportJiraIssueResult{}, err
	}

	item, err := createImportedJiraItem(opts.RepoPath, remoteIssue, settings.StatusMap)
	if err != nil {
		return ImportJiraIssueResult{}, err
	}

	return ImportJiraIssueResult{Item: item}, nil
}

func ExportJiraIssue(opts ExportJiraIssueOptions) (ExportJiraIssueResult, error) {
	if strings.TrimSpace(opts.ItemID) == "" {
		return ExportJiraIssueResult{}, errors.New("item id is required")
	}

	item, itemDir, err := loadItemForMutation(opts.RepoPath, opts.ItemID)
	if err != nil {
		return ExportJiraIssueResult{}, err
	}
	if item.Jira != nil && item.Jira.Key != "" {
		return ExportJiraIssueResult{Item: item, AlreadyLinked: true}, nil
	}

	settings, err := config.ResolveJiraSettings(opts.RepoPath)
	if err != nil {
		return ExportJiraIssueResult{}, err
	}
	projectKey := strings.TrimSpace(opts.ProjectKey)
	if projectKey == "" {
		projectKey = settings.Project
	}
	if projectKey == "" {
		return ExportJiraIssueResult{}, errors.New("jira project is required; set [jira].project or pass --project")
	}
	issueType := strings.TrimSpace(opts.IssueType)
	if issueType == "" {
		issueType = jiraIssueTypeForItem(item.Kind)
	}

	client := jira.Client{
		BaseURL:    settings.BaseURL,
		Email:      settings.Email,
		APIToken:   settings.APIToken,
		HTTPClient: jira.DefaultHTTPClient,
	}
	created, err := client.CreateIssue(context.Background(), jira.CreateIssueInput{
		ProjectKey:  projectKey,
		IssueType:   issueType,
		Summary:     item.Title,
		Description: jiraDescriptionForItem(item),
	})
	if err != nil {
		return ExportJiraIssueResult{}, err
	}

	now := time.Now().UTC().Truncate(time.Second)
	item.Jira = &domain.JiraLink{
		Key:          created.Key,
		URL:          created.URL,
		SyncMode:     "export_only",
		SyncState:    "clean",
		LastSyncedAt: &now,
	}
	item.UpdatedAt = now
	item.Summary = fmt.Sprintf("linked to Jira %s", created.Key)

	if err := persistItemMutation(itemDir, item, "linked_external", "system", item.Summary); err != nil {
		return ExportJiraIssueResult{}, err
	}
	return ExportJiraIssueResult{Item: item}, nil
}

func LinkJiraIssue(opts LinkJiraIssueOptions) (LinkJiraIssueResult, error) {
	if strings.TrimSpace(opts.ItemID) == "" {
		return LinkJiraIssueResult{}, errors.New("item id is required")
	}
	if strings.TrimSpace(opts.IssueKey) == "" {
		return LinkJiraIssueResult{}, errors.New("jira issue key is required")
	}

	item, itemDir, err := loadItemForMutation(opts.RepoPath, opts.ItemID)
	if err != nil {
		return LinkJiraIssueResult{}, err
	}
	if item.Jira != nil && strings.EqualFold(item.Jira.Key, strings.TrimSpace(opts.IssueKey)) {
		return LinkJiraIssueResult{Item: item, AlreadyLinked: true}, nil
	}
	if existing, ok, err := findItemByJiraKey(opts.RepoPath, opts.IssueKey); err != nil {
		return LinkJiraIssueResult{}, err
	} else if ok && existing.ID != item.ID {
		return LinkJiraIssueResult{}, fmt.Errorf("jira issue %s is already linked to %s", opts.IssueKey, existing.ID)
	}

	settings, err := config.ResolveJiraSettings(opts.RepoPath)
	if err != nil {
		return LinkJiraIssueResult{}, err
	}
	client := jira.Client{
		BaseURL:    settings.BaseURL,
		Email:      settings.Email,
		APIToken:   settings.APIToken,
		HTTPClient: jira.DefaultHTTPClient,
	}
	remoteIssue, err := client.GetIssue(context.Background(), opts.IssueKey)
	if err != nil {
		return LinkJiraIssueResult{}, err
	}

	item.Jira = &domain.JiraLink{
		Key:               remoteIssue.Key,
		URL:               remoteIssue.URL,
		SyncMode:          "bidirectional",
		SyncState:         "dirty_local",
		LastRemoteVersion: remoteIssue.Updated,
	}
	item.UpdatedAt = time.Now().UTC().Truncate(time.Second)
	item.Summary = fmt.Sprintf("linked to Jira %s", remoteIssue.Key)
	if err := persistItemMutation(itemDir, item, "linked_external", "system", item.Summary); err != nil {
		return LinkJiraIssueResult{}, err
	}
	return LinkJiraIssueResult{Item: item}, nil
}

func SyncJiraIssue(opts SyncJiraIssueOptions) (SyncJiraIssueResult, error) {
	if strings.TrimSpace(opts.ItemID) == "" {
		return SyncJiraIssueResult{}, errors.New("item id is required")
	}
	if opts.Resolve != "" && opts.Resolve != "keep-local" && opts.Resolve != "keep-remote" {
		return SyncJiraIssueResult{}, errors.New("resolve must be keep-local or keep-remote")
	}

	item, itemDir, err := loadItemForMutation(opts.RepoPath, opts.ItemID)
	if err != nil {
		return SyncJiraIssueResult{}, err
	}
	if item.Jira == nil || strings.TrimSpace(item.Jira.Key) == "" {
		return SyncJiraIssueResult{}, fmt.Errorf("item %s is not linked to Jira", item.ID)
	}

	settings, err := config.ResolveJiraSettings(opts.RepoPath)
	if err != nil {
		return SyncJiraIssueResult{}, err
	}
	client := jira.Client{
		BaseURL:    settings.BaseURL,
		Email:      settings.Email,
		APIToken:   settings.APIToken,
		HTTPClient: jira.DefaultHTTPClient,
	}
	remoteIssue, err := client.GetIssue(context.Background(), item.Jira.Key)
	if err != nil {
		return SyncJiraIssueResult{}, err
	}

	localDirty := jiraLocalDirty(item)
	remoteDirty := jiraRemoteDirty(item, remoteIssue)
	direction := "noop"

	switch {
	case localDirty && remoteDirty && opts.Resolve == "":
		if !opts.DryRun {
			item.Jira.SyncState = "conflict"
			item.UpdatedAt = time.Now().UTC().Truncate(time.Second)
			if err := persistItemMutationWithEventSummary(itemDir, item, "synced", "system", fmt.Sprintf("jira sync conflict detected for %s", item.Jira.Key)); err != nil {
				return SyncJiraIssueResult{}, err
			}
		}
		return SyncJiraIssueResult{}, fmt.Errorf("jira sync conflict for %s; rerun with --resolve keep-local or --resolve keep-remote", item.Jira.Key)
	case localDirty && remoteDirty && opts.Resolve == "keep-local":
		direction = "push"
	case localDirty && remoteDirty && opts.Resolve == "keep-remote":
		direction = "pull"
	case localDirty:
		direction = "push"
	case remoteDirty:
		direction = "pull"
	}

	if opts.DryRun {
		itemCopy := item
		if itemCopy.Jira != nil {
			itemCopy.Jira.SyncState = jiraSyncStateForDirection(direction)
		}
		return SyncJiraIssueResult{
			Item:      itemCopy,
			Direction: direction,
			DryRun:    true,
		}, nil
	}

	switch direction {
	case "push":
		if err := client.UpdateIssue(context.Background(), item.Jira.Key, jira.UpdateIssueInput{
			Summary:     item.Title,
			Description: jiraDescriptionForItem(item),
		}); err != nil {
			return SyncJiraIssueResult{}, err
		}
		remoteIssue, err = client.GetIssue(context.Background(), item.Jira.Key)
		if err != nil {
			return SyncJiraIssueResult{}, err
		}
		now := time.Now().UTC().Truncate(time.Second)
		item.Jira.SyncState = "clean"
		item.Jira.LastSyncedAt = &now
		item.Jira.LastRemoteVersion = remoteIssue.Updated
		item.Jira.URL = remoteIssue.URL
		item.UpdatedAt = now
		if err := persistItemMutationWithEventSummary(itemDir, item, "synced", "system", fmt.Sprintf("synced local changes to Jira %s", item.Jira.Key)); err != nil {
			return SyncJiraIssueResult{}, err
		}
	case "pull":
		item.Title = defaultJiraTitle(remoteIssue)
		item.Goal = defaultJiraGoal(remoteIssue)
		item.Priority = mapJiraPriority(remoteIssue.Priority)
		item.Status = mapJiraStatus(remoteIssue.Status, settings.StatusMap)
		item.Jira.SyncState = "clean"
		now := time.Now().UTC().Truncate(time.Second)
		item.Jira.LastSyncedAt = &now
		item.Jira.LastRemoteVersion = remoteIssue.Updated
		item.Jira.URL = remoteIssue.URL
		item.UpdatedAt = now
		if item.NextAction == "" && domain.StatusRequiresNextAction(item.Status) {
			item.NextAction = importedNextAction(item.Status, item.Jira.Key)
		}
		if err := persistItemMutationWithEventSummary(itemDir, item, "synced", "system", fmt.Sprintf("synced Jira %s into local item", item.Jira.Key)); err != nil {
			return SyncJiraIssueResult{}, err
		}
	default:
		item.Jira.SyncState = "clean"
		now := time.Now().UTC().Truncate(time.Second)
		item.Jira.LastSyncedAt = &now
		item.Jira.LastRemoteVersion = remoteIssue.Updated
		item.Jira.URL = remoteIssue.URL
	}

	return SyncJiraIssueResult{
		Item:      item,
		Direction: direction,
		DryRun:    false,
	}, nil
}

func ListItems(repoPath string) ([]domain.Item, error) {
	ajDir, err := ensureAJRepo(repoPath)
	if err != nil {
		return nil, err
	}

	issuesDir := filepath.Join(ajDir, "issues")
	entries, err := os.ReadDir(issuesDir)
	if err != nil {
		return nil, fmt.Errorf("read issues directory: %w", err)
	}

	items := make([]domain.Item, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		item, err := GetItem(repoPath, entry.Name())
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].Priority == items[j].Priority {
			if items[i].UpdatedAt.Equal(items[j].UpdatedAt) {
				return items[i].ID < items[j].ID
			}
			return items[i].UpdatedAt.After(items[j].UpdatedAt)
		}
		return items[i].Priority < items[j].Priority
	})

	return items, nil
}

func RecommendNext(repoPath, agent string) (NextItemResult, error) {
	items, err := ListItems(repoPath)
	if err != nil {
		return NextItemResult{}, err
	}

	now := time.Now().UTC()
	agent = strings.TrimSpace(agent)
	itemMap := indexItems(items)

	if agent != "" {
		owned := filterItems(items, func(item domain.Item) bool {
			return item.Lease != nil && item.Lease.Owner == agent && !item.Lease.Expired(now) && domain.StatusActionable(item.Status) && len(unmetDependencies(item, itemMap)) == 0
		})
		if len(owned) > 0 {
			return NextItemResult{
				Item:   owned[0],
				Reason: fmt.Sprintf("currently leased to %s", agent),
			}, nil
		}
	}

	available := filterItems(items, func(item domain.Item) bool {
		if !domain.StatusActionable(item.Status) {
			return false
		}
		if len(unmetDependencies(item, itemMap)) > 0 {
			return false
		}
		if item.Lease == nil {
			return true
		}
		return item.Lease.Expired(now)
	})
	if len(available) == 0 {
		waiting := filterItems(items, func(item domain.Item) bool {
			return item.Status != domain.StatusDone && item.Status != domain.StatusCanceled && len(unmetDependencies(item, itemMap)) > 0
		})
		if len(waiting) == 0 {
			return NextItemResult{}, errors.New("no actionable items available")
		}
		return NextItemResult{
			Item:      waiting[0],
			Reason:    "no ready items; next item is waiting on dependencies",
			WaitingOn: unmetDependencies(waiting[0], itemMap),
		}, nil
	}

	reason := "highest-priority available actionable item"
	if available[0].Lease != nil && available[0].Lease.Expired(now) {
		reason = fmt.Sprintf("stale lease expired for %s", available[0].Lease.Owner)
	}

	return NextItemResult{
		Item:   available[0],
		Reason: reason,
	}, nil
}

func Inbox(repoPath, agent string) ([]InboxEntry, error) {
	items, err := ListItems(repoPath)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	agent = strings.TrimSpace(agent)
	itemMap := indexItems(items)
	results := make([]InboxEntry, 0)

	for _, item := range items {
		if item.Status == domain.StatusDone || item.Status == domain.StatusCanceled {
			continue
		}

		waitingOn := unmetDependencies(item, itemMap)

		switch {
		case len(waitingOn) > 0:
			results = append(results, InboxEntry{Item: item, Reason: "waiting", WaitingOn: waitingOn})
		case item.Lease != nil && item.Lease.Expired(now):
			results = append(results, InboxEntry{Item: item, Reason: "stale"})
		case agent != "" && item.Lease != nil && item.Lease.Owner == agent && !item.Lease.Expired(now):
			results = append(results, InboxEntry{Item: item, Reason: "owned"})
		case item.Lease == nil && domain.StatusActionable(item.Status):
			results = append(results, InboxEntry{Item: item, Reason: "available"})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		priorityRank := func(reason string) int {
			switch reason {
			case "owned":
				return 0
			case "stale":
				return 1
			case "waiting":
				return 2
			case "available":
				return 3
			default:
				return 4
			}
		}

		leftRank := priorityRank(results[i].Reason)
		rightRank := priorityRank(results[j].Reason)
		if leftRank == rightRank {
			if results[i].Item.Priority == results[j].Item.Priority {
				if results[i].Item.UpdatedAt.Equal(results[j].Item.UpdatedAt) {
					return results[i].Item.ID < results[j].Item.ID
				}
				return results[i].Item.UpdatedAt.After(results[j].Item.UpdatedAt)
			}
			return results[i].Item.Priority < results[j].Item.Priority
		}
		return leftRank < rightRank
	})

	return results, nil
}

func Ready(opts ReadyOptions) ([]ReadyEntry, error) {
	items, err := ListItems(opts.RepoPath)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	agent := strings.TrimSpace(opts.Agent)
	itemMap := indexItems(items)
	results := make([]ReadyEntry, 0)

	for _, item := range items {
		if !domain.StatusActionable(item.Status) {
			continue
		}
		if len(unmetDependencies(item, itemMap)) > 0 {
			continue
		}

		switch {
		case item.Lease == nil:
			results = append(results, ReadyEntry{Item: item, Reason: "available"})
		case item.Lease.Expired(now):
			results = append(results, ReadyEntry{Item: item, Reason: "stale"})
		case agent != "" && item.Lease.Owner == agent:
			results = append(results, ReadyEntry{Item: item, Reason: "owned"})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		rank := func(reason string) int {
			switch reason {
			case "owned":
				return 0
			case "stale":
				return 1
			case "available":
				return 2
			default:
				return 3
			}
		}
		leftRank := rank(results[i].Reason)
		rightRank := rank(results[j].Reason)
		if leftRank == rightRank {
			if results[i].Item.Priority == results[j].Item.Priority {
				if results[i].Item.UpdatedAt.Equal(results[j].Item.UpdatedAt) {
					return results[i].Item.ID < results[j].Item.ID
				}
				return results[i].Item.UpdatedAt.After(results[j].Item.UpdatedAt)
			}
			return results[i].Item.Priority < results[j].Item.Priority
		}
		return leftRank < rightRank
	})

	return results, nil
}

func ListChanges(opts ChangesOptions) ([]domain.Event, error) {
	ajDir, err := ensureAJRepo(opts.RepoPath)
	if err != nil {
		return nil, err
	}

	searchRoot := filepath.Join(ajDir, "issues")
	if strings.TrimSpace(opts.ItemID) != "" {
		searchRoot = filepath.Join(searchRoot, opts.ItemID)
	}

	events := make([]domain.Event, 0)
	err = filepath.WalkDir(searchRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".toml" || filepath.Base(filepath.Dir(path)) != "events" {
			return nil
		}
		event, err := parseEventFile(path)
		if err != nil {
			return err
		}
		if opts.Since != nil && event.At.Before(*opts.Since) {
			return nil
		}
		events = append(events, event)
		return nil
	})
	if err != nil {
		if errors.Is(err, os.ErrNotExist) && strings.TrimSpace(opts.ItemID) != "" {
			return nil, fmt.Errorf("item %s not found", opts.ItemID)
		}
		return nil, fmt.Errorf("read changes: %w", err)
	}

	sort.Slice(events, func(i, j int) bool {
		if events[i].At.Equal(events[j].At) {
			if events[i].ItemID == events[j].ItemID {
				return events[i].ID > events[j].ID
			}
			return events[i].ItemID < events[j].ItemID
		}
		return events[i].At.After(events[j].At)
	})

	if opts.Limit > 0 && len(events) > opts.Limit {
		events = events[:opts.Limit]
	}
	return events, nil
}

func findItemByJiraKey(repoPath, issueKey string) (domain.Item, bool, error) {
	items, err := ListItems(repoPath)
	if err != nil {
		return domain.Item{}, false, err
	}
	for _, item := range items {
		if item.Jira != nil && strings.EqualFold(item.Jira.Key, strings.TrimSpace(issueKey)) {
			return item, true, nil
		}
	}
	return domain.Item{}, false, nil
}

func createImportedJiraItem(repoPath string, issue jira.Issue, statusMap map[string]domain.Status) (domain.Item, error) {
	ajDir, err := ensureAJRepo(repoPath)
	if err != nil {
		return domain.Item{}, err
	}

	var itemID string
	for attempt := 0; attempt < 5; attempt++ {
		itemID, err = idgen.NewItemID()
		if err != nil {
			return domain.Item{}, err
		}
		itemDir := filepath.Join(ajDir, "issues", itemID)
		_, statErr := os.Stat(itemDir)
		if errors.Is(statErr, os.ErrNotExist) {
			break
		}
		if statErr != nil {
			return domain.Item{}, fmt.Errorf("check item directory %s: %w", itemDir, statErr)
		}
		itemID = ""
	}
	if itemID == "" {
		return domain.Item{}, errors.New("failed to allocate a unique item id")
	}

	now := time.Now().UTC().Truncate(time.Second)
	status := mapJiraStatus(issue.Status, statusMap)
	nextAction := importedNextAction(status, issue.Key)
	lastSyncedAt := now
	item := domain.Item{
		ID:         itemID,
		Kind:       mapJiraIssueType(issue.IssueType),
		Title:      defaultJiraTitle(issue),
		Status:     status,
		Priority:   mapJiraPriority(issue.Priority),
		CreatedAt:  now,
		UpdatedAt:  now,
		Goal:       defaultJiraGoal(issue),
		Summary:    fmt.Sprintf("imported from Jira %s", issue.Key),
		NextAction: nextAction,
		Jira: &domain.JiraLink{
			Key:               issue.Key,
			URL:               issue.URL,
			SyncMode:          "import_only",
			SyncState:         "clean",
			LastSyncedAt:      &lastSyncedAt,
			LastRemoteVersion: issue.Updated,
		},
	}

	itemDir := filepath.Join(ajDir, "issues", item.ID)
	eventsDir := filepath.Join(itemDir, "events")
	if err := os.MkdirAll(eventsDir, 0o755); err != nil {
		return domain.Item{}, fmt.Errorf("create item directories: %w", err)
	}
	metaPath := filepath.Join(itemDir, "meta.toml")
	if err := os.WriteFile(metaPath, []byte(marshalItem(item)), 0o644); err != nil {
		return domain.Item{}, fmt.Errorf("write item metadata: %w", err)
	}
	if err := writeCreatedEvent(eventsDir, item); err != nil {
		return domain.Item{}, err
	}
	if err := appendEvent(eventsDir, item.ID, "linked_external", now, "system", item.Summary); err != nil {
		return domain.Item{}, err
	}
	return item, nil
}

func mapJiraIssueType(issueType string) domain.ItemKind {
	switch strings.ToLower(strings.TrimSpace(issueType)) {
	case "bug":
		return domain.KindBug
	case "feature", "story":
		return domain.KindFeature
	case "epic":
		return domain.KindEpic
	case "spike":
		return domain.KindSpike
	default:
		return domain.KindTask
	}
}

func jiraIssueTypeForItem(kind domain.ItemKind) string {
	switch kind {
	case domain.KindBug:
		return "Bug"
	case domain.KindEpic:
		return "Epic"
	case domain.KindFeature:
		return "Story"
	case domain.KindSpike:
		return "Task"
	default:
		return "Task"
	}
}

func mapJiraPriority(priority string) int {
	switch strings.ToLower(strings.TrimSpace(priority)) {
	case "highest":
		return 0
	case "high":
		return 1
	case "medium":
		return 2
	case "low":
		return 3
	case "lowest":
		return 4
	default:
		return 2
	}
}

func mapJiraStatus(status string, statusMap map[string]domain.Status) domain.Status {
	if mapped, ok := statusMap[strings.TrimSpace(status)]; ok {
		return mapped
	}
	return domain.StatusTodo
}

func importedNextAction(status domain.Status, issueKey string) string {
	switch status {
	case domain.StatusBlocked:
		return fmt.Sprintf("Wait for Jira issue %s blockers to clear or update the local next action", issueKey)
	case domain.StatusInReview:
		return "Review the imported Jira issue state and decide on the next follow-up"
	case domain.StatusDone, domain.StatusCanceled:
		return ""
	default:
		return "Review the imported Jira issue details and continue work"
	}
}

func defaultJiraTitle(issue jira.Issue) string {
	if strings.TrimSpace(issue.Summary) != "" {
		return strings.TrimSpace(issue.Summary)
	}
	return fmt.Sprintf("Imported Jira issue %s", issue.Key)
}

func defaultJiraGoal(issue jira.Issue) string {
	if strings.TrimSpace(issue.Description) != "" {
		return strings.TrimSpace(issue.Description)
	}
	if strings.TrimSpace(issue.Summary) != "" {
		return fmt.Sprintf("Imported from Jira %s: %s", issue.Key, strings.TrimSpace(issue.Summary))
	}
	return fmt.Sprintf("Imported from Jira %s", issue.Key)
}

func jiraDescriptionForItem(item domain.Item) string {
	parts := []string{
		"Goal:\n" + item.Goal,
	}
	if strings.TrimSpace(item.Summary) != "" {
		parts = append(parts, "Current Summary:\n"+strings.TrimSpace(item.Summary))
	}
	if strings.TrimSpace(item.NextAction) != "" {
		parts = append(parts, "Next Action:\n"+strings.TrimSpace(item.NextAction))
	}
	return strings.Join(parts, "\n\n")
}

func jiraLocalDirty(item domain.Item) bool {
	if item.Jira == nil {
		return false
	}
	if item.Jira.SyncState == "dirty_local" {
		return true
	}
	if item.Jira.LastSyncedAt == nil {
		return true
	}
	return item.UpdatedAt.After(*item.Jira.LastSyncedAt)
}

func jiraRemoteDirty(item domain.Item, remoteIssue jira.Issue) bool {
	if item.Jira == nil {
		return false
	}
	if item.Jira.SyncState == "dirty_remote" {
		return true
	}
	return strings.TrimSpace(item.Jira.LastRemoteVersion) != "" && strings.TrimSpace(item.Jira.LastRemoteVersion) != strings.TrimSpace(remoteIssue.Updated)
}

func jiraSyncStateForDirection(direction string) string {
	switch direction {
	case "push":
		return "dirty_local"
	case "pull":
		return "dirty_remote"
	default:
		return "clean"
	}
}

func markJiraDirtyLocal(item *domain.Item) {
	if item == nil || item.Jira == nil {
		return
	}
	item.Jira.SyncState = "dirty_local"
}

func loadItemForMutation(repoPath, itemID string) (domain.Item, string, error) {
	ajDir, err := ensureAJRepo(repoPath)
	if err != nil {
		return domain.Item{}, "", err
	}

	itemDir := filepath.Join(ajDir, "issues", itemID)
	item, err := GetItem(repoPath, itemID)
	if err != nil {
		return domain.Item{}, "", err
	}
	return item, itemDir, nil
}

func ensureAJRepo(repoPath string) (string, error) {
	if strings.TrimSpace(repoPath) == "" {
		return "", errors.New("repo path is required")
	}

	ajDir := filepath.Join(filepath.Clean(repoPath), ".aj")
	configPath := filepath.Join(ajDir, "config.toml")
	if _, err := os.Stat(configPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("aj is not initialized in %s (run `aj init` first)", filepath.Clean(repoPath))
		}
		return "", fmt.Errorf("check aj config: %w", err)
	}
	return ajDir, nil
}

func filterItems(items []domain.Item, keep func(domain.Item) bool) []domain.Item {
	filtered := make([]domain.Item, 0, len(items))
	for _, item := range items {
		if keep(item) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func indexItems(items []domain.Item) map[string]domain.Item {
	index := make(map[string]domain.Item, len(items))
	for _, item := range items {
		index[item.ID] = item
	}
	return index
}

func unmetDependencies(item domain.Item, itemMap map[string]domain.Item) []string {
	if len(item.DependsOn) == 0 {
		return nil
	}
	unmet := make([]string, 0, len(item.DependsOn))
	for _, depID := range item.DependsOn {
		dep, ok := itemMap[depID]
		if !ok || dep.Status != domain.StatusDone {
			unmet = append(unmet, depID)
		}
	}
	sort.Strings(unmet)
	return unmet
}

func marshalItem(item domain.Item) string {
	var lines []string
	lines = append(lines, fmt.Sprintf("id = %s", strconv.Quote(item.ID)))
	lines = append(lines, fmt.Sprintf("kind = %s", strconv.Quote(string(item.Kind))))
	lines = append(lines, fmt.Sprintf("title = %s", strconv.Quote(item.Title)))
	lines = append(lines, fmt.Sprintf("status = %s", strconv.Quote(string(item.Status))))
	lines = append(lines, fmt.Sprintf("priority = %d", item.Priority))
	lines = append(lines, fmt.Sprintf("created_at = %s", strconv.Quote(item.CreatedAt.Format(time.RFC3339))))
	lines = append(lines, fmt.Sprintf("updated_at = %s", strconv.Quote(item.UpdatedAt.Format(time.RFC3339))))
	lines = append(lines, fmt.Sprintf("goal = %s", strconv.Quote(item.Goal)))
	lines = append(lines, fmt.Sprintf("summary = %s", strconv.Quote(item.Summary)))
	lines = append(lines, fmt.Sprintf("next_action = %s", strconv.Quote(item.NextAction)))
	lines = append(lines, fmt.Sprintf("depends_on = %s", marshalStringList(item.DependsOn)))
	if item.Lease != nil {
		lines = append(lines, fmt.Sprintf("lease_owner = %s", strconv.Quote(item.Lease.Owner)))
		lines = append(lines, fmt.Sprintf("lease_claimed_at = %s", strconv.Quote(item.Lease.ClaimedAt.Format(time.RFC3339))))
		lines = append(lines, fmt.Sprintf("lease_expires_at = %s", strconv.Quote(item.Lease.ExpiresAt.Format(time.RFC3339))))
	}
	if item.Jira != nil {
		lines = append(lines, fmt.Sprintf("jira_key = %s", strconv.Quote(item.Jira.Key)))
		lines = append(lines, fmt.Sprintf("jira_url = %s", strconv.Quote(item.Jira.URL)))
		lines = append(lines, fmt.Sprintf("jira_sync_mode = %s", strconv.Quote(item.Jira.SyncMode)))
		lines = append(lines, fmt.Sprintf("jira_sync_state = %s", strconv.Quote(item.Jira.SyncState)))
		lines = append(lines, fmt.Sprintf("jira_last_remote_version = %s", strconv.Quote(item.Jira.LastRemoteVersion)))
		if item.Jira.LastSyncedAt != nil {
			lines = append(lines, fmt.Sprintf("jira_last_synced_at = %s", strconv.Quote(item.Jira.LastSyncedAt.Format(time.RFC3339))))
		}
	}
	lines = append(lines, "")
	return strings.Join(lines, "\n")
}

func parseItem(raw string) (domain.Item, error) {
	values := make(map[string]string)
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return domain.Item{}, fmt.Errorf("invalid metadata line %q", line)
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		values[key] = value
	}

	requiredString := func(key string) (string, error) {
		value, ok := values[key]
		if !ok {
			return "", fmt.Errorf("missing %s", key)
		}
		unquoted, err := strconv.Unquote(value)
		if err != nil {
			return "", fmt.Errorf("invalid quoted value for %s: %w", key, err)
		}
		return unquoted, nil
	}

	parseTime := func(key string) (time.Time, error) {
		rawValue, err := requiredString(key)
		if err != nil {
			return time.Time{}, err
		}
		parsed, err := time.Parse(time.RFC3339, rawValue)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid time for %s: %w", key, err)
		}
		return parsed, nil
	}

	id, err := requiredString("id")
	if err != nil {
		return domain.Item{}, err
	}
	kindRaw, err := requiredString("kind")
	if err != nil {
		return domain.Item{}, err
	}
	kind, err := domain.ParseItemKind(kindRaw)
	if err != nil {
		return domain.Item{}, err
	}
	title, err := requiredString("title")
	if err != nil {
		return domain.Item{}, err
	}
	statusRaw, err := requiredString("status")
	if err != nil {
		return domain.Item{}, err
	}
	priorityRaw, ok := values["priority"]
	if !ok {
		return domain.Item{}, errors.New("missing priority")
	}
	priority, err := strconv.Atoi(priorityRaw)
	if err != nil {
		return domain.Item{}, fmt.Errorf("invalid priority: %w", err)
	}
	createdAt, err := parseTime("created_at")
	if err != nil {
		return domain.Item{}, err
	}
	updatedAt, err := parseTime("updated_at")
	if err != nil {
		return domain.Item{}, err
	}
	goal, err := requiredString("goal")
	if err != nil {
		return domain.Item{}, err
	}
	summary, err := requiredString("summary")
	if err != nil {
		return domain.Item{}, err
	}
	nextAction, err := requiredString("next_action")
	if err != nil {
		return domain.Item{}, err
	}
	dependsOn, err := parseStringList(values["depends_on"])
	if err != nil {
		return domain.Item{}, fmt.Errorf("parse depends_on: %w", err)
	}

	var lease *domain.Lease
	leaseOwnerRaw, hasLeaseOwner := values["lease_owner"]
	if hasLeaseOwner {
		leaseOwner, err := strconv.Unquote(leaseOwnerRaw)
		if err != nil {
			return domain.Item{}, fmt.Errorf("invalid quoted value for lease_owner: %w", err)
		}
		claimedAt, err := parseTime("lease_claimed_at")
		if err != nil {
			return domain.Item{}, err
		}
		expiresAt, err := parseTime("lease_expires_at")
		if err != nil {
			return domain.Item{}, err
		}
		lease = &domain.Lease{
			Owner:     leaseOwner,
			ClaimedAt: claimedAt,
			ExpiresAt: expiresAt,
		}
	}

	var jiraLink *domain.JiraLink
	jiraKeyRaw, hasJiraKey := values["jira_key"]
	if hasJiraKey {
		jiraKey, err := strconv.Unquote(jiraKeyRaw)
		if err != nil {
			return domain.Item{}, fmt.Errorf("invalid quoted value for jira_key: %w", err)
		}
		jiraURL, err := requiredString("jira_url")
		if err != nil {
			return domain.Item{}, err
		}
		jiraSyncMode, err := requiredString("jira_sync_mode")
		if err != nil {
			return domain.Item{}, err
		}
		jiraSyncState, err := requiredString("jira_sync_state")
		if err != nil {
			return domain.Item{}, err
		}
		lastRemoteVersion := ""
		if rawVersion, ok := values["jira_last_remote_version"]; ok {
			lastRemoteVersion, err = strconv.Unquote(rawVersion)
			if err != nil {
				return domain.Item{}, fmt.Errorf("invalid quoted value for jira_last_remote_version: %w", err)
			}
		}
		var lastSyncedAt *time.Time
		if _, ok := values["jira_last_synced_at"]; ok {
			parsed, err := parseTime("jira_last_synced_at")
			if err != nil {
				return domain.Item{}, err
			}
			lastSyncedAt = &parsed
		}
		jiraLink = &domain.JiraLink{
			Key:               jiraKey,
			URL:               jiraURL,
			SyncMode:          jiraSyncMode,
			SyncState:         jiraSyncState,
			LastSyncedAt:      lastSyncedAt,
			LastRemoteVersion: lastRemoteVersion,
		}
	}

	return domain.Item{
		ID:         id,
		Kind:       kind,
		Title:      title,
		Status:     domain.Status(statusRaw),
		Priority:   priority,
		CreatedAt:  createdAt,
		UpdatedAt:  updatedAt,
		Goal:       goal,
		Summary:    summary,
		NextAction: nextAction,
		DependsOn:  dependsOn,
		Lease:      lease,
		Jira:       jiraLink,
	}, nil
}

func writeCreatedEvent(eventsDir string, item domain.Item) error {
	return appendEvent(eventsDir, item.ID, "created", item.CreatedAt, "system", "Item created.")
}

func persistItemMutation(itemDir string, item domain.Item, eventType, actor, summary string) error {
	return persistItemMutationWithEventSummary(itemDir, item, eventType, actor, summary)
}

func persistItemMutationWithEventSummary(itemDir string, item domain.Item, eventType, actor, eventSummary string) error {
	eventsDir := filepath.Join(itemDir, "events")
	if err := appendEvent(eventsDir, item.ID, eventType, item.UpdatedAt, actor, eventSummary); err != nil {
		return err
	}
	metaPath := filepath.Join(itemDir, "meta.toml")
	if err := os.WriteFile(metaPath, []byte(marshalItem(item)), 0o644); err != nil {
		return fmt.Errorf("write item metadata: %w", err)
	}
	return nil
}

func appendEvent(eventsDir, itemID, eventType string, at time.Time, actor, summary string) error {
	eventID, err := idgen.NewEventID()
	if err != nil {
		return err
	}
	fileName := fmt.Sprintf("%s_%s.toml", at.Format("2006-01-02T15-04-05Z"), eventID)
	eventPath := filepath.Join(eventsDir, fileName)
	content := strings.Join([]string{
		fmt.Sprintf("id = %s", strconv.Quote(eventID)),
		fmt.Sprintf("item_id = %s", strconv.Quote(itemID)),
		fmt.Sprintf("type = %s", strconv.Quote(eventType)),
		fmt.Sprintf("at = %s", strconv.Quote(at.Format(time.RFC3339))),
		fmt.Sprintf("actor = %s", strconv.Quote(actor)),
		fmt.Sprintf("summary = %s", strconv.Quote(summary)),
		"",
	}, "\n")
	if err := os.WriteFile(eventPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write %s event: %w", eventType, err)
	}
	return nil
}

func marshalStringList(values []string) string {
	if len(values) == 0 {
		return "[]"
	}
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, strconv.Quote(value))
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

func parseStringList(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return nil, nil
	}
	if !strings.HasPrefix(raw, "[") || !strings.HasSuffix(raw, "]") {
		return nil, fmt.Errorf("expected bracketed list, got %q", raw)
	}
	inner := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(raw, "["), "]"))
	if inner == "" {
		return nil, nil
	}
	parts := strings.Split(inner, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		value, err := strconv.Unquote(strings.TrimSpace(part))
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	sort.Strings(result)
	return result, nil
}

func parseEventFile(path string) (domain.Event, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return domain.Event{}, fmt.Errorf("read event file %s: %w", path, err)
	}

	values := make(map[string]string)
	for _, line := range strings.Split(string(bytes), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return domain.Event{}, fmt.Errorf("invalid event line %q", line)
		}
		values[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}

	requiredString := func(key string) (string, error) {
		value, ok := values[key]
		if !ok {
			return "", fmt.Errorf("missing %s", key)
		}
		unquoted, err := strconv.Unquote(value)
		if err != nil {
			return "", fmt.Errorf("invalid quoted value for %s: %w", key, err)
		}
		return unquoted, nil
	}

	id, err := requiredString("id")
	if err != nil {
		return domain.Event{}, err
	}
	itemID, err := requiredString("item_id")
	if err != nil {
		return domain.Event{}, err
	}
	eventType, err := requiredString("type")
	if err != nil {
		return domain.Event{}, err
	}
	atRaw, err := requiredString("at")
	if err != nil {
		return domain.Event{}, err
	}
	at, err := time.Parse(time.RFC3339, atRaw)
	if err != nil {
		return domain.Event{}, fmt.Errorf("invalid event time: %w", err)
	}
	actor, err := requiredString("actor")
	if err != nil {
		return domain.Event{}, err
	}
	summary, err := requiredString("summary")
	if err != nil {
		return domain.Event{}, err
	}

	return domain.Event{
		ID:      id,
		ItemID:  itemID,
		Type:    eventType,
		At:      at,
		Actor:   actor,
		Summary: summary,
	}, nil
}
