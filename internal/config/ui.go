package config

import "fmt"

// UI defaults and bounds (C1 v5).
const (
	DefaultPreviewWidth = 45
	MinPreviewWidth     = 20
	MaxPreviewWidth     = 80
)

// UI configures the interface.
type UI struct {
	Mouse        *bool `yaml:"mouse"`         // nil → true
	PreviewWidth int   `yaml:"preview_width"` // percent; 0 → 45
}

// MouseEnabled reports whether mouse support is on.
func (c *Config) MouseEnabled() bool {
	return c.UI.Mouse == nil || *c.UI.Mouse
}

// PreviewPercent returns the preview width in percent (0 → 45).
func (c *Config) PreviewPercent() int {
	if c.UI.PreviewWidth == 0 {
		return DefaultPreviewWidth
	}
	return c.UI.PreviewWidth
}

func (c *Config) validateUI() error {
	if w := c.UI.PreviewWidth; w != 0 && (w < MinPreviewWidth || w > MaxPreviewWidth) {
		return fmt.Errorf("ui.preview_width: %d is out of range %d..%d", w, MinPreviewWidth, MaxPreviewWidth)
	}
	return nil
}
