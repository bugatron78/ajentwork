package app

import "ajentwork/internal/store"

type InitService struct{}

func (s InitService) Run(repoPath string, force bool) (store.InitResult, error) {
	return store.InitRepo(store.InitOptions{
		RepoPath: repoPath,
		Force:    force,
	})
}
