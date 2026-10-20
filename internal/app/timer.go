package app

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/thegriglat/platui/internal/text"
)

type timerRec struct {
	ProjectID string    `json:"project_id"`
	Started   time.Time `json:"started"`
}

// Timer — запущенный таймер задачи.
type Timer struct {
	ProjectID, ItemID string
	Started           time.Time
}

// loadTimers лениво читает таймеры из Store. Вызывать под a.mu.
func (a *App) loadTimers() {
	if a.timers != nil {
		return
	}
	a.timers = map[string]timerRec{}
	var m map[string]timerRec
	if _, ok := a.load(a.key("timers"), &m); ok {
		for k, v := range m {
			a.timers[k] = v
		}
	}
}

// saveTimers пишет таймеры в Store. Вызывать под a.mu.
func (a *App) saveTimers() { a.save(a.key("timers"), a.timers) }

// StartTimer запускает таймер; уже идёт — ничего не меняет.
func (a *App) StartTimer(projectID, itemID string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.loadTimers()
	if _, ok := a.timers[itemID]; ok {
		return nil
	}
	a.timers[itemID] = timerRec{ProjectID: projectID, Started: a.d.Now()}
	a.saveTimers()
	return nil
}

// Timers — запущенные таймеры по времени старта.
func (a *App) Timers() []Timer {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.loadTimers()
	out := make([]Timer, 0, len(a.timers))
	for id, r := range a.timers {
		out = append(out, Timer{ProjectID: r.ProjectID, ItemID: id, Started: r.Started})
	}
	slices.SortFunc(out, func(x, y Timer) int {
		if c := x.Started.Compare(y.Started); c != 0 {
			return c
		}
		return compareStr(x.ItemID, y.ItemID)
	})
	return out
}

func compareStr(a, b string) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

// StopTimer комментирует затраченное время и удаляет таймер (только после успешного комментария).
func (a *App) StopTimer(ctx context.Context, projectID, itemID, note string) (time.Duration, error) {
	a.mu.Lock()
	a.loadTimers()
	r, ok := a.timers[itemID]
	a.mu.Unlock()
	if !ok {
		return 0, ErrNoTimer
	}
	d := max((a.d.Now().Sub(r.Started) + time.Minute - 1).Truncate(time.Minute), time.Minute)
	h := fmt.Sprintf("<p>Затрачено: %s</p>", FormatDuration(d))
	if note != "" {
		h += text.MarkdownToHTML(note)
	}
	if _, err := a.d.API.CreateComment(ctx, projectID, itemID, h); err != nil {
		return 0, err
	}
	a.mu.Lock()
	delete(a.timers, itemID)
	a.saveTimers()
	a.mu.Unlock()
	return d, nil
}

// FormatDuration — "1h 25m", "45m".
func FormatDuration(d time.Duration) string {
	m := int(d / time.Minute)
	if m >= 60 {
		return fmt.Sprintf("%dh %dm", m/60, m%60)
	}
	return fmt.Sprintf("%dm", m)
}
