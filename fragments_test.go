
package fragments

import (
	"bytes"
	"testing"
)

var t1 = []string{
	"abc", "abc",
	"{a}", "?",
	"{x}", "123",
	"a{x}b", "a123b",
	"a{x}{y}b", "a123456b",
	"a{x}c{y}e", "a123c456e",
	"a{}c{}e", "a?c?e",
	"a{x}c{}e", "a123c?e",
	"a{}c{y}e", "a?c456e",
	"{x}{y}", "123456",
	"{x}aaaaaaaaaaaa", "123aaaaaaaaaaaa", 
	"aaaaaaaaaaaa{y}", "aaaaaaaaaaaa456", 
}

func TestTemplate1(t *testing.T) {
	for i := 0; i < len(t1); i += 2 {
		src, tgt := t1[i], t1[i+1]
		tmpl, _ := Parser{"{", "}"}.Parse(src)
		var b bytes.Buffer
		tmpl.Exec(&b, func(action string) {
			switch action {
			case "x": b.Write([]byte("123"))
			case "y": b.Write([]byte("456"))
			default: b.Write([]byte("?"))
			}
		})
		if tgt != b.String() {
			t.Errorf(`"%s" != "%s"`, tgt, b.String())
		}
	}
}

