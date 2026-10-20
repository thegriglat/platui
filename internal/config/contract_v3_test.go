package config

import (
	"reflect"
	"strings"
	"testing"
)

const v3Base = "api_key: k\nworkspace: w\n"

func TestContractConfigV3Defaults(t *testing.T) {
	c, err := Parse([]byte(v3Base))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(c.Tabs.HideGroups, []string{"cancelled"}) || !c.TabsAll() || len(c.Tabs.States) != 0 {
		t.Fatalf("%+v", c.Tabs)
	}
}

func TestContractConfigV3Explicit(t *testing.T) {
	c, err := Parse([]byte(v3Base + "tabs:\n  states: [Todo, Done]\n  hide_groups: []\n  all: false\n"))
	if err != nil {
		t.Fatal(err)
	}
	if c.Tabs.HideGroups == nil || len(c.Tabs.HideGroups) != 0 {
		t.Fatalf("hide_groups: %#v", c.Tabs.HideGroups)
	}
	if c.TabsAll() || !reflect.DeepEqual(c.Tabs.States, []string{"Todo", "Done"}) {
		t.Fatalf("%+v", c.Tabs)
	}
	c, err = Parse([]byte(v3Base + "tabs:\n  hide_groups: [backlog, cancelled]\n  all: true\n"))
	if err != nil || !c.TabsAll() || len(c.Tabs.HideGroups) != 2 {
		t.Fatalf("%v %+v", err, c)
	}
}

func TestContractConfigV3Errors(t *testing.T) {
	cases := map[string]string{
		"tabs:\n  hide_groups: [bogus]\n": "bogus",
		"tabs:\n  states: [A, \"\"]\n":    "empty",
		"tabs:\n  states: [Todo, todo]\n": "duplicate",
		"tabs:\n  unknown_key: 1\n":       "unknown_key",
	}
	for y, sub := range cases {
		_, err := Parse([]byte(v3Base + y))
		if err == nil || !strings.Contains(err.Error(), sub) {
			t.Errorf("%q: err=%v want %q", y, err, sub)
		}
	}
}
