package gitbranch

import "testing"

func TestContractGitbranchV5(t *testing.T) {
	dir := newRepo(t)
	run(t, dir, "branch", "-M", "main")
	r := openRepo(t, dir)
	run(t, dir, "checkout", "-q", "-b", "feat")
	commitFile(t, dir, "a", "1\n2\n3\n")
	commitFile(t, dir, "b", "x\n")
	commitFile(t, dir, "f", "")
	run(t, dir, "checkout", "-q", "main")
	commitFile(t, dir, "other", "o\n") // base moves on: three-dot ignores it
	for _, base := range []string{"", "main"} {
		st, err := r.DiffStat(base, "feat")
		if err != nil {
			t.Fatal(err)
		}
		if want := (Stat{3, 4, 1}); st != want {
			t.Fatalf("base %q: got %+v want %+v", base, st, want)
		}
	}
	run(t, dir, "branch", "same")
	if st, err := r.DiffStat("", "same"); err != nil || st != (Stat{}) {
		t.Fatalf("same: %+v %v", st, err)
	}
	if _, err := r.DiffStat("", "nope"); err == nil {
		t.Fatal("missing branch: want error")
	}
}

func TestContractGitbranchV5Parse(t *testing.T) {
	cases := map[string]Stat{
		"":                                 {},
		" 1 file changed, 2 insertions(+)": {1, 2, 0},
		" 2 files changed, 3 deletions(-)": {2, 0, 3},
		" 3 files changed, 4 insertions(+), 5 deletions(-)": {3, 4, 5},
	}
	for in, want := range cases {
		if got := parseShortstat(in); got != want {
			t.Errorf("%q: %+v want %+v", in, got, want)
		}
	}
}
