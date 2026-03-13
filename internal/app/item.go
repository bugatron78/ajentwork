package app

import (
	"time"

	"ajentwork/internal/domain"
	"ajentwork/internal/store"
)

type NewItemInput struct {
	RepoPath   string
	Kind       domain.ItemKind
	Title      string
	Goal       string
	NextAction string
	Priority   int
}

type NewItemService struct{}

func (s NewItemService) Run(input NewItemInput) (domain.Item, error) {
	return store.CreateItem(store.CreateItemOptions{
		RepoPath:   input.RepoPath,
		Kind:       input.Kind,
		Title:      input.Title,
		Goal:       input.Goal,
		NextAction: input.NextAction,
		Priority:   input.Priority,
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
