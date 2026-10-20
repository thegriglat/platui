package tui

import (
	"fmt"
	"os"
	"time"

	"github.com/thegriglat/platui/internal/text"

	"github.com/thegriglat/platui/internal/app"
)

// netState — связь с Plane и сохранённые черновики $EDITOR (C4 v8).
type netState struct {
	offline bool      // данные показаны из кэша, сеть недоступна
	offAt   time.Time // время самых старых показанных данных
	failing bool      // последнее фоновое обновление или загрузка не удались
	drafts  map[string]draft
}

// offlineText — «офлайн, данные от ЧЧ:ММ».
func (n netState) offlineText() string {
	return "офлайн, данные от " + n.offAt.Format("15:04")
}

// errText — текст ошибки загрузки для статусной строки; сетевую ошибку при наличии
// кэша помечает как офлайн.
func (s *session) errText(err error) string {
	s.net.failing = true
	msg := err.Error()
	if app.IsCertError(err) {
		msg += " (" + app.CertHint + ")"
	}
	if !app.IsNetworkError(err) {
		return msg
	}
	var at time.Time
	if off, t := s.deps.App.Offline(); off {
		at = t
	}
	if s.project != nil {
		if _, t, ok := s.deps.App.CachedItems(s.project.ID); ok && (at.IsZero() || t.Before(at)) {
			at = t
		}
	}
	if at.IsZero() {
		return msg
	}
	s.net.offline, s.net.offAt = true, at
	return s.net.offlineText() + ": " + msg
}

// restored снимает офлайн и ошибку; true, если связь действительно восстановилась.
func (s *session) restored() bool {
	was := s.net.offline || s.net.failing
	s.net.offline, s.net.failing = false, false
	return was
}

// draft — сохранённая после ошибки правка: сессия (с подсказками) и время загрузки задачи.
type draft struct {
	sess   *text.EditSession
	loaded time.Time
}

// editFailMsg — отправка текста из $EDITOR не удалась; файл сохранён.
type editFailMsg struct {
	kind, itemID string
	sess         *text.EditSession
	loaded       time.Time
	err          error
}

func draftKey(kind, itemID string) string { return kind + ":" + itemID }

func (s *session) onEditFail(m editFailMsg) {
	if s.net.drafts == nil {
		s.net.drafts = map[string]draft{}
	}
	s.net.drafts[draftKey(m.kind, m.itemID)] = draft{m.sess, m.loaded}
	s.setStatus(fmt.Sprintf("не отправлено, текст сохранён: %s: %v", m.sess.Path, m.err), true)
}

// savedDraft — сохранённая после ошибки правка (забывается при выдаче; файл
// удаляет успешная отправка, либо правка заново в редакторе по отказу).
func (s *session) savedDraft(kind, itemID string) (draft, bool) {
	k := draftKey(kind, itemID)
	d, ok := s.net.drafts[k]
	if !ok {
		return draft{}, false
	}
	delete(s.net.drafts, k)
	if _, err := os.Stat(d.sess.Path); err != nil {
		return draft{}, false
	}
	return d, true
}
