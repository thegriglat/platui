package cli

import (
	"runtime/debug"
	"testing"
)

func TestContractE1Version(t *testing.T) {
	set := func(kv ...string) []debug.BuildSetting {
		var s []debug.BuildSetting
		for i := 0; i < len(kv); i += 2 {
			s = append(s, debug.BuildSetting{Key: kv[i], Value: kv[i+1]})
		}
		return s
	}
	rev := "0123456789abcdef0123"
	tests := []struct {
		name    string
		ldflags string
		bi      *debug.BuildInfo
		ok      bool
		want    string
	}{
		{"ldflags override", "9.9.9", &debug.BuildInfo{Main: debug.Module{Version: "v1.2.3"}}, true, "9.9.9"},
		{"module version", "dev", &debug.BuildInfo{Main: debug.Module{Version: "v1.2.3"}}, true, "v1.2.3"},
		{"empty ldflags module version", "", &debug.BuildInfo{Main: debug.Module{Version: "v1.2.3"}}, true, "v1.2.3"},
		{"vcs revision", "dev", &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}, Settings: set("vcs.revision", rev, "vcs.modified", "false")}, true, "devel 0123456789ab"},
		{"vcs dirty", "dev", &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}, Settings: set("vcs.revision", rev, "vcs.modified", "true")}, true, "devel 0123456789ab-dirty"},
		{"short revision", "dev", &debug.BuildInfo{Settings: set("vcs.revision", "abc")}, true, "devel abc"},
		{"no vcs", "dev", &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}}, true, "dev"},
		{"no build info", "dev", nil, false, "dev"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolveVersion(tc.ldflags, tc.bi, tc.ok); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}
