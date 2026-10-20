package tui

import (
	"github.com/thegriglat/platui/internal/config"
)

const previewStep = 5 // шаг изменения ширины превью, %

// panelState — сессионная раскладка панелей: ширина превью (%) и режим «превью на весь экран» (C4 v6, фича 14).
// Хранится в session.panel.
type panelState struct {
	pct  int
	zoom bool
}

func clampPct(p int) int {
	return max(config.MinPreviewWidth, min(p, config.MaxPreviewWidth))
}

// resizePreview меняет ширину превью на delta процентов в пределах 20..80.
func (s *session) resizePreview(delta int) {
	p := &s.panel
	p.pct = clampPct(p.pct + delta)
	s.setStatus("превью: "+itoa(p.pct)+"%", false)
}

// toggleZoom — превью на весь экран и обратно.
func (s *session) toggleZoom() {
	p := &s.panel
	p.zoom = !p.zoom
	if p.zoom {
		s.focus = 1
	}
}

// panelKey обрабатывает клавиши раскладки главного экрана; true — клавиша обработана.
func (s *session) panelKey(k string) bool {
	switch k {
	case "<":
		s.resizePreview(-previewStep)
	case ">":
		s.resizePreview(previewStep)
	case "z":
		s.toggleZoom()
	case "esc":
		if !s.panel.zoom {
			return false
		}
		s.panel.zoom = false
	default:
		return false
	}
	return true
}
