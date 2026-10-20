package search

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
)

func genDocs(n int) []Doc {
	r := rand.New(rand.NewSource(1))
	words := strings.Fields("login page refactor cache backend frontend bug fix api search index fuzzy board kanban sprint release deploy build test review docs plane state label user project теги кэш задача поиск настройка ошибка сервер клиент")
	docs := make([]Doc, n)
	for i := range docs {
		w := make([]string, 3+r.Intn(8))
		for j := range w {
			w[j] = words[r.Intn(len(words))]
		}
		docs[i] = Doc{ID: fmt.Sprintf("id-%d", i), Key: fmt.Sprintf("PROJ-%d", i+1), Title: strings.Join(w, " ")}
	}
	return docs
}

func BenchmarkNew5000(b *testing.B) {
	d := genDocs(5000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		New(d)
	}
}

func BenchmarkSearch5000(b *testing.B) {
	ix := New(genDocs(5000))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ix.Search("fix cache")
	}
}

func benchFuzzyLong(b *testing.B, n int) {
	const pat = "abcdefghijklmnopqrstuvwxyzabcdef"
	text := strings.Repeat("abcdefghijklmnopqrstuvwxyz ", n/27+1)[:n]
	if strings.Contains(text, pat) {
		b.Fatal("substring")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, ok := Fuzzy(pat, text); !ok {
			b.Fatal("no match")
		}
	}
}

func BenchmarkFuzzyLong3000(b *testing.B) { benchFuzzyLong(b, 3000) }
func BenchmarkFuzzyLong9000(b *testing.B) { benchFuzzyLong(b, 9000) }
