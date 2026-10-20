// Package forgetest provides a fake forge.Tool for tests.
package forgetest

import (
	"context"

	"github.com/thegriglat/platui/internal/forge"
)

// Fake records requests and returns the configured URL/MRs/error.
type Fake struct {
	URL   string
	MRs   []forge.MR
	Err   error
	Reqs  []forge.MRRequest
	Finds [][]string // per call: branches, then key
}

func (f *Fake) CreateMR(_ context.Context, r forge.MRRequest) (string, error) {
	f.Reqs = append(f.Reqs, r)
	return f.URL, f.Err
}

func (f *Fake) FindMRs(_ context.Context, _ string, branches []string, key string) ([]forge.MR, error) {
	f.Finds = append(f.Finds, append(append([]string{}, branches...), key))
	return f.MRs, f.Err
}

var _ forge.Tool = (*Fake)(nil)
