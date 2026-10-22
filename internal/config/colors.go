package config

// ColorKeys lists the valid theme.colors keys.
var ColorKeys = []string{"accent", "muted", "text", "error", "success", "warning",
	"selection", "border", "bar", "priority_urgent", "priority_high", "priority_medium", "priority_low"}

var defaultColors = map[string]string{
	"accent": "#7D56F4", "muted": "245", "text": "252", "error": "#FF5F87", "success": "#5FD787",
	"warning": "#FFD75F", "selection": "#3A3A5A", "border": "240", "bar": "236",
	"priority_urgent": "#FF5F5F", "priority_high": "#FFAF5F", "priority_medium": "#FFD75F", "priority_low": "#87AFFF",
}

// Color returns the color for key (theme override or package default); unknown key gives "".
func (c *Config) Color(key string) string {
	if v, ok := c.Theme.Colors[key]; ok && contains(ColorKeys, key) {
		return v
	}
	return defaultColors[key]
}
