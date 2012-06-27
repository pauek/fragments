package fragments

import (
	"bytes"
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
	"a{b b}c", "ab bb bc",
}

func TestRender(t *testing.T) {
	for i := 0; i < len(templateRender); i += 2 {
		source := templateRender[i]
		target := templateRender[i+1]
		tmpl, _ := Parse(source, "{", "}")
		var b bytes.Buffer
		tmpl.Exec(&b, func (id string) {
			b.Write([]byte(id))
			b.Write([]byte(id))
		})
		if b.String() != target {
			t.Errorf("'%s' != '%s'", b.String(), target)
		}
	}
}