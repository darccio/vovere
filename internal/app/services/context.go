package services

import (
	"context"
)

type contextKey string

const (
	repositoryKey contextKey = "repository"
)

// WithRepository stores a repository service in the context
func WithRepository(ctx context.Context, repo *Repository) context.Context {
	return context.WithValue(ctx, repositoryKey, repo)
}

// RepositoryFromContext retrieves the repository service from the context
func RepositoryFromContext(ctx context.Context) *Repository {
	repo, _ := ctx.Value(repositoryKey).(*Repository)
	return repo
}
