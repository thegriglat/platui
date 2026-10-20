package app

import "github.com/thegriglat/platui/internal/gitbranch"

func gitbranchtestStatus(n string) gitbranch.BranchStatus {
	return gitbranch.BranchStatus{Name: n, Local: true}
}
