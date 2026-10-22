// Package app is the platui service layer: scenarios on top of plane, gitbranch,
// forge and cache, shared by the TUI and the CLI (contract C5 v1).
package app

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/thegriglat/platui/internal/cache"
	"github.com/thegriglat/platui/internal/config"
	"github.com/thegriglat/platui/internal/forge"
	"github.com/thegriglat/platui/internal/gitbranch"
	"github.com/thegriglat/platui/internal/plane"
	"github.com/thegriglat/platui/internal/runner"
)

// Deps are the external dependencies of the scenarios.
type Deps struct {
	API    plane.API
	Git    gitbranch.Git // nil: not a git repository, git scenarios return ErrNoRepo
	Forge  forge.Tool    // nil: forge.New from cfg.GitLab
	Store  cache.Store   // nil: no cache
	Config *config.Config
	Now    func() time.Time // nil → time.Now
	Shell  runner.Shell     // nil → runner.Default()
}

// Scenario errors.
var (
	ErrNoRepo          = errors.New("not in a git repository")
	ErrNoItemForBranch = errors.New("current branch does not reference a work item")
	ErrItemNotFound    = errors.New("work item not found")
	ErrProjectNotFound = errors.New("project not found")
	ErrUnknownMember   = errors.New("unknown member")
	ErrUnknownLabel    = errors.New("unknown label")
	ErrNoTimer         = errors.New("no running timer")
)

// Meta holds the project reference data.
type Meta struct {
	Project plane.Project
	States  []plane.State
	Members []plane.Member
	Labels  []plane.Label
}

// App is the service layer; safe to call from multiple goroutines.
type App struct {
	d Deps

	mu       sync.Mutex
	me       *plane.Member
	projects []plane.Project
	meta     map[string]Meta
	items    map[string][]plane.WorkItem
	itemsAt  map[string]time.Time
	synced   map[string]time.Time // newest updated_at as of the last sync with Plane
	timers   map[string]timerRec
	mrs      map[string][]forge.MR
	marks    map[string][]string

	offline bool
	offAt   time.Time
}

// New creates an App.
func New(d Deps) *App {
	if d.Now == nil {
		d.Now = time.Now
	}
	if d.Config == nil {
		d.Config = &config.Config{}
	}
	if d.Forge == nil {
		g := d.Config.GitLab
		d.Forge = forge.New(forge.Options{Glab: g.Glab, Host: g.Host, Repo: g.Repo})
	}
	return &App{d: d, meta: map[string]Meta{}, items: map[string][]plane.WorkItem{}, itemsAt: map[string]time.Time{}, synced: map[string]time.Time{}, mrs: map[string][]forge.MR{}}
}

func (a *App) cfg() *config.Config { return a.d.Config }

func (a *App) git() (gitbranch.Git, error) {
	if a.d.Git == nil {
		return nil, ErrNoRepo
	}
	return a.d.Git, nil
}

func (a *App) key(parts ...string) string {
	return a.cfg().Workspace + "/" + strings.Join(parts, "/")
}

func (a *App) load(key string, v any) (time.Time, bool) {
	if a.d.Store == nil {
		return time.Time{}, false
	}
	t, ok, err := a.d.Store.Load(key, v)
	return t, ok && err == nil
}

// save: a cache failure does not break the scenario.
func (a *App) save(key string, v any) {
	if a.d.Store != nil {
		_ = a.d.Store.Save(key, v)
	}
}

// Me returns the current user; one request per App lifetime.
func (a *App) Me(ctx context.Context) (plane.Member, error) {
	a.mu.Lock()
	if a.me != nil {
		m := *a.me
		a.mu.Unlock()
		return m, nil
	}
	a.mu.Unlock()
	m, err := a.d.API.Me(ctx)
	if err != nil {
		var c plane.Member
		if at, ok := a.fallback(err, a.key("me"), &c); ok {
			a.markOffline(at)
			return c, nil
		}
		return plane.Member{}, err
	}
	a.mu.Lock()
	a.me = &m
	a.mu.Unlock()
	a.online()
	a.save(a.key("me"), m)
	return m, nil
}

// Projects returns the workspace projects (in memory for the App lifetime).
func (a *App) Projects(ctx context.Context) ([]plane.Project, error) {
	a.mu.Lock()
	if a.projects != nil {
		p := append([]plane.Project(nil), a.projects...)
		a.mu.Unlock()
		return p, nil
	}
	a.mu.Unlock()
	ps, err := a.d.API.Projects(ctx)
	if err != nil {
		var c []plane.Project
		if at, ok := a.fallback(err, a.key("projects"), &c); ok && c != nil {
			a.markOffline(at)
			return c, nil
		}
		return nil, err
	}
	a.mu.Lock()
	a.projects = append([]plane.Project{}, ps...)
	a.mu.Unlock()
	a.online()
	a.save(a.key("projects"), ps)
	return ps, nil
}

// Project finds a project by identifier, case-insensitively.
func (a *App) Project(ctx context.Context, identifier string) (plane.Project, error) {
	ps, err := a.Projects(ctx)
	if err != nil {
		return plane.Project{}, err
	}
	for _, p := range ps {
		if strings.EqualFold(p.Identifier, strings.TrimSpace(identifier)) {
			return p, nil
		}
	}
	return plane.Project{}, fmt.Errorf("%w: %s", ErrProjectNotFound, identifier)
}

// Meta returns the project reference data (memory, otherwise network).
func (a *App) Meta(ctx context.Context, projectID string) (Meta, error) {
	a.mu.Lock()
	m, ok := a.meta[projectID]
	a.mu.Unlock()
	if ok {
		return m, nil
	}
	m, err := a.fetchMeta(ctx, projectID)
	if err != nil {
		var c Meta
		if at, ok := a.fallback(err, a.key("meta", projectID), &c); ok && c.Project.ID == projectID {
			a.markOffline(at)
			return c, nil
		}
		return Meta{}, err
	}
	a.online()
	a.commitMeta(m)
	return m, nil
}

func (a *App) commitMeta(m Meta) {
	a.mu.Lock()
	a.meta[m.Project.ID] = m
	a.mu.Unlock()
	a.save(a.key("meta", m.Project.ID), m)
}

func (a *App) fetchMeta(ctx context.Context, projectID string) (Meta, error) {
	ps, err := a.Projects(ctx)
	if err != nil {
		return Meta{}, err
	}
	var m Meta
	found := false
	for _, p := range ps {
		if p.ID == projectID {
			m.Project, found = p, true
		}
	}
	if !found {
		return Meta{}, fmt.Errorf("%w: %s", ErrProjectNotFound, projectID)
	}
	if m.States, err = a.d.API.States(ctx, projectID); err != nil {
		return Meta{}, err
	}
	if m.Members, err = a.d.API.Members(ctx, projectID); err != nil {
		return Meta{}, err
	}
	if m.Labels, err = a.d.API.Labels(ctx, projectID); err != nil {
		return Meta{}, err
	}
	return m, nil
}

// Invalidate drops the project's in-memory Meta.
func (a *App) Invalidate(projectID string) {
	a.mu.Lock()
	delete(a.meta, projectID)
	a.mu.Unlock()
}

// IsNetworkError reports a network error per C10 v4 (exit code 4): *url.Error, net.Error,
// context.DeadlineExceeded, a dropped connection (io.ErrUnexpectedEOF, ECONNRESET),
// TLS/x509, APIError with status >= 500, plane.ErrRateLimited. Authorization, 404,
// cancellation, response parsing and other errors are not network errors. An error that lost its type when
// wrapped as text is recognized by the network stack's characteristic messages.
func IsNetworkError(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, plane.ErrUnauthorized) || errors.Is(err, plane.ErrNotFound) {
		return false
	}
	if errors.Is(err, plane.ErrRateLimited) {
		return true
	}
	var ae *plane.APIError
	if errors.As(err, &ae) {
		return ae.Status >= 500
	}
	var (
		ue  *url.Error
		ne  net.Error
		ua  x509.UnknownAuthorityError
		ci  x509.CertificateInvalidError
		hn  x509.HostnameError
		rh  tls.RecordHeaderError
		tlv *tls.CertificateVerificationError
	)
	if errors.As(err, &ue) || errors.As(err, &ne) || errors.As(err, &ua) || errors.As(err, &ci) ||
		errors.As(err, &hn) || errors.As(err, &rh) || errors.As(err, &tlv) ||
		errors.Is(err, context.DeadlineExceeded) || errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, io.EOF) || errors.Is(err, syscall.ECONNRESET) || errors.Is(err, syscall.ECONNREFUSED) {
		return true
	}
	msg := err.Error()
	for _, s := range []string{"dial tcp", "no such host", "connection refused", "connection reset", "i/o timeout", "tls:", "x509:"} {
		if strings.Contains(msg, s) {
			return true
		}
	}
	return false
}

// fallback reads the cache on a network error.
func (a *App) fallback(err error, key string, v any) (time.Time, bool) {
	if !IsNetworkError(err) {
		return time.Time{}, false
	}
	return a.load(key, v)
}

func (a *App) markOffline(at time.Time) {
	a.mu.Lock()
	if !a.offline || at.Before(a.offAt) {
		a.offAt = at
	}
	a.offline = true
	a.mu.Unlock()
}

func (a *App) online() {
	a.mu.Lock()
	a.offline, a.offAt = false, time.Time{}
	a.mu.Unlock()
}

// Offline is true if the last network attempt failed and the data came from the cache;
// the time is the saved time of the oldest cache served.
func (a *App) Offline() (bool, time.Time) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.offline, a.offAt
}
