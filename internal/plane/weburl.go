package plane

import (
	"fmt"
)

// WebURL — ссылка на задачу в веб-интерфейсе.
func (c *Client) WebURL(p Project, it WorkItem) string {
	base := c.web
	if base == "" {
		base = c.base
		if base == "https://api.plane.so" {
			base = "https://app.plane.so"
		}
	}
	return fmt.Sprintf("%s/%s/browse/%s-%d/", base, c.ws, p.Identifier, it.Sequence)
}
