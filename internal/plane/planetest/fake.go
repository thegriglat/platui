// Package planetest — фейк plane.API для тестов потребителей (контракт C3 v3).
package planetest

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/thegriglat/platui/internal/plane"
)

// Fake — потокобезопасная реализация plane.API в памяти.
type Fake struct {
	Mu        sync.Mutex
	MeV       plane.Member
	ProjectsV []plane.Project
	StatesV   map[string][]plane.State
	MembersV  map[string][]plane.Member
	LabelsV   map[string][]plane.Label
	ItemsV    map[string][]plane.WorkItem
	CommentsV map[string][]plane.Comment
	Err       map[string]error // имя метода → ошибка
	Calls     []string         // журнал вызовов
	Web       string           // база для WebURL; пусто — https://app.plane.so/ws
}

var _ plane.API = (*Fake)(nil)

// NewFake создаёт Fake с пустыми картами.
func NewFake() *Fake {
	return &Fake{
		StatesV: map[string][]plane.State{}, MembersV: map[string][]plane.Member{},
		LabelsV: map[string][]plane.Label{}, ItemsV: map[string][]plane.WorkItem{},
		CommentsV: map[string][]plane.Comment{}, Err: map[string]error{},
	}
}

// enter пишет вызов в журнал и возвращает заданную ошибку; Mu берётся здесь, снимается через unlock.
func (f *Fake) enter(method string, args ...string) error {
	f.Mu.Lock()
	call := method
	for _, a := range args {
		call += ":" + a
	}
	f.Calls = append(f.Calls, call)
	return f.Err[method]
}

func (f *Fake) unlock() { f.Mu.Unlock() }

// Me возвращает MeV.
func (f *Fake) Me(context.Context) (plane.Member, error) {
	defer f.unlock()
	if err := f.enter("Me"); err != nil {
		return plane.Member{}, err
	}
	return f.MeV, nil
}

// Projects возвращает копию ProjectsV.
func (f *Fake) Projects(context.Context) ([]plane.Project, error) {
	defer f.unlock()
	if err := f.enter("Projects"); err != nil {
		return nil, err
	}
	return slices.Clone(f.ProjectsV), nil
}

// States возвращает состояния проекта.
func (f *Fake) States(_ context.Context, pid string) ([]plane.State, error) {
	defer f.unlock()
	if err := f.enter("States", pid); err != nil {
		return nil, err
	}
	return slices.Clone(f.StatesV[pid]), nil
}

// Members возвращает участников проекта.
func (f *Fake) Members(_ context.Context, pid string) ([]plane.Member, error) {
	defer f.unlock()
	if err := f.enter("Members", pid); err != nil {
		return nil, err
	}
	return slices.Clone(f.MembersV[pid]), nil
}

// Labels возвращает метки проекта.
func (f *Fake) Labels(_ context.Context, pid string) ([]plane.Label, error) {
	defer f.unlock()
	if err := f.enter("Labels", pid); err != nil {
		return nil, err
	}
	return slices.Clone(f.LabelsV[pid]), nil
}

// WorkItems возвращает задачи проекта.
func (f *Fake) WorkItems(_ context.Context, pid string) ([]plane.WorkItem, error) {
	defer f.unlock()
	if err := f.enter("WorkItems", pid); err != nil {
		return nil, err
	}
	return cloneItems(f.ItemsV[pid]), nil
}

func cloneItem(it plane.WorkItem) plane.WorkItem {
	it.AssigneeIDs = slices.Clone(it.AssigneeIDs)
	it.LabelIDs = slices.Clone(it.LabelIDs)
	return it
}

func cloneItems(l []plane.WorkItem) []plane.WorkItem {
	out := slices.Clone(l)
	for i := range out {
		out[i] = cloneItem(out[i])
	}
	return out
}

// WorkItemByKey ищет по ProjectsV[].Identifier и Sequence.
func (f *Fake) WorkItemByKey(_ context.Context, identifier string, seq int) (plane.WorkItem, error) {
	defer f.unlock()
	if err := f.enter("WorkItemByKey", fmt.Sprintf("%s-%d", identifier, seq)); err != nil {
		return plane.WorkItem{}, err
	}
	for _, p := range f.ProjectsV {
		if !strings.EqualFold(p.Identifier, identifier) {
			continue
		}
		for _, it := range f.ItemsV[p.ID] {
			if it.Sequence == seq {
				return cloneItem(it), nil
			}
		}
	}
	return plane.WorkItem{}, plane.ErrNotFound
}

// Search — подстрока без учёта регистра в Name, limit<=0 → 50.
func (f *Fake) Search(_ context.Context, query, pid string, limit int) ([]plane.SearchHit, error) {
	defer f.unlock()
	if err := f.enter("Search", query, pid); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 50
	}
	ident := map[string]string{}
	for _, p := range f.ProjectsV {
		ident[p.ID] = p.Identifier
	}
	q := strings.ToLower(query)
	var out []plane.SearchHit
	pids := make([]string, 0, len(f.ItemsV))
	for k := range f.ItemsV {
		pids = append(pids, k)
	}
	slices.Sort(pids)
	for _, k := range pids {
		if pid != "" && k != pid {
			continue
		}
		for _, it := range f.ItemsV[k] {
			if len(out) >= limit {
				return out, nil
			}
			if strings.Contains(strings.ToLower(it.Name), q) {
				out = append(out, plane.SearchHit{ID: it.ID, Name: it.Name, Sequence: it.Sequence, ProjectID: k, ProjectIdentifier: ident[k]})
			}
		}
	}
	return out, nil
}

// WorkItem возвращает задачу или plane.ErrNotFound.
func (f *Fake) WorkItem(_ context.Context, pid, id string) (plane.WorkItem, error) {
	defer f.unlock()
	if err := f.enter("WorkItem", pid, id); err != nil {
		return plane.WorkItem{}, err
	}
	for _, it := range f.ItemsV[pid] {
		if it.ID == id {
			return cloneItem(it), nil
		}
	}
	return plane.WorkItem{}, plane.ErrNotFound
}

// CreateWorkItem добавляет задачу со следующим Sequence.
func (f *Fake) CreateWorkItem(_ context.Context, pid string, n plane.NewWorkItem) (plane.WorkItem, error) {
	defer f.unlock()
	if err := f.enter("CreateWorkItem", pid); err != nil {
		return plane.WorkItem{}, err
	}
	seq := 0
	for _, it := range f.ItemsV[pid] {
		seq = max(seq, it.Sequence)
	}
	now := time.Now()
	it := plane.WorkItem{
		ID: fmt.Sprintf("%s-item-%d", pid, seq+1), ProjectID: pid, Sequence: seq + 1,
		Name: n.Name, DescHTML: n.DescHTML, Priority: "none", StateID: n.StateID, ParentID: n.ParentID,
		AssigneeIDs: slices.Clone(n.AssigneeIDs), LabelIDs: slices.Clone(n.LabelIDs), CreatedAt: now, UpdatedAt: now,
	}
	if n.Priority != "" {
		it.Priority = n.Priority
	}
	f.ItemsV[pid] = append(f.ItemsV[pid], it)
	return cloneItem(it), nil
}

// UpdateWorkItem применяет заданные поля патча.
func (f *Fake) UpdateWorkItem(_ context.Context, pid, id string, p plane.WorkItemPatch) (plane.WorkItem, error) {
	defer f.unlock()
	if err := f.enter("UpdateWorkItem", pid, id); err != nil {
		return plane.WorkItem{}, err
	}
	items := f.ItemsV[pid]
	for i := range items {
		if items[i].ID != id {
			continue
		}
		if p.StateID != nil {
			items[i].StateID = *p.StateID
		}
		if p.Priority != nil {
			items[i].Priority = *p.Priority
		}
		if p.AssigneeIDs != nil {
			items[i].AssigneeIDs = slices.Clone(*p.AssigneeIDs)
		}
		if p.Name != nil {
			items[i].Name = *p.Name
		}
		if p.DescHTML != nil {
			items[i].DescHTML = *p.DescHTML
		}
		if p.LabelIDs != nil {
			items[i].LabelIDs = slices.Clone(*p.LabelIDs)
		}
		items[i].UpdatedAt = time.Now()
		return cloneItem(items[i]), nil
	}
	return plane.WorkItem{}, plane.ErrNotFound
}

// Comments возвращает комментарии задачи.
func (f *Fake) Comments(_ context.Context, pid, id string) ([]plane.Comment, error) {
	defer f.unlock()
	if err := f.enter("Comments", pid, id); err != nil {
		return nil, err
	}
	return slices.Clone(f.CommentsV[id]), nil
}

// CreateComment добавляет комментарий от MeV.
func (f *Fake) CreateComment(_ context.Context, pid, id, html string) (plane.Comment, error) {
	defer f.unlock()
	if err := f.enter("CreateComment", pid, id); err != nil {
		return plane.Comment{}, err
	}
	cm := plane.Comment{ID: fmt.Sprintf("c-%d", len(f.CommentsV[id])+1), Author: f.MeV.ID, HTML: html, CreatedAt: time.Now()}
	f.CommentsV[id] = append(f.CommentsV[id], cm)
	return cm, nil
}

// WebURL строит ссылку <Web>/browse/<Identifier>-<Sequence>/.
func (f *Fake) WebURL(p plane.Project, it plane.WorkItem) string {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	base := f.Web
	if base == "" {
		base = "https://app.plane.so/ws"
	}
	return fmt.Sprintf("%s/browse/%s-%d/", base, p.Identifier, it.Sequence)
}
