package gitbranch

// Push publishes name to remote and sets upstream (`git push -u`).
func (r *Repo) Push(remote, name string) error {
	if err := ValidName(name); err != nil {
		return err
	}
	_, _, err := r.git("push", "-u", remote, name)
	return err
}

// RemoteURL returns the URL of remote.
func (r *Repo) RemoteURL(remote string) (string, error) {
	return r.gitOut("remote", "get-url", remote)
}
