package fragments

import (
	"bytes"
	"fmt"
	"testing"
)

var templatesOk = []string{
	"a",
	"{b}",
	"{bb}",
	"{bb}{bb}",
	"{bbbbbbb}",
	"a{b}",
	"a{bbbbbbbbb}",
	"a{bbb}c",
	"aaaaa{bb}c",
	"a{b}ccccc",
	"a{b}c{d}",
	"a{b}{c}{d}{e}",
}

func TestParse1(t *testing.T) {
	for _, tmpl := range templatesOk {
		if _, e := Parse(tmpl, "{", "}"); e != nil {
			t.Errorf("Parse('%s') shouldn't give error '%s'", tmpl, e)
		}
	}	
}

var templatesError = []string{
	"a[",
	"[a",
	"[a[",
	"]a]",
	"[a][",
	"aaaaa]",
	"[a]b[",
	"[aaaaaa]bbbbbbb[",
	"aaaaaaaaaaaaaaaaaaaaaaaaa]b[",
	"aaaa[b][c][d][e][",
}

func TestParse2(t *testing.T) {
	for _, tmpl := range templatesError {
		if _, e := Parse(tmpl, "[", "]"); e == nil {
			t.Errorf("Parse should give error")
		}
	}	
}

var templateSize = []string{
	"<a>b<c>",
	"<>a<>b<>c<>",
	"<a><b><c>",
	"<aaaa>b<>c",
	"<>a<>b<c>",
}

func TestParse3(t *testing.T) {
	for _, tmpl := range templateSize {
		p, _ := Parse(tmpl, "<", ">")
		if len(p) != 3 {
			t.Errorf("len('%#v') should be 3", p)
		}
	}	
}

var templateRender = []string{
	"xxx{y}xxx", "xxxyyxxx",
	"a{b b}c", "a[b b][b b]c",
}

func TestRender(t *testing.T) {
	c := NewCache()
	c.Register("y", func(C *Cache, args []string) Fragment {
		return Text("yy")
	})
	c.Register("b", func(C *Cache, args []string) Fragment {
		return Text(fmt.Sprintf("%v%v", args, args))
	})
	for i := 0; i < len(templateRender); i += 2 {
		source := templateRender[i]
		target := templateRender[i+1]
		tmpl, _ := Parse(source, "{", "}")
		var b bytes.Buffer
		tmpl.Render(&b, c, Recursive)
		if b.String() != target {
			t.Errorf("'%s' != '%s'", b.String(), target)
		}
	}
}

var templateCache = []string{
	"mult 1",     "1",
	"mult 6 7",   "6 * 7",
	"mult 5 6 7", "5 * 6 * 7",
}

func TestCache(t *testing.T) {
	C := NewCache()
	C.Register("mult", func(C *Cache, args []string) Fragment {
		s := fmt.Sprintf("%s", args[1])
		if len(args) > 2 {
			s += " * {mult"
			for _, a := range args[2:] {
				s += fmt.Sprintf(" %s", a)
			}
			s += "}"
			tmpl, _ := Parse(s, "{", "}")
			return tmpl
		}
		return Text(s)
	})
	for i := 0; i < len(templateCache); i += 2 {
		src, tgt := templateCache[i], templateCache[i+1]
		var b bytes.Buffer
		C.Render(&b, src)
		if b.String() != tgt {
			t.Errorf("'%s' != '%s'", b.String(), tgt)
		}
	}
}