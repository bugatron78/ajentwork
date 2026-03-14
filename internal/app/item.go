package app

import (
	"time"

	"ajentwork/internal/domain"
	"ajentwork/internal/store"
)

type NewItemInput struct {
	RepoPath      string
	Kind          domain.ItemKind
	Title         string
	Goal          string
	NextAction    string
	Acceptance    []string
	Constraints   []string
	Risks         []string
	RelevantFiles []string
	Verification  []string
	Priority      int
}

type NewItemService struct{}

func (s NewItemService) Run(input NewItemInput) (domain.Item, error) {
	return store.CreateItem(store.CreateItemOptions{
		RepoPath:      input.RepoPath,
		Kind:          input.Kind,
		Title:         input.Title,
		Goal:          input.Goal,
		NextAction:    input.NextAction,
		Acceptance:    input.Acceptance,
		Constraints:   input.Constraints,
		Risks:         input.Risks,
		RelevantFiles: input.RelevantFiles,
		Verification:  input.Verification,
		Priority:      input.Priority,
	})
}

type ListItemsService struct{}

func (s ListItemsService) Run(repoPath string) ([]domain.Item, error) {
	return store.ListItems(repoPath)
}

type ShowItemService struct{}

func (s ShowItemService) Run(repoPath, itemID string) (domain.Item, error) {
	return store.GetItem(repoPath, itemID)
}

type UpdateItemInput struct {
	RepoPath   string
	ItemID     string
	Summary    string
	NextAction *string
	Status     *domain.Status
}

type UpdateItemService struct{}

func (s UpdateItemService) Run(input UpdateItemInput) (domain.Item, error) {
	return store.UpdateItem(store.UpdateItemOptions{
		RepoPath:   input.RepoPath,
		ItemID:     input.ItemID,
		Summary:    input.Summary,
		NextAction: input.NextAction,
		Status:     input.Status,
	})
}

type CompleteItemInput struct {
	RepoPath string
	ItemID   string
	Summary  string
}

type CompleteItemService struct{}

func (s CompleteItemService) Run(input CompleteItemInput) (domain.Item, error) {
	return store.CompleteItem(store.CompleteItemOptions{
		RepoPath: input.RepoPath,
		ItemID:   input.ItemID,
		Summary:  input.Summary,
	})
}

type BlockItemInput struct {
	RepoPath   string
	ItemID     string
	Summary    string
	OnID       string
	NextAction *string
}

type BlockItemService struct{}

func (s BlockItemService) Run(input BlockItemInput) (domain.Item, error) {
	return store.BlockItem(store.BlockItemOptions{
		RepoPath:   input.RepoPath,
		ItemID:     input.ItemID,
		Summary:    input.Summary,
		OnID:       input.OnID,
		NextAction: input.NextAction,
	})
}

type UnblockItemInput struct {
	RepoPath   string
	ItemID     string
	Summary    string
	NextAction *string
	Status     *domain.Status
}

type UnblockItemService struct{}

func (s UnblockItemService) Run(input UnblockItemInput) (domain.Item, error) {
	return store.UnblockItem(store.UnblockItemOptions{
		RepoPath:   input.RepoPath,
		ItemID:     input.ItemID,
		Summary:    input.Summary,
		NextAction: input.NextAction,
		Status:     input.Status,
	})
}

type TakeItemInput struct {
	RepoPath string
	ItemID   string
	Agent    string
	TTL      time.Duration
	Force    bool
}

type TakeItemService struct{}

func (s TakeItemService) Run(input TakeItemInput) (domain.Item, error) {
	return store.TakeItem(store.TakeItemOptions{
		RepoPath: input.RepoPath,
		ItemID:   input.ItemID,
		Agent:    input.Agent,
		TTL:      input.TTL,
		Force:    input.Force,
	})
}

type ReleaseItemInput struct {
	RepoPath string
	ItemID   string
}

type ReleaseItemService struct{}

func (s ReleaseItemService) Run(input ReleaseItemInput) (domain.Item, error) {
	return store.ReleaseItem(store.ReleaseItemOptions{
		RepoPath: input.RepoPath,
		ItemID:   input.ItemID,
	})
}

type HandoffItemInput struct {
	RepoPath   string
	ItemID     string
	ToAgent    string
	Summary    string
	NextAction *string
	TTL        time.Duration
}

type HandoffItemService struct{}

func (s HandoffItemService) Run(input HandoffItemInput) (domain.Item, error) {
	return store.HandoffItem(store.HandoffItemOptions{
		RepoPath:   input.RepoPath,
		ItemID:     input.ItemID,
		ToAgent:    input.ToAgent,
		Summary:    input.Summary,
		NextAction: input.NextAction,
		TTL:        input.TTL,
	})
}

type ReopenItemInput struct {
	RepoPath   string
	ItemID     string
	Summary    string
	NextAction string
	Status     *domain.Status
}

type ReopenItemService struct{}

func (s ReopenItemService) Run(input ReopenItemInput) (domain.Item, error) {
	return store.ReopenItem(store.ReopenItemOptions{
		RepoPath:   input.RepoPath,
		ItemID:     input.ItemID,
		Summary:    input.Summary,
		NextAction: input.NextAction,
		Status:     input.Status,
	})
}

type CheckpointItemInput struct {
	RepoPath   string
	ItemID     string
	Summary    string
	NextAction *string
	Risks      []string
	Verify     []string
}

type CheckpointItemService struct{}

func (s CheckpointItemService) Run(input CheckpointItemInput) (domain.Item, error) {
	return store.CheckpointItem(store.CheckpointItemOptions{
		RepoPath:   input.RepoPath,
		ItemID:     input.ItemID,
		Summary:    input.Summary,
		NextAction: input.NextAction,
		Risks:      input.Risks,
		Verify:     input.Verify,
	})
}

type NextItemInput struct {
	RepoPath string
	Agent    string
}

type NextItemService struct{}

func (s NextItemService) Run(input NextItemInput) (store.NextItemResult, error) {
	return store.RecommendNext(input.RepoPath, input.Agent)
}

type InboxInput struct {
	RepoPath string
	Agent    string
}

type InboxService struct{}

func (s InboxService) Run(input InboxInput) ([]store.InboxEntry, error) {
	return store.Inbox(input.RepoPath, input.Agent)
}

type LinkDependencyInput struct {
	RepoPath    string
	ItemID      string
	DependsOnID string
}

type LinkDependencyService struct{}

func (s LinkDependencyService) Run(input LinkDependencyInput) (domain.Item, error) {
	return store.LinkDependency(store.LinkDependencyOptions{
		RepoPath:    input.RepoPath,
		ItemID:      input.ItemID,
		DependsOnID: input.DependsOnID,
	})
}

type SetParentInput struct {
	RepoPath string
	ItemID   string
	ParentID string
}

type SetParentService struct{}

func (s SetParentService) Run(input SetParentInput) (domain.Item, error) {
	return store.SetParent(store.SetParentOptions{
		RepoPath: input.RepoPath,
		ItemID:   input.ItemID,
		ParentID: input.ParentID,
	})
}

type UnlinkRelationInput struct {
	RepoPath     string
	ItemID       string
	DependsOnID  string
	RemoveParent bool
}

type UnlinkRelationService struct{}

func (s UnlinkRelationService) Run(input UnlinkRelationInput) (domain.Item, error) {
	return store.UnlinkRelation(store.UnlinkRelationOptions{
		RepoPath:     input.RepoPath,
		ItemID:       input.ItemID,
		DependsOnID:  input.DependsOnID,
		RemoveParent: input.RemoveParent,
	})
}

type ChangesInput struct {
	RepoPath string
	ItemID   string
	Since    *time.Time
	Limit    int
}

type ChangesService struct{}

func (s ChangesService) Run(input ChangesInput) ([]domain.Event, error) {
	return store.ListChanges(store.ChangesOptions{
		RepoPath: input.RepoPath,
		ItemID:   input.ItemID,
		Since:    input.Since,
		Limit:    input.Limit,
	})
}

type AttachArtifactInput struct {
	RepoPath string
	ItemID   string
	Path     string
	Summary  string
	Label    string
}

type AttachArtifactService struct{}

func (s AttachArtifactService) Run(input AttachArtifactInput) (domain.Artifact, error) {
	return store.AttachArtifact(store.AttachArtifactOptions{
		RepoPath: input.RepoPath,
		ItemID:   input.ItemID,
		Path:     input.Path,
		Summary:  input.Summary,
		Label:    input.Label,
	})
}

type RecordReceiptInput struct {
	RepoPath string
	ItemID   string
	Summary  string
	Command  string
	ExitCode int
	Output   string
	Label    string
}

type RecordReceiptService struct{}

func (s RecordReceiptService) Run(input RecordReceiptInput) (domain.Artifact, error) {
	return store.RecordReceipt(store.RecordReceiptOptions{
		RepoPath: input.RepoPath,
		ItemID:   input.ItemID,
		Summary:  input.Summary,
		Command:  input.Command,
		ExitCode: input.ExitCode,
		Output:   input.Output,
		Label:    input.Label,
	})
}

type ListArtifactsInput struct {
	RepoPath string
	ItemID   string
	Limit    int
}

type ListArtifactsService struct{}

func (s ListArtifactsService) Run(input ListArtifactsInput) ([]domain.Artifact, error) {
	return store.ListArtifacts(input.RepoPath, input.ItemID, input.Limit)
}

type ReadyInput struct {
	RepoPath string
	Agent    string
}

type ReadyService struct{}

func (s ReadyService) Run(input ReadyInput) ([]store.ReadyEntry, error) {
	return store.Ready(store.ReadyOptions{
		RepoPath: input.RepoPath,
		Agent:    input.Agent,
	})
}

type SearchItemsInput struct {
	RepoPath string
	Query    string
	Status   *domain.Status
	Kind     *domain.ItemKind
	Limit    int
}

type SearchItemsService struct{}

func (s SearchItemsService) Run(input SearchItemsInput) (store.SearchItemsResult, error) {
	return store.SearchItems(store.SearchItemsOptions{
		RepoPath: input.RepoPath,
		Query:    input.Query,
		Status:   input.Status,
		Kind:     input.Kind,
		Limit:    input.Limit,
	})
}

type ReportInput struct {
	RepoPath string
	Agent    string
	Limit    int
}

type ReportService struct{}

func (s ReportService) Run(input ReportInput) (store.ReportResult, error) {
	return store.BuildReport(store.ReportOptions{
		RepoPath: input.RepoPath,
		Agent:    input.Agent,
		Limit:    input.Limit,
	})
}

type ImportJiraIssueInput struct {
	RepoPath string
	IssueKey string
}

type ImportJiraIssueService struct{}

func (s ImportJiraIssueService) Run(input ImportJiraIssueInput) (store.ImportJiraIssueResult, error) {
	return store.ImportJiraIssue(store.ImportJiraIssueOptions{
		RepoPath: input.RepoPath,
		IssueKey: input.IssueKey,
	})
}

type ExportJiraIssueInput struct {
	RepoPath   string
	ItemID     string
	ProjectKey string
	IssueType  string
}

type ExportJiraIssueService struct{}

func (s ExportJiraIssueService) Run(input ExportJiraIssueInput) (store.ExportJiraIssueResult, error) {
	return store.ExportJiraIssue(store.ExportJiraIssueOptions{
		RepoPath:   input.RepoPath,
		ItemID:     input.ItemID,
		ProjectKey: input.ProjectKey,
		IssueType:  input.IssueType,
	})
}

type LinkJiraIssueInput struct {
	RepoPath string
	ItemID   string
	IssueKey string
	Replace  bool
}

type LinkJiraIssueService struct{}

func (s LinkJiraIssueService) Run(input LinkJiraIssueInput) (store.LinkJiraIssueResult, error) {
	return store.LinkJiraIssue(store.LinkJiraIssueOptions{
		RepoPath: input.RepoPath,
		ItemID:   input.ItemID,
		IssueKey: input.IssueKey,
		Replace:  input.Replace,
	})
}

type SearchJiraIssuesInput struct {
	RepoPath   string
	Query      string
	JQL        string
	ProjectKey string
	Limit      int
}

type SearchJiraIssuesService struct{}

func (s SearchJiraIssuesService) Run(input SearchJiraIssuesInput) (store.JiraSearchResult, error) {
	return store.SearchJiraIssues(store.SearchJiraIssuesOptions{
		RepoPath:   input.RepoPath,
		Query:      input.Query,
		JQL:        input.JQL,
		ProjectKey: input.ProjectKey,
		Limit:      input.Limit,
	})
}

type UnlinkJiraIssueInput struct {
	RepoPath string
	ItemID   string
	Force    bool
}

type UnlinkJiraIssueService struct{}

func (s UnlinkJiraIssueService) Run(input UnlinkJiraIssueInput) (domain.Item, error) {
	return store.UnlinkJiraIssue(store.UnlinkJiraIssueOptions{
		RepoPath: input.RepoPath,
		ItemID:   input.ItemID,
		Force:    input.Force,
	})
}

type SyncJiraIssueInput struct {
	RepoPath string
	ItemID   string
	DryRun   bool
	Resolve  string
}

type SyncJiraIssueService struct{}

func (s SyncJiraIssueService) Run(input SyncJiraIssueInput) (store.SyncJiraIssueResult, error) {
	return store.SyncJiraIssue(store.SyncJiraIssueOptions{
		RepoPath: input.RepoPath,
		ItemID:   input.ItemID,
		DryRun:   input.DryRun,
		Resolve:  input.Resolve,
	})
}

type JiraStatusMapService struct{}

func (s JiraStatusMapService) Run(repoPath string) (store.JiraStatusMapResult, error) {
	return store.ShowJiraStatusMap(repoPath)
}

type JiraTransitionsInput struct {
	RepoPath string
	ItemID   string
}

type JiraTransitionsService struct{}

func (s JiraTransitionsService) Run(input JiraTransitionsInput) (store.JiraTransitionsResult, error) {
	return store.ShowJiraTransitions(input.RepoPath, input.ItemID)
}

type CommentJiraIssueInput struct {
	RepoPath string
	ItemID   string
	Summary  string
}

type CommentJiraIssueService struct{}

func (s CommentJiraIssueService) Run(input CommentJiraIssueInput) (domain.Item, error) {
	return store.CommentJiraIssue(store.CommentJiraIssueOptions{
		RepoPath: input.RepoPath,
		ItemID:   input.ItemID,
		Summary:  input.Summary,
	})
}
