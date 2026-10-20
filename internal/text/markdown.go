package text

import (
	"bytes"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

// md — CommonMark + GFM; сырой HTML (без WithUnsafe) выводится экранированным.
var md = goldmark.New(
	goldmark.WithExtensions(extension.GFM),
	goldmark.WithRendererOptions(renderer.WithNodeRenderers(util.Prioritized(rawHTMLEscaper{}, 100))),
)

// rawHTMLEscaper выводит сырой HTML исходника как экранированный текст.
type rawHTMLEscaper struct{}

// RegisterFuncs реализует renderer.NodeRenderer.
func (rawHTMLEscaper) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindRawHTML, rawInline)
	reg.Register(ast.KindHTMLBlock, rawBlock)
}

func rawInline(w util.BufWriter, src []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		raw := n.(*ast.RawHTML)
		for i := 0; i < raw.Segments.Len(); i++ {
			seg := raw.Segments.At(i)
			_, _ = w.Write(util.EscapeHTML(seg.Value(src)))
		}
	}
	return ast.WalkSkipChildren, nil
}

func rawBlock(w util.BufWriter, src []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	b := n.(*ast.HTMLBlock)
	_, _ = w.WriteString("<p>")
	for i := 0; i < b.Lines().Len(); i++ {
		seg := b.Lines().At(i)
		_, _ = w.Write(util.EscapeHTML(seg.Value(src)))
	}
	if b.HasClosure() {
		_, _ = w.Write(util.EscapeHTML(b.ClosureLine.Value(src)))
	}
	_, _ = w.WriteString("</p>\n")
	return ast.WalkSkipChildren, nil
}

// MarkdownToHTML переводит Markdown в HTML.
func MarkdownToHTML(src string) string {
	var buf bytes.Buffer
	if err := md.Convert([]byte(src), &buf); err != nil {
		return ""
	}
	return buf.String()
}
