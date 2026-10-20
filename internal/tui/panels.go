package tui

import (
	"github.com/thegriglat/platui/internal/config"
)

const previewStep = 5 // preview width change step, %

// panelState is the session pane layout: preview width (%) and "preview full screen" mode (C4 v6, feature 14).
// Stored in session.panel.
type panelState struct {
	pct  int
	zoom bool
}

func clampPct(p int) int {
	return max(config.MinPreviewWidth, min(p, config.MaxPreviewWidth))
}

// resizePreview changes the preview width by delta percent within 20..80.
func (s *session) resizePreview(delta int) {
	p := &s.panel
	p.pct = clampPct(p.pct + delta)
	s.setStatus("preview: "+itoa(p.pct)+"%", false)
}

// toggleZoom switches the preview to full screen and back.
func (s *session) toggleZoom() {
	p := &s.panel
	p.zoom = !p.zoom
	if p.zoom {
		s.focus = 1
	}
}

// panelKey handles main screen layout keys; true means the key was handled.
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
