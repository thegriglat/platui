package search

import (
	"sort"
	"strconv"
	"strings"

	"github.com/junegunn/fzf/src/algo"
	"github.com/junegunn/fzf/src/util"
)

type entry struct {
	id, key string
	title   util.Chars
}

// Index — неизменяемый после New индекс; безопасен для чтения из горутин.
type Index struct {
	docs []entry
}

// New строит индекс: заголовки заранее приводятся к рунам.
func New(docs []Doc) *Index {
	ix := &Index{docs: make([]entry, len(docs))}
	for i, d := range docs {
		ix.docs[i] = entry{id: d.ID, key: d.Key, title: util.RunesToChars([]rune(d.Title))}
	}
	return ix
}

// Len — число документов.
func (ix *Index) Len() int { return len(ix.docs) }

func clip(q string) string {
	if r := []rune(q); len(r) > MaxQuery {
		q = string(r[:MaxQuery])
	}
	return q
}

// Search ищет документы по запросу (И по словам); пустой запрос и запрос-ключ дают nil.
func (ix *Index) Search(q string) []Hit {
	q = clip(q)
	if _, _, ok := ParseKey(q); ok {
		return nil
	}
	var words [][]rune
	for _, w := range strings.Fields(q) {
		words = append(words, algo.NormalizeRunes(lowerRunes(w)))
	}
	if len(words) == 0 {
		return nil
	}
	var hits []Hit
	for i := range ix.docs {
		e := &ix.docs[i]
		h := Hit{ID: e.id}
		set := map[int]bool{}
		ok := true
		for _, w := range words {
			sc, pos, m := match(w, &e.title)
			if !m {
				ok = false
				break
			}
			h.Score += sc
			for _, p := range pos {
				set[p] = true
			}
		}
		if !ok {
			continue
		}
		for p := range set {
			h.Pos = append(h.Pos, p)
		}
		sort.Ints(h.Pos)
		hits = append(hits, h)
	}
	sort.SliceStable(hits, func(a, b int) bool { return hits[a].Score > hits[b].Score })
	return hits
}

// ParseKey разбирает «PROJ-42», «42», «#42»; иначе ok=false.
func ParseKey(q string) (identifier string, seq int, ok bool) {
	q = strings.TrimSpace(clip(strings.TrimSpace(q)))
	q = strings.TrimPrefix(q, "#")
	if i := strings.LastIndexByte(q, '-'); i >= 0 {
		identifier, q = q[:i], q[i+1:]
		if !validIdent(identifier) {
			return "", 0, false
		}
		identifier = strings.ToUpper(identifier)
	}
	n, err := strconv.Atoi(q)
	if err != nil || n < 0 || q == "" || q[0] == '+' || q[0] == '-' {
		return "", 0, false
	}
	return identifier, n, true
}

func validIdent(s string) bool {
	for i, c := range s {
		letter := c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
		if !letter && (i == 0 || (c < '0' || c > '9') && c != '_') {
			return false
		}
	}
	return s != ""
}

// Lookup — точный поиск по Key; identifier "" — первый документ с Key, оканчивающимся на "-<seq>".
func (ix *Index) Lookup(identifier string, seq int) (id string, ok bool) {
	n := strconv.Itoa(seq)
	for _, e := range ix.docs {
		if identifier == "" {
			if strings.HasSuffix(e.key, "-"+n) {
				return e.id, true
			}
		} else if strings.EqualFold(e.key, identifier+"-"+n) {
			return e.id, true
		}
	}
	return "", false
}
