// Package plane — клиент REST API Plane v1 (контракт C3 v6).
package plane

import (
	"context"
	"strings"
	"time"
)

// Project — тип контракта C3.
type Project struct {
	ID, Identifier, Name string
}

// State — тип контракта C3.
type State struct {
	ID, Name, Group, Color string
	Sequence               float64 // `sequence`: число или строка с числом; иначе 0
	Default                bool    // `default`, иначе `is_default`
}

// SearchHit — элемент серверного поиска.
type SearchHit struct {
	ID, Name          string
	Sequence          int
	ProjectID         string
	ProjectIdentifier string
}

type Member struct {
	ID, DisplayName, Email, FirstName, LastName string
}

// Name — DisplayName, иначе "First Last", иначе Email.
func (m Member) Name() string {
	if m.DisplayName != "" {
		return m.DisplayName
	}
	if n := strings.TrimSpace(m.FirstName + " " + m.LastName); n != "" {
		return n
	}
	return m.Email
}

// Label — тип контракта C3.
type Label struct {
	ID, Name, Color string
}

// Comment — тип контракта C3.
type Comment struct {
	ID        string
	Author    string
	HTML      string
	CreatedAt time.Time
}

// WorkItem — тип контракта C3.
type WorkItem struct {
	ID                   string
	ProjectID            string
	Sequence             int
	Name                 string
	DescHTML             string
	StateID              string
	Priority             string
	AssigneeIDs          []string
	LabelIDs             []string
	ParentID             string // родитель (подзадача), иначе ""
	CreatedAt, UpdatedAt time.Time
}

// WorkItemPatch: nil — поле не отправляется.
type WorkItemPatch struct {
	StateID, Priority, Name, DescHTML *string
	AssigneeIDs, LabelIDs             *[]string
}

// NewWorkItem — тип контракта C3.
type NewWorkItem struct {
	Name, DescHTML        string
	StateID, Priority     string   // пусто — не отправляется
	AssigneeIDs, LabelIDs []string // nil/пусто — не отправляется
	ParentID              string   // пусто — не отправляется
}

// API — тип контракта C3.
type API interface {
	Me(ctx context.Context) (Member, error)
	Projects(ctx context.Context) ([]Project, error)
	States(ctx context.Context, projectID string) ([]State, error)
	Members(ctx context.Context, projectID string) ([]Member, error)
	Labels(ctx context.Context, projectID string) ([]Label, error)
	WorkItems(ctx context.Context, projectID string) ([]WorkItem, error)
	WorkItem(ctx context.Context, projectID, id string) (WorkItem, error)
	CreateWorkItem(ctx context.Context, projectID string, n NewWorkItem) (WorkItem, error)
	UpdateWorkItem(ctx context.Context, projectID, id string, p WorkItemPatch) (WorkItem, error)
	Comments(ctx context.Context, projectID, id string) ([]Comment, error)
	CreateComment(ctx context.Context, projectID, id, html string) (Comment, error)
	WorkItemByKey(ctx context.Context, identifier string, seq int) (WorkItem, error)
	Search(ctx context.Context, query, projectID string, limit int) ([]SearchHit, error)
	WebURL(p Project, it WorkItem) string
}
