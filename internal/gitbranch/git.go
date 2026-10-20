package gitbranch

// Git is the full repository interface for app, tui and cli.
type Git interface {
	Brancher
	Exists(name string) (bool, error)
	Dirty() (bool, error)
	Toplevel() (string, error)
	Current() (string, error)
	Branches() ([]string, error)
	Statuses(names []string, remote, base string) (map[string]BranchStatus, error)
	AddWorktree(name, path string, o Options) (Result, error)
	Delete(name string, force bool) error
	WriteCommitMsg(branch, msg string) error
	ReadCommitMsg(branch string) (msg string, ok bool, err error)
	InstallHook() (string, bool, error)
	Push(remote, name string) error
	RemoteURL(remote string) (string, error)
	DiffStat(base, branch string) (Stat, error)
}

var _ Git = (*Repo)(nil)
