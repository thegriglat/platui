// Package plane is the Plane REST API v1 client (contract C3 v6).
package plane

import (
	"context"
	"strings"
	"time"
)

// Project is a contract C3 type.
type Project struct {
	ID, Identifier, Name string
}

// State is a contract C3 type.
type State struct {
	ID, Name, Group, Color string
	Sequence               float64 // `sequence`: a number or a numeric string; otherwise 0
	Default                bool    // `default`, otherwise `is_default`
}

// SearchHit is a server search element.
type SearchHit struct {
	ID, Name          string
	Sequence          int
	ProjectID         string
	ProjectIdentifier string
}

type Member struct {
	ID, DisplayName, Email, FirstName, LastName string
}

// Name is DisplayName, otherwise "First Last", otherwise Email.
func (m Member) Name() string {
	if m.DisplayName != "" {
		return m.DisplayName
	}
	if n := strings.TrimSpace(m.FirstName + " " + m.LastName); n != "" {
		return n
	}
	return m.Email
}

// Label is a contract C3 type.
type Label struct {
	ID, Name, Color string
}

// Comment is a contract C3 type.
type Comment struct {
	ID        string
	Author    string
	HTML      string
	CreatedAt time.Time
}

// WorkItem is a contract C3 type.
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
	ParentID             string // parent (subtask), otherwise ""
	CreatedAt, UpdatedAt time.Time
}

// WorkItemPatch: nil means the field is not sent.
type WorkItemPatch struct {
	StateID, Priority, Name, DescHTML *string
	AssigneeIDs, LabelIDs             *[]string
}

// NewWorkItem is a contract C3 type.
type NewWorkItem struct {
	Name, DescHTML        string
	StateID, Priority     string   // empty: not sent
	AssigneeIDs, LabelIDs []string // nil/empty: not sent
	ParentID              string   // empty: not sent
}

// API is a contract C3 type.
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
