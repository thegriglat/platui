package text

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestContractTextParseQuick(t *testing.T) {
	ok := []struct {
		in   string
		want Quick
	}{
		{"fix login !high @me @anna #bug #ui", Quick{"fix login", "high", []string{"me", "anna"}, []string{"bug", "ui"}}},
		{"  a   b  ", Quick{Title: "a b"}},
		{"x !U", Quick{Title: "x", Priority: "urgent"}},
		{"x !h", Quick{Title: "x", Priority: "high"}},
		{"x !Medium", Quick{Title: "x", Priority: "medium"}},
		{"x !l", Quick{Title: "x", Priority: "low"}},
		{"x !NONE", Quick{Title: "x", Priority: "none"}},
		{"x !n", Quick{Title: "x", Priority: "none"}},
		{"x @a @a #t #t", Quick{Title: "x", Assignees: []string{"a"}, Labels: []string{"t"}}},
		{"x @ # !", Quick{Title: "x @ # !"}},
		{`\!high \@me \#tag`, Quick{Title: "!high @me #tag"}},
		{"a @me b #l c", Quick{Title: "a b c", Assignees: []string{"me"}, Labels: []string{"l"}}},
		{"a\tb\nc", Quick{Title: "a b c"}},
	}
	for _, c := range ok {
		got, err := ParseQuick(c.in)
		if err != nil || !reflect.DeepEqual(got, c.want) {
			t.Errorf("ParseQuick(%q) = %+v, %v; want %+v", c.in, got, err, c.want)
		}
	}
	for _, in := range []string{"", "   ", "@me #bug !high"} {
		if _, err := ParseQuick(in); !errors.Is(err, ErrEmptyTitle) {
			t.Errorf("ParseQuick(%q) err = %v, want ErrEmptyTitle", in, err)
		}
	}
	for in, tok := range map[string]string{"x !bogus": "!bogus", "x !h !l": "!l", "x !high !high": "!high"} {
		if _, err := ParseQuick(in); err == nil || !strings.Contains(err.Error(), tok) {
			t.Errorf("ParseQuick(%q) err = %v, want mention %q", in, err, tok)
		}
	}
}

func TestContractTextMarkdownToHTML(t *testing.T) {
	cases := []struct{ in, want string }{
		{"# Title", "<h1>Title</h1>"},
		{"- a\n- b", "<li>a</li>"},
		{"[x](http://e.com)", `<a href="http://e.com">x</a>`},
		{"`c`", "<code>c</code>"},
		{"```go\nx := 1\n```", `<pre><code class="language-go">x := 1`},
		{"| a | b |\n|---|---|\n| 1 | 2 |", "<table>"},
		{"~~del~~", "<del>del</del>"},
		{"- [x] done", `type="checkbox"`},
		{"see http://example.com now", `<a href="http://example.com">`},
		{"<script>alert(1)</script>", "&lt;script&gt;"},
	}
	for _, c := range cases {
		if got := MarkdownToHTML(c.in); !strings.Contains(got, c.want) {
			t.Errorf("MarkdownToHTML(%q) = %q, want contains %q", c.in, got, c.want)
		}
	}
	if got := MarkdownToHTML("<script>alert(1)</script>"); strings.Contains(got, "<script>") {
		t.Errorf("raw script not escaped: %q", got)
	}
}

func TestContractTextHTMLToMarkdown(t *testing.T) {
	cases := []struct{ in, want string }{
		{"<p>a <b>b</b> <i>c</i> <s>d</s></p>", "a **b** *c* ~~d~~"},
		{"<h2>Head</h2><p>t</p>", "## Head\n\nt"},
		{"<p>a<br>b</p>", "a\\\nb"},
		{"<ul><li>a<ul><li>b</li></ul></li><li>c</li></ul>", "- a\n  - b\n- c"},
		{"<ol><li>a</li><li>b</li></ol>", "1. a\n2. b"},
		{"<blockquote><p>q</p></blockquote>", "> q"},
		{"<p>a</p><hr><p>b</p>", "a\n\n---\n\nb"},
		{`<pre><code class="language-go">x &lt; 1` + "\n</code></pre>", "```go\nx < 1\n```"},
		{`<p><a href="http://e.com">x</a> <code>c</code></p>`, "[x](http://e.com) `c`"},
		{"<p>&lt;tag&gt; &amp; <span>plain</span> <u>u</u></p>", `\<tag\> \& plain u`},
		{"<div><p>a</p></div>", "a"},
		{"", ""},
		{"<p>unclosed <b>bold", "unclosed **bold**"},
		{"<!-- c --><p>x</p>", "x"},
		{"<p>1 < 2</p>", `1 \< 2`},
	}
	for _, c := range cases {
		if got := HTMLToMarkdown(c.in); got != c.want {
			t.Errorf("HTMLToMarkdown(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestContractTextRoundTrip(t *testing.T) {
	samples := []string{
		"# H1\n\n## H2\n\ntext **bold** *em* ~~del~~ `code`",
		"- a\n- b\n  - c\n  - d\n- e",
		"1. one\n2. two\n3. three",
		"- [x] done\n- [ ] todo",
		"> quote\n> more\n\npara",
		"[link](http://e.com \"t\") and http://auto.example.com/x_y",
		"```go\nfunc main() {}\n```",
		"| a | b |\n|:--|--:|\n| 1 | 2 |\n| x \\| y | 3 |",
		"a\\\nb\n\n---\n\nc",
		"snake_case * star \\# not heading <b>raw</b> &amp; \\- dash",
		"- loose a\n\n- loose b\n\n  para",
		"``` `tick` ```\n\n`` a`b ``",
		"![img](http://e.com/i.png)",
		"10\\. not list",
	}
	for _, s := range samples {
		h := MarkdownToHTML(s)
		back := HTMLToMarkdown(h)
		if h2 := MarkdownToHTML(back); h2 != h {
			t.Errorf("round trip %q:\n html  %q\n back  %q\n html2 %q", s, h, back, h2)
		}
	}
}

func TestContractTextEditSession(t *testing.T) {
	s, err := NewEditSession("old text", "hint one\nhint two")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	b, _ := os.ReadFile(s.Path)
	if !strings.HasPrefix(string(b), "old text") || !strings.Contains(string(b), "# hint one\n# hint two") {
		t.Fatalf("file = %q", b)
	}
	if got, err := s.Result(); err != nil || got != "old text" {
		t.Fatalf("untouched Result = %q, %v", got, err)
	}
	if out, err := s.Command(`printf "# T\nnew" >`).CombinedOutput(); err != nil {
		t.Fatalf("editor: %v %s", err, out)
	}
	if got, err := s.Result(); err != nil || got != "# T\nnew" {
		t.Fatalf("Result = %q, %v", got, err)
	}
	os.WriteFile(s.Path, []byte("body\n\n# hint one\n# hint two\n"), 0o600)
	if got, _ := s.Result(); got != "body" {
		t.Fatalf("hints not stripped: %q", got)
	}
	os.WriteFile(s.Path, []byte("\n# hint one\n  \n"), 0o600)
	if _, err := s.Result(); !errors.Is(err, ErrEmptyEdit) {
		t.Fatalf("err = %v, want ErrEmptyEdit", err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(s.Path); !os.IsNotExist(err) {
		t.Fatal("file not removed")
	}
	if err := s.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
	if _, err := s.Result(); err == nil {
		t.Fatal("Result on removed file should fail")
	}
}

func TestContractTextEditNoHint(t *testing.T) {
	s, err := NewEditSession("x\n", "")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if got, err := s.Result(); err != nil || got != "x" {
		t.Fatalf("Result = %q, %v", got, err)
	}
}

func TestContractTextOSC52(t *testing.T) {
	if got, want := OSC52("hi", false), "\x1b]52;c;aGk=\x07"; got != want {
		t.Errorf("got %q want %q", got, want)
	}
	if got, want := OSC52("hi", true), "\x1bPtmux;\x1b\x1b]52;c;aGk=\x07\x1b\\"; got != want {
		t.Errorf("got %q want %q", got, want)
	}
}
