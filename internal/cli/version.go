package cli

import "runtime/debug"

// resolveVersion picks the version string: ldflags override, then module
// version from build info, then VCS revision, then "dev".
func resolveVersion(ldflags string, bi *debug.BuildInfo, ok bool) string {
	if ldflags != "dev" && ldflags != "" {
		return ldflags
	}
	if !ok || bi == nil {
		return "dev"
	}
	if v := bi.Main.Version; v != "" && v != "(devel)" {
		return v
	}
	var rev string
	var dirty bool
	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.modified":
			dirty = s.Value == "true"
		}
	}
	if rev == "" {
		return "dev"
	}
	if len(rev) > 12 {
		rev = rev[:12]
	}
	v := "devel " + rev
	if dirty {
		v += "-dirty"
	}
	return v
}

func versionString() string {
	bi, ok := debug.ReadBuildInfo()
	return resolveVersion(Version, bi, ok)
}
