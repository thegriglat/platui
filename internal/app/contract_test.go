package app

import (
	"errors"
	"github.com/thegriglat/platui/internal/forge"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/thegriglat/platui/internal/config"
	"github.com/thegriglat/platui/internal/gitbranch"
	"github.com/thegriglat/platui/internal/plane"
)

func TestContractAppMetaAndProjects(t *testing.T) {
	e := newEnv(t)
	for range 2 {
		if _, err := e.app.Me(ctx()); err != nil {
			t.Fatal(err)
		}
	}
	if e.calls("Me") != 1 {
		t.Fatalf("Me calls = %d", e.calls("Me"))
	}
	p, err := e.app.Project(ctx(), "proj")
	if err != nil || p.ID != "p1" {
		t.Fatalf("%v %v", p, err)
	}
	if _, err := e.app.Project(ctx(), "nope"); !errors.Is(err, ErrProjectNotFound) {
		t.Fatalf("%v", err)
	}
	m, err := e.app.Meta(ctx(), "p1")
	if err != nil || len(m.States) != 6 || len(m.Members) != 4 || len(m.Labels) != 2 || m.Project.Identifier != "PROJ" {
		t.Fatalf("%+v %v", m, err)
	}
	_, _ = e.app.Meta(ctx(), "p1")
	if e.calls("States") != 1 {
		t.Fatal("meta not cached")
	}
	e.app.Invalidate("p1")
	_, _ = e.app.Meta(ctx(), "p1")
	if e.calls("States") != 2 {
		t.Fatal("Invalidate did not reset")
	}
	if _, err := e.app.Meta(ctx(), "zzz"); !errors.Is(err, ErrProjectNotFound) {
		t.Fatalf("%v", err)
	}
	var saved Meta
	if _, ok, _ := e.store.Load("ws/meta/p1", &saved); !ok || saved.Project.ID != "p1" {
		t.Fatal("meta not stored")
	}
	e.api.Err["States"] = errors.New("boom")
	e.app.Invalidate("p1")
	if _, err := e.app.Meta(ctx(), "p1"); err == nil {
		t.Fatal("want error")
	}
}

func TestContractAppReload(t *testing.T) {
	e := newEnv(t)
	if _, ch, err := e.app.RefreshItems(ctx(), "p1"); err != nil || !ch.Empty() {
		t.Fatalf("first refresh: %v %+v", err, ch)
	}
	e.api.ItemsV["p1"] = append(e.api.ItemsV["p1"], plane.WorkItem{ID: "i9", ProjectID: "p1", Sequence: 9, Name: "New"})
	m, items, ch, err := e.app.Reload(ctx(), "p1")
	if err != nil || len(items) != 3 || len(ch.New) != 1 || ch.New[0].ID != "i9" || m.Project.ID != "p1" {
		t.Fatalf("%v %+v", err, ch)
	}
	before := e.calls("States")
	e.api.Err["WorkItems"] = errors.New("down")
	if _, _, _, err := e.app.Reload(ctx(), "p1"); err == nil {
		t.Fatal("want error")
	}
	got, _, ok := e.app.CachedItems("p1")
	if !ok || len(got) != 3 {
		t.Fatalf("data damaged: %v", got)
	}
	var disk []plane.WorkItem
	if _, ok, _ := e.store.Load("ws/items/p1", &disk); !ok || len(disk) != 3 {
		t.Fatal("disk damaged")
	}
	if e.calls("States") != before+1 {
		t.Fatal("Reload must hit network for meta")
	}
	e.api.Err["WorkItems"] = nil
	e.api.Err["States"] = errors.New("down")
	if _, _, _, err := e.app.Reload(ctx(), "p1"); err == nil {
		t.Fatal("want error")
	}
}

func TestContractAppStateTabs(t *testing.T) {
	e := newEnv(t)
	st := e.api.StatesV["p1"]
	names := func(l []plane.State) []string {
		var r []string
		for _, s := range l {
			r = append(r, s.Name)
		}
		return r
	}
	cfg := &config.Config{Tabs: config.Tabs{States: []string{"done", " Todo", "Missing"}}}
	if got := names(StateTabs(cfg, st)); !slices.Equal(got, []string{"Done", "Todo"}) {
		t.Fatalf("config order: %v", got)
	}
	cfg = &config.Config{Tabs: config.Tabs{HideGroups: []string{"cancelled"}}}
	shuffled := []plane.State{st[4], st[0], st[5], st[3], st[1], st[2]}
	if got := names(StateTabs(cfg, shuffled)); !slices.Equal(got, []string{"Backlog", "Todo", "In Progress", "In Review", "Done"}) {
		t.Fatalf("default: %v", got)
	}
	cfg = &config.Config{Tabs: config.Tabs{HideGroups: []string{}}}
	if got := StateTabs(cfg, st); len(got) != 6 {
		t.Fatalf("hide none: %v", names(got))
	}
	odd := []plane.State{{Name: "B", Group: "backlog"}, {Name: "Z", Group: "weird", Sequence: 9}, {Name: "A2", Group: "started", Sequence: 2}, {Name: "A1", Group: "started", Sequence: 2}}
	if got := names(StateTabs(cfg, odd)); !slices.Equal(got, []string{"Z", "B", "A1", "A2"}) {
		t.Fatalf("unknown group/sequence/name: %v", got)
	}
}

func TestContractAppSearchDocs(t *testing.T) {
	e := newEnv(t)
	m, _ := e.app.Meta(ctx(), "p1")
	docs := SearchDocs(m, e.api.ItemsV["p1"])
	d := docs[1]
	if d.Key != "PROJ-42" || d.ID != "i42" || d.Title != "Add search" {
		t.Fatalf("%+v", d)
	}
	hits, err := e.app.ServerSearch(ctx(), "login", "")
	if err != nil || len(hits) != 1 || hits[0].ProjectIdentifier != "PROJ" {
		t.Fatalf("%v %v", hits, err)
	}
	if e.calls("Search:login:") != 1 {
		t.Fatal("search not called")
	}
}

func TestContractAppFind(t *testing.T) {
	e := newEnv(t)
	p, it, err := e.app.Find(ctx(), "oth", 7)
	if err != nil || p.ID != "p2" || it.ID != "o7" {
		t.Fatalf("%v %v %v", p, it, err)
	}
	if e.calls("WorkItemByKey") != 1 {
		t.Fatal("want WorkItemByKey for uncached project")
	}
	_, _, _ = e.app.RefreshItems(ctx(), "p1")
	if _, it, err = e.app.Find(ctx(), "PROJ", 42); err != nil || it.ID != "i42" || e.calls("WorkItemByKey") != 1 {
		t.Fatalf("%v %v", it, err)
	}
	if _, _, err = e.app.Find(ctx(), "PROJ", 99); !errors.Is(err, ErrItemNotFound) {
		t.Fatalf("%v", err)
	}
	if _, _, err = e.app.Find(ctx(), "NOPE", 1); !errors.Is(err, ErrProjectNotFound) {
		t.Fatalf("%v", err)
	}
}

func TestContractAppStart(t *testing.T) {
	t.Run("new branch", func(t *testing.T) {
		e := newEnv(t)
		e.cfg.Workflow.StartState = "in progress"
		e.git.DirtyV = true
		r, err := e.app.Start(ctx(), e.project(), e.item("i1"), StartOptions{})
		if err != nil || r.Branch.Name != "PROJ-1-fix-login" || !r.Branch.Created || r.State != "In Progress" || !r.Assigned {
			t.Fatalf("%+v %v", r, err)
		}
		if e.git.CommitMsgs["PROJ-1-fix-login"] != "PROJ-1: Fix login" || !e.git.HookInstalled || len(r.Warnings) != 1 {
			t.Fatalf("%v %+v", e.git.CommitMsgs, r)
		}
		got := e.item("i1")
		if got.StateID != "s-prog" || !slices.Equal(got.AssigneeIDs, []string{"u-me"}) || e.calls("UpdateWorkItem") != 1 {
			t.Fatalf("%+v", got)
		}
	})
	t.Run("existing branch switches", func(t *testing.T) {
		e := newEnv(t)
		e.git.BranchesV["PROJ-1-fix-login"] = gitbranchtestStatus("PROJ-1-fix-login")
		r, err := e.app.Start(ctx(), e.project(), e.item("i1"), StartOptions{})
		if err != nil || r.Branch.Created || !r.Branch.Switched || e.git.CurrentV != "PROJ-1-fix-login" {
			t.Fatalf("%+v %v", r, err)
		}
		// группа started: первое состояние группы
		if e.item("i1").StateID != "s-prog" {
			t.Fatal("state by group")
		}
	})
	t.Run("worktree and custom branch", func(t *testing.T) {
		e := newEnv(t)
		wt := true
		r, err := e.app.Start(ctx(), e.project(), e.item("i1"), StartOptions{Branch: "feat/x", Worktree: &wt})
		if err != nil || r.Branch.Path != "/src/repo-feat-x" && r.Branch.Path != "/src/../repo-feat-x" && !strings.HasSuffix(r.Branch.Path, "/repo-feat-x") {
			t.Fatalf("%+v %v", r, err)
		}
		if e.git.CurrentV != "main" {
			t.Fatal("worktree must not switch")
		}
	})
	t.Run("already assigned and in state", func(t *testing.T) {
		e := newEnv(t)
		e.api.ItemsV["p1"][0].StateID = "s-prog"
		e.api.ItemsV["p1"][0].AssigneeIDs = []string{"u-me"}
		r, err := e.app.Start(ctx(), e.project(), e.item("i1"), StartOptions{})
		if err != nil || r.State != "" || r.Assigned || e.calls("UpdateWorkItem") != 0 {
			t.Fatalf("%+v %v", r, err)
		}
	})
	t.Run("foreign hook", func(t *testing.T) {
		e := newEnv(t)
		e.git.Err["InstallHook"] = gitbranch.ErrHookExists
		r, err := e.app.Start(ctx(), e.project(), e.item("i1"), StartOptions{})
		if err != nil || len(r.Warnings) != 1 || !strings.Contains(r.Warnings[0], "hook") {
			t.Fatalf("%+v %v", r, err)
		}
	})
	t.Run("git error leaves plane untouched", func(t *testing.T) {
		e := newEnv(t)
		e.git.Err["Create"] = errors.New("git failed")
		_, err := e.app.Start(ctx(), e.project(), e.item("i1"), StartOptions{})
		if err == nil || e.calls("UpdateWorkItem") != 0 {
			t.Fatalf("%v", err)
		}
	})
	t.Run("api error after git", func(t *testing.T) {
		e := newEnv(t)
		e.api.Err["UpdateWorkItem"] = errors.New("api down")
		r, err := e.app.Start(ctx(), e.project(), e.item("i1"), StartOptions{})
		if err == nil || r.Branch.Name == "" {
			t.Fatalf("%+v %v", r, err)
		}
	})
	t.Run("no repo, checkout off", func(t *testing.T) {
		e := newEnv(t)
		a := New(Deps{API: e.api, Config: e.cfg})
		if _, err := a.Start(ctx(), e.project(), e.item("i1"), StartOptions{}); !errors.Is(err, ErrNoRepo) {
			t.Fatalf("%v", err)
		}
		off := false
		e.cfg.Branch.Checkout = &off
		e.git.BranchesV["PROJ-1-fix-login"] = gitbranchtestStatus("PROJ-1-fix-login")
		if r, err := e.app.Start(ctx(), e.project(), e.item("i1"), StartOptions{}); err != nil || r.Branch.Switched {
			t.Fatalf("%+v %v", r, err)
		}
	})
}

func TestContractAppReview(t *testing.T) {
	setup := func() *env {
		e := newEnv(t)
		e.git.CurrentV = "PROJ-1-fix-login"
		e.cfg.GitLab = config.GitLab{TargetBranch: "main", Draft: true}
		return e
	}
	t.Run("full", func(t *testing.T) {
		e := setup()
		r, err := e.app.Review(ctx(), e.project(), e.item("i1"))
		if err != nil || r.MRURL != "https://gh/pr/1" || r.State != "In Review" || len(r.Warnings) != 0 {
			t.Fatalf("%+v %v", r, err)
		}
		if !slices.Equal(e.git.Pushed, []string{"origin/PROJ-1-fix-login"}) || len(e.forge.Reqs) != 1 {
			t.Fatalf("%v %v", e.git.Pushed, e.forge.Reqs)
		}
		q := e.forge.Reqs[0]
		if q.Head != "PROJ-1-fix-login" || q.Target != "main" || !q.Draft || q.Dir != "/src/repo" || q.Title != "PROJ-1: Fix login" ||
			q.Body != "https://app.plane.so/ws/browse/PROJ-1/" {
			t.Fatalf("%+v", q)
		}
		cm := e.api.CommentsV["i1"]
		if len(cm) != 1 || cm[0].HTML != `<p>MR: <a href="https://gh/pr/1">https://gh/pr/1</a></p>` {
			t.Fatalf("%+v", cm)
		}
		if e.item("i1").StateID != "s-rev" {
			t.Fatal("state")
		}
	})
	t.Run("pr error is warning", func(t *testing.T) {
		e := setup()
		e.forge.Err = errors.New("glab broke")
		r, err := e.app.Review(ctx(), e.project(), e.item("i1"))
		if err != nil || len(r.Warnings) != 1 || len(e.api.CommentsV["i1"]) != 0 || r.State != "In Review" {
			t.Fatalf("%+v %v", r, err)
		}
	})
	t.Run("push error", func(t *testing.T) {
		e := setup()
		e.git.Err["Push"] = errors.New("rejected")
		if _, err := e.app.Review(ctx(), e.project(), e.item("i1")); err == nil || len(e.forge.Reqs) != 0 {
			t.Fatalf("%v", err)
		}
	})
	t.Run("detached", func(t *testing.T) {
		e := setup()
		e.git.CurrentV = ""
		if _, err := e.app.Review(ctx(), e.project(), e.item("i1")); !errors.Is(err, gitbranch.ErrDetached) {
			t.Fatalf("%v", err)
		}
	})
	t.Run("review state by config and missing", func(t *testing.T) {
		e := setup()
		e.cfg.Workflow.ReviewState = "done"
		if r, err := e.app.Review(ctx(), e.project(), e.item("i1")); err != nil || r.State != "Done" {
			t.Fatalf("%+v %v", r, err)
		}
		e = setup()
		e.cfg.Workflow.ReviewState = "QA"
		r, err := e.app.Review(ctx(), e.project(), e.item("i1"))
		if err != nil || r.State != "" || len(r.Warnings) != 1 || e.calls("UpdateWorkItem") != 0 {
			t.Fatalf("%+v %v", r, err)
		}
	})
	t.Run("forge not found is warning", func(t *testing.T) {
		e := setup()
		e.forge.Err = forge.ErrNotFound
		if r, err := e.app.Review(ctx(), e.project(), e.item("i1")); err != nil || len(r.Warnings) != 1 || r.MRURL != "" {
			t.Fatalf("%+v %v", r, err)
		}
	})
	t.Run("default forge factory and no repo", func(t *testing.T) {
		e := setup()
		a := New(Deps{API: e.api, Git: e.git, Store: e.store, Config: e.cfg})
		if a.d.Forge == nil {
			t.Fatal("default forge")
		}
		if _, err := New(Deps{API: e.api, Config: e.cfg}).Review(ctx(), e.project(), e.item("i1")); !errors.Is(err, ErrNoRepo) {
			t.Fatalf("%v", err)
		}
	})
}

func TestContractAppCreateQuick(t *testing.T) {
	e := newEnv(t)
	it, err := e.app.CreateQuick(ctx(), "p1", "fix it !high @me @anna @Anna #bug #ui", "# Desc")
	if err != nil || it.Name != "fix it" || it.Priority != "high" || !strings.Contains(it.DescHTML, "<h1") ||
		!slices.Equal(it.AssigneeIDs, []string{"u-me", "u-anna"}) || !slices.Equal(it.LabelIDs, []string{"l-bug", "l-ui"}) {
		t.Fatalf("%+v %v", it, err)
	}
	it, err = e.app.CreateQuick(ctx(), "p1", "plain", "")
	if err != nil || it.Priority != "none" || it.DescHTML != "" || len(it.AssigneeIDs) != 0 {
		t.Fatalf("%+v %v", it, err)
	}
	if it, err = e.app.CreateQuick(ctx(), "p1", "by email @anna", ""); err != nil || it.AssigneeIDs[0] != "u-anna" {
		t.Fatalf("%+v %v", it, err)
	}
	cases := []struct {
		line string
		want error
		in   string
	}{
		{"x @ghost", ErrUnknownMember, "ghost"},
		{"x @bob", ErrUnknownMember, "bob"}, // неоднозначно
		{"x #nolabel", ErrUnknownLabel, "nolabel"},
	}
	n := len(e.api.ItemsV["p1"])
	for _, c := range cases {
		_, err := e.app.CreateQuick(ctx(), "p1", c.line, "")
		if !errors.Is(err, c.want) || !strings.Contains(err.Error(), c.in) {
			t.Fatalf("%s: %v", c.line, err)
		}
	}
	if len(e.api.ItemsV["p1"]) != n {
		t.Fatal("must not create on error")
	}
	if _, err := e.app.CreateQuick(ctx(), "p1", "  !high ", ""); err == nil {
		t.Fatal("empty title")
	}
	if _, err := e.app.CreateQuick(ctx(), "zzz", "x", ""); err == nil {
		t.Fatal("unknown project")
	}
}

func TestContractAppDiffAndPoll(t *testing.T) {
	a := plane.WorkItem{ID: "a", UpdatedAt: t0}
	b := plane.WorkItem{ID: "b", UpdatedAt: t0}
	b2 := plane.WorkItem{ID: "b", UpdatedAt: t0.Add(time.Minute)}
	c := plane.WorkItem{ID: "c"}
	ch := Diff([]plane.WorkItem{a, b}, []plane.WorkItem{b2, c})
	if len(ch.New) != 1 || ch.New[0].ID != "c" || len(ch.Updated) != 1 || ch.Updated[0].ID != "b" || !slices.Equal(ch.Removed, []string{"a"}) {
		t.Fatalf("%+v", ch)
	}
	if !Diff([]plane.WorkItem{a}, []plane.WorkItem{a}).Empty() {
		t.Fatal("same")
	}
	base := 2 * time.Minute
	rl := &plane.APIError{Status: 429}
	other := errors.New("x")
	for _, c := range []struct {
		prev time.Duration
		err  error
		want time.Duration
	}{
		{base, nil, base}, {base, rl, 4 * time.Minute}, {4 * time.Minute, rl, 8 * time.Minute}, {8 * time.Minute, rl, 10 * time.Minute},
		{0, other, 4 * time.Minute}, {4 * time.Minute, other, 5 * time.Minute}, {time.Second, other, 4 * time.Minute},
	} {
		if got := NextPoll(base, c.prev, c.err); got != c.want {
			t.Errorf("NextPoll(%v,%v)=%v want %v", c.prev, c.err, got, c.want)
		}
	}
}

func TestContractAppRefreshItemsCache(t *testing.T) {
	e := newEnv(t)
	if _, _, ok := e.app.CachedItems("p1"); ok {
		t.Fatal("empty")
	}
	if _, _, err := e.app.RefreshItems(ctx(), "p1"); err != nil {
		t.Fatal(err)
	}
	// новый процесс: прошлое состояние — из диска
	e.now = t0.Add(time.Hour)
	e.rebuild()
	its, at, ok := e.app.CachedItems("p1")
	if !ok || len(its) != 2 || !at.Equal(t0) {
		t.Fatalf("%v %v %v", its, at, ok)
	}
	e.api.ItemsV["p1"][0].UpdatedAt = t0.Add(time.Minute)
	_, ch, err := e.app.RefreshItems(ctx(), "p1")
	if err != nil || len(ch.Updated) != 1 || ch.Updated[0].ID != "i1" {
		t.Fatalf("%+v %v", ch, err)
	}
	e.api.Err["WorkItems"] = errors.New("x")
	if _, _, err := e.app.RefreshItems(ctx(), "p1"); err == nil {
		t.Fatal("want err")
	}
}

func TestContractAppMatch(t *testing.T) {
	e := newEnv(t)
	m, _ := e.app.Meta(ctx(), "p1")
	it := plane.WorkItem{ID: "x", Name: "Fix Login bug", StateID: "s-prog", Priority: "high", AssigneeIDs: []string{"u-me"}, LabelIDs: []string{"l-bug"}}
	ok := func(f config.Filter, it plane.WorkItem, me string) bool { return Match(f, it, m, me) }
	for name, c := range map[string]struct {
		f    config.Filter
		it   plane.WorkItem
		me   string
		want bool
	}{
		"empty":         {config.Filter{}, it, "u-me", true},
		"mine yes":      {config.Filter{Mine: true}, it, "u-me", true},
		"mine no":       {config.Filter{Mine: true}, it, "u-anna", false},
		"mine unknown":  {config.Filter{Mine: true}, it, "", false},
		"group yes":     {config.Filter{StateGroups: []string{"Started"}}, it, "", true},
		"group no":      {config.Filter{StateGroups: []string{"completed"}}, it, "", false},
		"state yes":     {config.Filter{States: []string{"in progress"}}, it, "", true},
		"state no":      {config.Filter{States: []string{"done"}}, it, "", false},
		"state unknown": {config.Filter{States: []string{"done"}}, plane.WorkItem{StateID: "?"}, "", false},
		"prio yes":      {config.Filter{Priorities: []string{"urgent", "HIGH"}}, it, "", true},
		"prio no":       {config.Filter{Priorities: []string{"low"}}, it, "", false},
		"prio none":     {config.Filter{Priorities: []string{"none"}}, plane.WorkItem{}, "", true},
		"label yes":     {config.Filter{Labels: []string{"bug"}}, it, "", true},
		"label no":      {config.Filter{Labels: []string{"ui"}}, it, "", false},
		"text yes":      {config.Filter{Text: "login"}, it, "", true},
		"text no":       {config.Filter{Text: "zzz"}, it, "", false},
		"project yes":   {config.Filter{Project: "proj"}, it, "", true},
		"project no":    {config.Filter{Project: "OTH"}, it, "", false},
		"all and":       {config.Filter{Mine: true, Priorities: []string{"high"}, Text: "fix"}, it, "u-me", true},
		"one fails":     {config.Filter{Mine: true, Priorities: []string{"low"}}, it, "u-me", false},
	} {
		if got := ok(c.f, c.it, c.me); got != c.want {
			t.Errorf("%s: got %v", name, got)
		}
	}
}

func TestContractAppBranches(t *testing.T) {
	e := newEnv(t)
	e.api.ItemsV["p1"] = append(e.api.ItemsV["p1"], plane.WorkItem{ID: "i5", ProjectID: "p1", Sequence: 5, Name: "Old", StateID: "s-canc"})
	for _, b := range []string{"PROJ-42-add-search", "PROJ-5-old", "PROJ-1-fix-login", "feature/PROJ-1-extra", "OTH-7-x", "misc"} {
		e.git.BranchesV[b] = gitbranchtestStatus(b)
	}
	e.git.CurrentV = "PROJ-5-old"
	st, err := e.app.BranchStatuses(e.project(), e.api.ItemsV["p1"])
	if err != nil || len(st) != 3 || st["i1"].Name != "PROJ-1-fix-login" || st["i42"].Name != "PROJ-42-add-search" || !st["i5"].Current {
		t.Fatalf("%+v %v", st, err)
	}
	e.git.BranchesV = map[string]gitbranch.BranchStatus{}
	if st, err = e.app.BranchStatuses(e.project(), e.api.ItemsV["p1"]); err != nil || len(st) != 0 {
		t.Fatalf("%+v %v", st, err)
	}
	e.git.BranchesV["PROJ-42-add-search"] = gitbranchtestStatus("PROJ-42-add-search")
	e.git.BranchesV["PROJ-5-old"] = gitbranchtestStatus("PROJ-5-old")
	e.git.BranchesV["PROJ-1-fix-login"] = gitbranchtestStatus("PROJ-1-fix-login")
	e.git.CurrentV = "PROJ-1-fix-login"
	stale, err := e.app.StaleBranches(ctx(), e.project())
	if err != nil || len(stale) != 2 || stale[0].Branch != "PROJ-42-add-search" || stale[1].Branch != "PROJ-5-old" {
		t.Fatalf("%+v %v", stale, err)
	}
	e.git.CurrentV = "PROJ-5-old"
	if stale, _ = e.app.StaleBranches(ctx(), e.project()); len(stale) != 1 {
		t.Fatalf("current must be skipped: %+v", stale)
	}
	res := e.app.DeleteBranches([]string{"PROJ-42-add-search", "ghost", "PROJ-5-old"}, true)
	if res["PROJ-42-add-search"] != nil || res["ghost"] == nil || res["PROJ-5-old"] == nil {
		t.Fatalf("%v", res)
	}
	if name, err := e.app.BranchName(e.project(), e.item("i1")); err != nil || name != "PROJ-1-fix-login" {
		t.Fatalf("%q %v", name, err)
	}
	norepo := New(Deps{API: e.api, Config: e.cfg})
	if _, err := norepo.BranchStatuses(e.project(), nil); !errors.Is(err, ErrNoRepo) {
		t.Fatal(err)
	}
	if _, err := norepo.StaleBranches(ctx(), e.project()); !errors.Is(err, ErrNoRepo) {
		t.Fatal(err)
	}
	if r := norepo.DeleteBranches([]string{"a"}, false); !errors.Is(r["a"], ErrNoRepo) {
		t.Fatal(r)
	}
}

func TestContractAppBulkAndComments(t *testing.T) {
	e := newEnv(t)
	res := e.app.BulkPatch(ctx(), "p1", []string{"i1", "i42", "nope"}, plane.WorkItemPatch{Priority: str("low")})
	if res["i1"] != nil || res["i42"] != nil || !errors.Is(res["nope"], plane.ErrNotFound) || e.item("i1").Priority != "low" {
		t.Fatalf("%v", res)
	}
	items := e.api.ItemsV["p1"]
	res = e.app.BulkAssign(ctx(), "p1", items, "u-me", true)
	if len(res) != 2 || res["i1"] != nil || !slices.Equal(e.item("i42").AssigneeIDs, []string{"u-anna", "u-me"}) {
		t.Fatalf("%v", res)
	}
	n := e.calls("UpdateWorkItem")
	e.app.BulkAssign(ctx(), "p1", e.api.ItemsV["p1"], "u-me", true) // уже назначен — без запросов
	if e.calls("UpdateWorkItem") != n {
		t.Fatal("no-op expected")
	}
	e.app.BulkAssign(ctx(), "p1", e.api.ItemsV["p1"], "u-me", false)
	if len(e.item("i1").AssigneeIDs) != 0 || !slices.Equal(e.item("i42").AssigneeIDs, []string{"u-anna"}) {
		t.Fatal("unassign")
	}
	e.app.BulkLabel(ctx(), "p1", e.api.ItemsV["p1"], "l-ui", true)
	if !slices.Equal(e.item("i42").LabelIDs, []string{"l-bug", "l-ui"}) || !slices.Equal(e.item("i1").LabelIDs, []string{"l-ui"}) {
		t.Fatal("label add")
	}
	e.app.BulkLabel(ctx(), "p1", e.api.ItemsV["p1"], "l-ui", false)
	if len(e.item("i1").LabelIDs) != 0 {
		t.Fatal("label remove")
	}
	cc, cancel := contextCancelled()
	defer cancel()
	res = e.app.BulkPatch(cc, "p1", []string{"i1", "i42"}, plane.WorkItemPatch{Priority: str("urgent")})
	if !errors.Is(res["i1"], cc.Err()) || !errors.Is(res["i42"], cc.Err()) {
		t.Fatalf("%v", res)
	}
	cm, err := e.app.Comment(ctx(), "p1", "i1", "**hi**")
	if err != nil || !strings.Contains(cm.HTML, "<strong>hi</strong>") {
		t.Fatalf("%+v %v", cm, err)
	}
	up, err := e.app.SetDescription(ctx(), "p1", "i1", "text")
	if err != nil || !strings.Contains(up.DescHTML, "<p>text</p>") {
		t.Fatalf("%+v %v", up, err)
	}
}

func TestContractAppTimer(t *testing.T) {
	e := newEnv(t)
	_ = e.app.StartTimer("p1", "i42")
	e.now = t0.Add(time.Hour)
	_ = e.app.StartTimer("p1", "i1")
	_ = e.app.StartTimer("p1", "i42") // no-op
	ts := e.app.Timers()
	if len(ts) != 2 || ts[0].ItemID != "i42" || !ts[0].Started.Equal(t0) || ts[1].ItemID != "i1" {
		t.Fatalf("%+v", ts)
	}
	e.rebuild() // персистентность через Store
	if got := e.app.Timers(); len(got) != 2 {
		t.Fatalf("persist: %+v", got)
	}
	e.now = t0.Add(85*time.Minute + 10*time.Second)
	e.api.Err["CreateComment"] = errors.New("net")
	if _, err := e.app.StopTimer(ctx(), "p1", "i42", ""); err == nil || len(e.app.Timers()) != 2 {
		t.Fatal("timer must survive a failed comment")
	}
	e.api.Err["CreateComment"] = nil
	d, err := e.app.StopTimer(ctx(), "p1", "i42", "- did it")
	if err != nil || d != 86*time.Minute {
		t.Fatalf("%v %v", d, err)
	}
	cm := e.api.CommentsV["i42"]
	if len(cm) != 1 || !strings.HasPrefix(cm[0].HTML, "<p>Затрачено: 1h 26m</p>") || !strings.Contains(cm[0].HTML, "<li>did it</li>") {
		t.Fatalf("%+v", cm)
	}
	if len(e.app.Timers()) != 1 {
		t.Fatal("timer not removed")
	}
	if _, err := e.app.StopTimer(ctx(), "p1", "i42", ""); !errors.Is(err, ErrNoTimer) {
		t.Fatalf("%v", err)
	}
	e.now = e.now.Add(time.Second)
	if d, _ = e.app.StopTimer(ctx(), "p1", "i1", ""); d != 26*time.Minute {
		t.Fatalf("want 26m, got %v", d)
	}
	_ = e.app.StartTimer("p1", "i1")
	if d, _ = e.app.StopTimer(ctx(), "p1", "i1", ""); d != time.Minute {
		t.Fatalf("min 1m, got %v", d)
	}
	for in, want := range map[time.Duration]string{time.Minute: "1m", 45 * time.Minute: "45m", 85 * time.Minute: "1h 25m", 2 * time.Hour: "2h 0m", 10 * time.Second: "0m"} {
		if got := FormatDuration(in); got != want {
			t.Errorf("%v: %q", in, got)
		}
	}
	// без Store — только память
	m := New(Deps{API: e.api, Config: e.cfg, Now: func() time.Time { return t0 }})
	_ = m.StartTimer("p1", "i1")
	if len(m.Timers()) != 1 {
		t.Fatal("memory only")
	}
}

func TestContractAppPrompt(t *testing.T) {
	e := newEnv(t)
	e.git.CurrentV = "PROJ-42-add-search"
	if got := e.app.Prompt(); got != "PROJ-42" {
		t.Fatalf("%q", got)
	}
	_, _ = e.app.Projects(ctx())
	_, _ = e.app.Meta(ctx(), "p1")
	_, _, _ = e.app.RefreshItems(ctx(), "p1")
	e.rebuild() // как новый процесс: только кэш на диске
	before := len(e.api.Calls)
	if got := e.app.Prompt(); got != "PROJ-42 Done" {
		t.Fatalf("%q", got)
	}
	if len(e.api.Calls) != before {
		t.Fatal("Prompt must not touch the network")
	}
	e.git.CurrentV = "main"
	if e.app.Prompt() != "" {
		t.Fatal("no item")
	}
	e.git.CurrentV = ""
	if e.app.Prompt() != "" {
		t.Fatal("detached")
	}
	if New(Deps{API: e.api, Config: e.cfg}).Prompt() != "" {
		t.Fatal("no repo")
	}
	e.git.CurrentV = "PROJ-99-x"
	if e.app.Prompt() != "PROJ-99" {
		t.Fatalf("unknown item: %q", e.app.Prompt())
	}
}

func TestContractAppParseRef(t *testing.T) {
	for _, c := range []struct {
		in, def string
		id      string
		seq     int
		ok      bool
	}{
		{"PROJ-42", "", "PROJ", 42, true}, {"proj-42", "X", "PROJ", 42, true}, {" 42 ", "PROJ", "PROJ", 42, true},
		{"42", "", "", 0, false}, {"PROJ-", "", "", 0, false}, {"PROJ-0", "", "", 0, false}, {"abc", "PROJ", "", 0, false},
		{"PROJ--3", "", "", 0, false}, {"A1-5", "", "A1", 5, true},
	} {
		id, seq, ok := ParseRef(c.in, c.def)
		if id != c.id || seq != c.seq || ok != c.ok {
			t.Errorf("%q: %q %d %v", c.in, id, seq, ok)
		}
	}
}

func TestContractAppCurrentItemAndCopy(t *testing.T) {
	e := newEnv(t)
	e.git.CurrentV = "PROJ-42-add-search"
	p, it, err := e.app.CurrentItem(ctx())
	if err != nil || p.ID != "p1" || it.ID != "i42" {
		t.Fatalf("%v %v %v", p, it, err)
	}
	e.git.CurrentV = "main"
	if _, _, err = e.app.CurrentItem(ctx()); !errors.Is(err, ErrNoItemForBranch) {
		t.Fatalf("%v", err)
	}
	if _, _, err = New(Deps{API: e.api, Config: e.cfg}).CurrentItem(ctx()); !errors.Is(err, ErrNoRepo) {
		t.Fatalf("%v", err)
	}
	e.git.CurrentV = ""
	if _, _, err = e.app.CurrentItem(ctx()); !errors.Is(err, gitbranch.ErrDetached) {
		t.Fatalf("%v", err)
	}
	c := e.app.CopyItems(e.project(), e.item("i42"))
	want := CopyItems{ID: "PROJ-42", URL: "https://app.plane.so/ws/browse/PROJ-42/", Title: "Add search", Branch: "PROJ-42-add-search",
		Markdown: "[PROJ-42](https://app.plane.so/ws/browse/PROJ-42/)"}
	if c != want {
		t.Fatalf("%+v", c)
	}
}

func TestContractAppBranchStatusesNoPerItemGit(t *testing.T) {
	e := newEnv(t)
	var items []plane.WorkItem
	for i := 1; i <= 200; i++ {
		items = append(items, plane.WorkItem{ID: "x" + itoa(i), Sequence: i, Name: "Task " + itoa(i)})
	}
	e.git.BranchesV["PROJ-7-task-7"] = gitbranchtestStatus("PROJ-7-task-7")
	st, err := e.app.BranchStatuses(e.project(), items)
	if err != nil || len(st) != 1 || st["x7"].Name != "PROJ-7-task-7" {
		t.Fatalf("%v %v", st, err)
	}
	if len(e.git.Calls) > 3 {
		t.Fatalf("git calls grow per item: %d", len(e.git.Calls))
	}
}

func TestContractAppV2ItemUpdateComments(t *testing.T) {
	e := newEnv(t)
	if _, _, err := e.app.RefreshItems(ctx(), "p1"); err != nil {
		t.Fatal(err)
	}
	// Item: замена в памяти и кэше
	e.api.ItemsV["p1"][0].Name = "Renamed"
	it, err := e.app.Item(ctx(), "p1", "i1")
	if err != nil || it.Name != "Renamed" {
		t.Fatalf("Item: %v %v", it, err)
	}
	its, _, _ := e.app.CachedItems("p1")
	if its[0].Name != "Renamed" {
		t.Fatal("memory not replaced")
	}
	var disk []plane.WorkItem
	if _, ok := e.app.load(e.app.itemsKey("p1"), &disk); !ok || disk[0].Name != "Renamed" {
		t.Fatal("cache not replaced")
	}
	// Update
	u, err := e.app.Update(ctx(), "p1", "i1", plane.WorkItemPatch{Name: str("Again")})
	if err != nil || u.Name != "Again" {
		t.Fatalf("Update: %v %v", u, err)
	}
	if its, _, _ = e.app.CachedItems("p1"); its[0].Name != "Again" {
		t.Fatal("update not in memory")
	}
	// без загруженного списка — не падает, кэш не создаётся
	if _, err := e.app.Update(ctx(), "p2", "o7", plane.WorkItemPatch{Name: str("X")}); err != nil {
		t.Fatal(err)
	}
	if _, _, ok := e.app.CachedItems("p2"); ok {
		t.Fatal("cache created")
	}
	// ошибки
	if _, err := e.app.Item(ctx(), "p1", "nope"); err == nil {
		t.Fatal("want error")
	}
	if _, err := e.app.Update(ctx(), "p1", "nope", plane.WorkItemPatch{}); err == nil {
		t.Fatal("want error")
	}
	// Comments: сортировка
	e.api.CommentsV["i1"] = []plane.Comment{{ID: "b", CreatedAt: t0.Add(time.Hour)}, {ID: "a", CreatedAt: t0}}
	cs, err := e.app.Comments(ctx(), "p1", "i1")
	if err != nil || len(cs) != 2 || cs[0].ID != "a" {
		t.Fatalf("Comments: %v %v", cs, err)
	}
	if _, err := e.app.Comments(ctx(), "p1", "i1"); err != nil {
		t.Fatal(err)
	}
}
