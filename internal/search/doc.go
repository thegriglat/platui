// Package search implements instant fuzzy search over issue titles
// (contract C9 v4). A leaf package: the caller assembles the documents.
// Matching uses the fzf algorithm (github.com/junegunn/fzf/src/algo); we do not write our own.
package search

// MaxQuery is the maximum query length in runes; Search and ParseKey truncate the query.
const MaxQuery = 32

// Doc is a document to index.
type Doc struct {
	ID    string // issue ID (UUID)
	Key   string // "PROJ-42"
	Title string
}

// Hit is a matched document.
type Hit struct {
	ID    string
	Score int
	Pos   []int // positions of matched runes in Title, ascending, no duplicates
}
