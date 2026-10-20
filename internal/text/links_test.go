package text

import (
	"reflect"
	"testing"
)

func TestContractTextLinks(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"<p>no links</p>", nil},
		{`<a href="http://a.com/x">t</a>`, []string{"http://a.com/x"}},
		{`<p>see https://b.org/p?q=1 now.</p>`, []string{"https://b.org/p?q=1"}},
		{`<a href="https://a.com">https://a.com</a> and https://a.com`, []string{"https://a.com"}},
		{`<a href="mailto:x@y.z">m</a><a href="javascript:alert(1)">j</a><a href="/rel">r</a>`, nil},
		{`<p>first https://one.io</p><a href="http://two.io">2</a> https://three.io`, []string{"https://one.io", "http://two.io", "https://three.io"}},
		{`<p>(see https://w.org/a_(b))</p>`, []string{"https://w.org/a_(b)"}},
		{`<a href="http://a.com?x=1&amp;y=2">t</a>`, []string{"http://a.com?x=1&y=2"}},
	}
	for _, c := range cases {
		if got := ExtractLinks(c.in); !reflect.DeepEqual(got, c.want) {
			t.Errorf("ExtractLinks(%q) = %q; want %q", c.in, got, c.want)
		}
	}
}
