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

// Размеры слэба как у fzf по умолчанию.
var slabs = sync.Pool{New: func() any { return util.MakeSlab(100*1024, 2048) }}

func lowerRunes(s string) []rune {
	r := []rune(s)
	for i, c := range r {
		r[i] = unicode.ToLower(c)
	}
	return r
}

// match — вызов algo.FuzzyMatchV2 с позициями по возрастанию.
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

// Fuzzy — нечёткое совпадение шаблона с s (регистр не важен); pos — позиции рун s.
func Fuzzy(pattern, s string) (score int, pos []int, ok bool) {
	in := util.RunesToChars([]rune(s))
	return match(algo.NormalizeRunes(lowerRunes(pattern)), &in)
}

// Highlight оборачивает подряд идущие совпавшие руны s (позиции pos) в mark.
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
