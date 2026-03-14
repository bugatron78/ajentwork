package app

import (
	"ajentwork/internal/store"
)

type JiraSpaceExistsInput struct {
	RepoPath string
	Key      string
}

type JiraSpaceExistsService struct{}

func (s JiraSpaceExistsService) Run(input JiraSpaceExistsInput) (store.JiraSpaceExistsResult, error) {
	return store.JiraSpaceExists(store.JiraSpaceExistsOptions{
		RepoPath: input.RepoPath,
		Key:      input.Key,
	})
}

type JiraSpaceCreateInput struct {
	RepoPath string
	Key      string
	Name     string
	Type     string
	Template string
}

type JiraSpaceCreateService struct{}

func (s JiraSpaceCreateService) Run(input JiraSpaceCreateInput) (store.JiraSpaceCreateResult, error) {
	return store.CreateJiraSpace(store.JiraSpaceCreateOptions{
		RepoPath: input.RepoPath,
		Key:      input.Key,
		Name:     input.Name,
		Type:     input.Type,
		Template: input.Template,
	})
}

type JiraSpaceEnsureInput struct {
	RepoPath string
	Key      string
	Name     string
	Type     string
	Template string
}

type JiraSpaceEnsureService struct{}

func (s JiraSpaceEnsureService) Run(input JiraSpaceEnsureInput) (store.JiraSpaceCreateResult, error) {
	return store.EnsureJiraSpace(store.JiraSpaceEnsureOptions{
		RepoPath: input.RepoPath,
		Key:      input.Key,
		Name:     input.Name,
		Type:     input.Type,
		Template: input.Template,
	})
}

type JiraSpaceListInput struct {
	RepoPath string
	Query    string
	Limit    int
}

type JiraSpaceListService struct{}

func (s JiraSpaceListService) Run(input JiraSpaceListInput) (store.JiraSpaceListResult, error) {
	return store.ListJiraSpaces(store.JiraSpaceListOptions{
		RepoPath: input.RepoPath,
		Query:    input.Query,
		Limit:    input.Limit,
	})
}
