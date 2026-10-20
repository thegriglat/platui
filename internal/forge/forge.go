// Package forge creates and finds MRs in GitLab via the glab CLI.
package forge

import (
	"context"
	"errors"
	"time"
)

// MRRequest holds the parameters of an MR to create.
type MRRequest struct {
	Dir         string // repository directory (the command's cwd)
	Head        string // source branch
	Target      string // empty — the project default
	Title, Body string
	Draft       bool
}

// MR is a found merge request.
type MR struct {
	IID          int
	Title, URL   string // URL — web_url
	State        string // opened|merged|closed (as glab reports it)
	SourceBranch string
	TargetBranch string
	UpdatedAt    time.Time
}

// Tool provides MR operations.
type Tool interface {
	CreateMR(ctx context.Context, r MRRequest) (url string, err error)
	FindMRs(ctx context.Context, dir string, branches []string, key string) ([]MR, error)
}

// ErrNotFound means glab is not installed.
var ErrNotFound = errors.New("glab not installed")
