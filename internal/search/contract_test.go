package search

import (
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/junegunn/fzf/src/algo"
	"github.com/junegunn/fzf/src/util"
)

func fixture() *Index {
	return New([]Doc{
		{ID: "a", Key: "PROJ-1", Title: "Login page"},
		{ID: "b", Key: "PROJ-42", Title: "Настройка кэша"},
		{ID: "c", Key: "PROJ-420", Title: "Refactor"},
		{ID: "d", Key: "OTHER-7", Title: "Login flow"},
	})
}

func ids(h []Hit) []string {
	var r []string
	for _, x := range h {
		r = append(r, x.ID)
	}
	return r
}

func TestContractSearchAnd(t *testing.T) {
	ix := fixture()
	if got := ids(ix.Search("login page")); !reflect.DeepEqual(got, []string{"a"}) {
		t.Fatalf("got %v", got)
	}
	if got := ids(ix.Search("lgn")); !reflect.DeepEqual(got, []string{"a", "d"}) {
		t.Fatalf("got %v", got)
	}
	if ix.Len() != 4 {
		t.Fatal("len")
	}
}

func TestContractSearchRanking(t *testing.T) {
	ix := New([]Doc{{ID: "sparse", Title: "a x b y c"}, {ID: "tight", Title: "zz abc"}})
	if got := ids(ix.Search("abc")); !reflect.DeepEqual(got, []string{"tight", "sparse"}) {
		t.Fatalf("consecutive: %v", got)
	}
	ix = New([]Doc{{ID: "mid", Title: "xxfoo"}, {ID: "start", Title: "xx foo"}})
	if got := ids(ix.Search("foo")); !reflect.DeepEqual(got, []string{"start", "mid"}) {
		t.Fatalf("word start: %v", got)
	}
}

func TestContractSearchStableOrder(t *testing.T) {
	ix := New([]Doc{{ID: "1", Title: "same"}, {ID: "2", Title: "same"}, {ID: "3", Title: "same"}})
	if got := ids(ix.Search("same")); !reflect.DeepEqual(got, []string{"1", "2", "3"}) {
		t.Fatalf("%v", got)
	}
}

func TestContractSearchPos(t *testing.T) {
	ix := New([]Doc{{ID: "a", Title: "Настройка кэша"}, {ID: "b", Title: "fix cache"}})
	h := ix.Search("кэш")
	if len(h) != 1 || !reflect.DeepEqual(h[0].Pos, []int{10, 11, 12}) {
		t.Fatalf("%+v", h)
	}
	h = ix.Search("fix cac")
	if len(h) != 1 || !reflect.DeepEqual(h[0].Pos, []int{0, 1, 2, 4, 5, 6}) {
		t.Fatalf("%+v", h)
	}
	if h = ix.Search("ФИКС"); len(h) != 0 {
		t.Fatalf("%+v", h)
	}
	if h = New([]Doc{{ID: "x", Title: "КЭШ"}}).Search("кэш"); len(h) != 1 {
		t.Fatal("case")
	}
}

func TestContractSearchEmptyAndKeys(t *testing.T) {
	ix := fixture()
	for _, q := range []string{"", "   ", "PROJ-42", "proj-42", "42", "#42"} {
		if h := ix.Search(q); h != nil {
			t.Fatalf("%q: %v", q, h)
		}
	}
}

func TestContractSearchParseKey(t *testing.T) {
	cases := []struct {
		q     string
		ident string
		seq   int
		ok    bool
	}{
		{"PROJ-42", "PROJ", 42, true}, {"proj-42", "PROJ", 42, true}, {"42", "", 42, true},
		{"#42", "", 42, true}, {"  P2-7 ", "P2", 7, true},
		{"", "", 0, false}, {"abc", "", 0, false}, {"PROJ-", "", 0, false}, {"-5", "", 0, false},
		{"1PROJ-5", "", 0, false}, {"a b-4", "", 0, false}, {"4x", "", 0, false},
	}
	for _, c := range cases {
		id, n, ok := ParseKey(c.q)
		if id != c.ident || n != c.seq || ok != c.ok {
			t.Errorf("%q: %q %d %v", c.q, id, n, ok)
		}
	}
}

func TestContractSearchLookup(t *testing.T) {
	ix := fixture()
	if id, ok := ix.Lookup("proj", 42); !ok || id != "b" {
		t.Fatal(id, ok)
	}
	if id, ok := ix.Lookup("", 7); !ok || id != "d" {
		t.Fatal(id, ok)
	}
	if id, ok := ix.Lookup("", 42); !ok || id != "b" {
		t.Fatal(id, ok)
	}
	if _, ok := ix.Lookup("PROJ", 7); ok {
		t.Fatal("PROJ-7")
	}
	if _, ok := ix.Lookup("", 99); ok {
		t.Fatal("99")
	}
}

func TestContractSearchMaxQuery(t *testing.T) {
	long := strings.Repeat("a", 32)
	ix := New([]Doc{{ID: "x", Title: long}, {ID: "y", Title: long + "bbbbbbbb"}})
	if MaxQuery != 32 {
		t.Fatal("MaxQuery")
	}
	q := long + "zzzzzzzz"
	if !reflect.DeepEqual(ix.Search(q), ix.Search(long)) || len(ix.Search(q)) != 2 {
		t.Fatalf("%v", ix.Search(q))
	}
	if _, n, ok := ParseKey(strings.Repeat("1", 40)); ok {
		t.Fatalf("%d", n)
	}
}

func TestContractSearchFuzzyMatchesFzf(t *testing.T) {
	algo.Init("default")
	cases := [][2]string{
		{"abc", "a x b y c"}, {"foo", "xx foo"}, {"кэш", "Настройка кэша"},
		{"FixMe", "fix me now"}, {"zzz", "abc"}, {"ab", "FooBar ab"},
	}
	for _, c := range cases {
		in := util.RunesToChars([]rune(c[1]))
		res, pos := algo.FuzzyMatchV2(false, true, true, &in, algo.NormalizeRunes(lowerRunes(c[0])), true, util.MakeSlab(100*1024, 2048))
		sc, p, ok := Fuzzy(c[0], c[1])
		if ok != (res.Start >= 0) {
			t.Fatalf("%v ok=%v", c, ok)
		}
		if !ok {
			continue
		}
		want := append([]int(nil), *pos...)
		for i, j := 0, len(want)-1; i < j; i, j = i+1, j-1 {
			want[i], want[j] = want[j], want[i]
		}
		if sc != res.Score || !reflect.DeepEqual(p, want) {
			t.Fatalf("%v: %d %v vs %d %v", c, sc, p, res.Score, want)
		}
	}
}

func TestContractSearchHighlight(t *testing.T) {
	m := func(s string) string { return "[" + s + "]" }
	if got := Highlight("fix cache", []int{0, 1, 4, 5}, m); got != "[fi]x [ca]che" {
		t.Fatal(got)
	}
	if got := Highlight("кэш", []int{1, 2}, m); got != "к[эш]" {
		t.Fatal(got)
	}
	if got := Highlight("abc", nil, m); got != "abc" {
		t.Fatal(got)
	}
}

func TestContractSearchConcurrent(t *testing.T) {
	ix := New(genDocs(200))
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				ix.Search("fix cache")
			}
		}()
	}
	wg.Wait()
}

func TestContractSearchNormalize(t *testing.T) {
	if _, _, ok := Fuzzy("café", "café"); !ok {
		t.Fatal("café")
	}
	if h := New([]Doc{{ID: "a", Title: "Café"}}); len(h.Search("cafe")) != 1 || len(h.Search("café")) != 1 {
		t.Fatal("Café")
	}
	if h := New([]Doc{{ID: "a", Title: "Cafe"}}); len(h.Search("café")) != 1 {
		t.Fatal("Cafe")
	}
}
