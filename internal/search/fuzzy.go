package search

import (
	"sort"
	"strings"
	"sync"
	"unicode"

	"github.com/junegunn/fzf/src/algo"
	"github.com/junegunn/fzf/src/util"
)

func init() { algo.Init("default") }

// Slab sizes are the fzf defaults.
var slabs = sync.Pool{New: func() any { return util.MakeSlab(100*1024, 2048) }}

func lowerRunes(s string) []rune {
	r := []rune(s)
	for i, c := range r {
		r[i] = unicode.ToLower(c)
	}
	return r
}

// match calls algo.FuzzyMatchV2 with positions in ascending order.
func match(pattern []rune, in *util.Chars) (int, []int, bool) {
	slab := slabs.Get().(*util.Slab)
	res, pos := algo.FuzzyMatchV2(false, true, true, in, pattern, true, slab)
	slabs.Put(slab)
	if res.Start < 0 {
		return 0, nil, false
	}
	var p []int
	if pos != nil {
		p = append(p, *pos...)
		sort.Ints(p)
	}
	return res.Score, p, true
}

// Fuzzy matches a pattern against s fuzzily (case-insensitive); pos holds rune positions in s.
func Fuzzy(pattern, s string) (score int, pos []int, ok bool) {
	in := util.RunesToChars([]rune(s))
	return match(algo.NormalizeRunes(lowerRunes(pattern)), &in)
}

// Highlight wraps consecutive matched runes of s (positions pos) in mark.
func Highlight(s string, pos []int, mark func(string) string) string {
	if len(pos) == 0 {
		return s
	}
	set := make(map[int]bool, len(pos))
	for _, p := range pos {
		set[p] = true
	}
	var b strings.Builder
	var run []rune
	flush := func() {
		if len(run) > 0 {
			b.WriteString(mark(string(run)))
			run = run[:0]
		}
	}
	for i, c := range []rune(s) {
		if set[i] {
			run = append(run, c)
			continue
		}
		flush()
		b.WriteRune(c)
	}
	flush()
	return b.String()
}
