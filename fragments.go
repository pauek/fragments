
package fragments

import (
	"io"
	"fmt"
	"time"
	"bytes"
	"strings"
	"html/template"
	"encoding/json"
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
	stamp    time.Time
	valid    bool
	text     *template.Template
	fn       UpdateFunc
	children []*Fragment
}

func (frag *Fragment) update(fn UpdateFunc) {
	// call update function
	res := fn(frag.id) 
	frag.stamp = time.Now()

	// collect children + insert placeholders
	children := []string{}
	fmap := map[string]interface{} {
		"fragment": func(id string) string {
			i := len(children)
			children = append(children, id)
			return fmt.Sprintf("{{ index .children %d }}", i)
		},
	}
	tmpl, err := template.New("frag").Funcs(fmap).Parse(res)
	if err != nil {
		fmt.Println(err)
		panic(fmt.Sprintf("Template for '%s' has errors", frag.kind))
	}
	var text bytes.Buffer
	tmpl.Execute(&text, nil)
	
	// compile final template
	frag.text, err = template.New("frag").Parse(text.String())
	if err != nil {
		panic("Internal template error")
	}
	frag.valid = true

	// Get fragments recursively
	frag.children = make([]*Fragment, len(children))
	for i := range children {
		frag.children[i] = Get(children[i])
	}
}

func (frag *Fragment) Render(w io.Writer) {
	children := []template.HTML{}
	for i := range frag.children {
		child := frag.children[i]
		var result bytes.Buffer
		child.Render(&result)
		children = append(children, template.HTML(result.String()))
	}
	frag.text.Execute(w, Values { "children": children })
}

func (frag *Fragment) ID() string {
	return fmt.Sprintf("%s:%s", frag.kind, frag.id)
}

type DiffItem struct { id, html string }

func (d DiffItem) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string { "id": d.id, "html": d.html })
}

func (frag *Fragment) Diff(since time.Time) []DiffItem {
	fmt.Println("Diff:", since)
	D := []DiffItem{}
	if (frag.stamp.After(since)) {
		stubs := make([]template.HTML, len(frag.children))
		for i := range frag.children {
			stubs[i] = frag.children[i].Stub()
		}
		var b bytes.Buffer
		frag.text.Execute(&b, Values { "children": stubs })
		D = append(D, DiffItem{ id: frag.ID(), html: b.String() })
	}
	// call recursively
	for i := range frag.children {
		d := frag.children[i].Diff(since)
		for j := range d {
			D = append(D, d[j])
		}
	}
	return D
}

func (frag *Fragment) Stub() template.HTML {
	div := fmt.Sprintf("<div fragment=\"%s:%s\"></div>", frag.kind, frag.id)
   return template.HTML(div)
}

//////////////////////////////////////////////////////////////////////

func Get(id string) *Fragment {
	if frag, ok := cache[id]; ok {
		if frag.valid {
			return frag
		}
	}
	parts := strings.Split(id, ":")
	if len(parts) != 2 {
		panic(fmt.Sprintf("Malformed fragment ID: '%s'", id))
	}
	fn, ok := routes[parts[0]]
	if ! ok {
		panic(fmt.Sprintf("Fragment type '%s' not found", parts[0]))
	}
	frag := &Fragment{ kind: parts[0], id: parts[1], fn: fn, valid: false }
	frag.update(fn)
	cache[id] = frag
	return frag
}

func Ref(id string) template.HTML {
	return template.HTML(fmt.Sprintf(`{{ "%s" | fragment }}`, id))
}

func Invalidate(id string) bool {
	if frag, ok := cache[id]; ok {
		frag.valid = false
		return true
	}
	return false
}

func Add(id string, fn UpdateFunc) {
	routes[id] = fn
}
