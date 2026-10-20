// Package forge создаёт и ищет MR в GitLab через CLI glab.
package forge

import (
	"context"
	"errors"
	"time"
)

// MRRequest — параметры создаваемого MR.
type MRRequest struct {
	Dir         string // каталог репозитория (cwd команды)
	Head        string // исходная ветка
	Target      string // пусто — по умолчанию проекта
	Title, Body string
	Draft       bool
}

// MR — найденный merge request.
type MR struct {
	IID          int
	Title, URL   string // URL — web_url
	State        string // opened|merged|closed (как отдаёт glab)
	SourceBranch string
	TargetBranch string
	UpdatedAt    time.Time
}

// Tool — операции с MR.
type Tool interface {
	CreateMR(ctx context.Context, r MRRequest) (url string, err error)
	FindMRs(ctx context.Context, dir string, branches []string, key string) ([]MR, error)
}

// ErrNotFound — glab не установлен.
var ErrNotFound = errors.New("glab not installed")
