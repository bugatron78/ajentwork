package app

import "ajentwork/internal/store"

type InitService struct{}

type InitInput struct {
	RepoPath          string
	Force             bool
	JiraEnabled       bool
	JiraBaseURL       string
	JiraProject       string
	EnsureJiraSpace   bool
	JiraSpaceName     string
	JiraSpaceType     string
	JiraSpaceTemplate string
}

func (s InitService) Run(input InitInput) (store.InitResult, error) {
	return store.InitRepo(store.InitOptions{
		RepoPath:          input.RepoPath,
		Force:             input.Force,
		JiraEnabled:       input.JiraEnabled,
		JiraBaseURL:       input.JiraBaseURL,
		JiraProject:       input.JiraProject,
		EnsureJiraSpace:   input.EnsureJiraSpace,
		JiraSpaceName:     input.JiraSpaceName,
		JiraSpaceType:     input.JiraSpaceType,
		JiraSpaceTemplate: input.JiraSpaceTemplate,
	})
}
