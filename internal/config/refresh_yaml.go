package config

import (
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

// UnmarshalYAML decodes refresh, accepting a bare 0 (off) as well as Go durations.
func (r *Refresh) UnmarshalYAML(n *yaml.Node) error {
	for i := 0; n.Kind == yaml.MappingNode && i+1 < len(n.Content); i += 2 {
		if k := n.Content[i].Value; k != "interval" && k != "notify" {
			return fmt.Errorf("refresh: unknown key %q (line %d)", k, n.Content[i].Line)
		}
	}
	var raw struct {
		Interval yaml.Node `yaml:"interval"`
		Notify   *bool     `yaml:"notify"`
	}
	if err := n.Decode(&raw); err != nil {
		return err
	}
	r.Notify = raw.Notify
	iv := &raw.Interval
	if iv.Kind == 0 {
		return nil
	}
	if iv.Kind == yaml.ScalarNode && iv.Value == "0" {
		var z time.Duration
		r.Interval = &z
		return nil
	}
	var s string
	if err := iv.Decode(&s); err != nil {
		return fmt.Errorf("refresh.interval: %w", err)
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("refresh.interval: %q is not a duration (e.g. 90s, 2m)", s)
	}
	r.Interval = &d
	return nil
}
