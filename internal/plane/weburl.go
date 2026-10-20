package plane

import (
	"fmt"
)

// WebURL is the link to a task in the web interface.
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
