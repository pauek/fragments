
package main

import (
	"os"
	"io"
	"bytes"
	"fmt"
	"strings"
	"html/template"
)

type UpdateFunc func(id string) string
type Values map[string]interface{}

var routes map[string]UpdateFunc
var cache  map[string]*Fragment

func init() {
	routes = make(map[string]UpdateFunc)
	cache  = make(map[string]*Fragment)
}

type Fragment struct {
	kind     string
	id       string
	text     *template.Template
	fn       UpdateFunc
	children []*Fragment
}

func (frag *Fragment) update(fn UpdateFunc) {
	children := make([]string, 0)
	fmap := map[string]interface{} {
		"fragment": func(id string) string {
			i := len(children)
			children = append(children, id)
			return fmt.Sprintf("{{ index .children %d }}", i)
		},
	}
	tmpl, err := template.New("frag").Funcs(fmap).Parse(fn(frag.id))
	if err != nil {
		fmt.Println(err)
		panic(fmt.Sprintf("Template for '%s' has errors", frag.kind))
	}
	var text bytes.Buffer
	tmpl.Execute(&text, nil)
	frag.text, err = template.New("frag").Parse(text.String())
	if err != nil {
		panic("Internal template error")
	}
	frag.children = make([]*Fragment, 0)
	for i := range children {
		frag.children = append(frag.children, Get(children[i]))
	}
}

func (frag *Fragment) Render(w io.Writer) {
	children := make([]template.HTML, 0)
	for i := range frag.children {
		child := frag.children[i]
		var result bytes.Buffer
		child.Render(&result)
		children = append(children, template.HTML(result.String()))
	}
	frag.text.Execute(w, Values { "children": children })
}

func Get(id string) *Fragment {
	if frag, ok := cache[id]; ok {
		fmt.Println("FROM CACHE!")
		return frag
	}
	parts := strings.Split(id, ":")
	if len(parts) != 2 {
		panic(fmt.Sprintf("Malformed fragment ID: '%s'", id))
	}
	fn, ok := routes[parts[0]]
	if ! ok {
		panic(fmt.Sprintf("Fragment type '%s' not found", parts[0]))
	}
	frag := &Fragment{ kind: parts[0], id: parts[1], fn: fn }
	frag.update(fn)
	cache[id] = frag
	return frag
}

func Ref(id string) template.HTML {
	return template.HTML(fmt.Sprintf("{{ \"%s\" | fragment }}", id))
}

const T = `<div id="{{ .id }}" class="comments">
<p>{{ .comment }}</p>
by {{ .user }}
</div>
`

func comments(id string) string {
	cT, _ := template.New("comment").Parse(T)
	var b bytes.Buffer
	cT.Execute(&b, Values {
		"id": id,
		"comment": "Blah blah",
		"user": Ref("user:pauek"),
	})
	return b.String()
}

func user(id string) string {
	return fmt.Sprintf("<div class=\"user\">User: %s</div>", id)
}

func main() {
	routes["comments"] = comments
	routes["user"] = user

	var frag *Fragment
	frag = Get("comments:jarl")
	frag.Render(os.Stdout)
	frag = Get("comments:xxx")
	frag.Render(os.Stdout)

	for k, v := range cache {
		fmt.Println(k, v)
	}
}