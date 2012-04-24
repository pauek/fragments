package fragments

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"os"
	"strings"
	"time"
)

type Values map[string]interface{}

type UpdateFunc func(id string) (string, error)

type Fragment struct {
	kind     string
	id       string
	stamp    time.Time
	valid    bool
	text     *template.Template
	children []string
	err      error
}

type Type struct {
	updatefn UpdateFunc
}

var (
	types  = make(map[string]Type)
	cache  = make(map[string]*Fragment)
	depend = make(map[string]map[string]bool)
)

func (frag *Fragment) update(fn UpdateFunc) {
	// call update function
	res, err := fn(frag.id)
	if err != nil {
		frag.err = err
		return
	}
	frag.stamp = time.Now()

	// collect children + insert placeholders
	children := []string{}
	fmap := map[string]interface{}{
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
	var b bytes.Buffer
	tmpl.Execute(&b, nil)
	text := fmt.Sprintf(`<div fragment="%s">%s</div>`, frag.ID(), b.String())
	
	// compile final template
	frag.text, err = template.New("frag").Parse(text)
	if err != nil {
		frag.err = err
		return
	}
	frag.valid = true
	frag.children = children
}

func (frag *Fragment) Render(w io.Writer) {
	children := []template.HTML{}
	for i := range frag.children {
		child, err := Get(frag.children[i])
		if err != nil {
			frag.err = fmt.Errorf("Children '%s' fragment error: %s\n", frag.children[i], err)
		}
		var result bytes.Buffer
		child.Render(&result)
		children = append(children, template.HTML(result.String()))
	}
	frag.text.Execute(w, Values{"children": children})
}

func (frag *Fragment) ID() string {
	return fmt.Sprintf("%s:%s", frag.kind, frag.id)
}

type DiffItem struct{ id, html string }

func (d DiffItem) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{"id": d.id, "html": d.html /*, timestamp */})
}

func (frag *Fragment) Diff(since time.Time) []DiffItem {
	fmt.Println("Diff:", since)
	D := []DiffItem{}
	if frag.stamp.After(since) {
		stubs := make([]template.HTML, len(frag.children))
		for i := range frag.children {
			child, err := Get(frag.children[i])
			if err != nil {
				panic("Cannot get child!")
			}
			stubs[i] = child.Stub()
		}
		var b bytes.Buffer
		frag.text.Execute(&b, Values{"children": stubs})
		D = append(D, DiffItem{id: frag.ID(), html: b.String()})
	}
	// call recursively
	for i := range frag.children {
		child, err := Get(frag.children[i])
		if err != nil {
			panic("Cannot get child!")
		}
		d := child.Diff(since)
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

// Register a type of fragment. The ID of the type is a prefix of the
// fragment ID. For instance, a fragment 'user:pauek' has type 'user'.
//
func Add(typId string, fn UpdateFunc) {
	types[typId] = Type{updatefn: fn}
}

// Get a fragment from cache. If it exists and is valid, it is
// returned immediately.  Otherwise it is first updated.
//
func Get(id string) (F *Fragment, err error) {
	if frag, ok := cache[id]; ok {
		if frag.valid {
			return frag, nil
		}
		if frag.err != nil {
			return frag, frag.err
		}
	}
	parts := strings.Split(id, ":")
	if len(parts) != 2 {
		panic(fmt.Sprintf("Malformed fragment ID: '%s'", id))
	}
	typ, ok := types[parts[0]]
	if !ok {
		panic(fmt.Sprintf("Fragment type '%s' not found", parts[0]))
	}
	fn := typ.updatefn
	frag := &Fragment{kind: parts[0], id: parts[1], valid: false}
	frag.update(fn)
	cache[id] = frag
	if frag.err != nil {
		return frag, frag.err
	}
	return frag, nil
}

// Create a reference to another fragment to put in a template.
//
func Ref(id string) template.HTML {
	return template.HTML(fmt.Sprintf(`{{ "%s" | fragment }}`, id))
}

// invalidate all fragments depending on the object with 'id'
//
func Invalidate(objId string) {
	fmt.Fprintf(os.Stderr, "Invalidate(%s)\n", objId)
	if fragIds, ok := depend[objId]; ok {
		for fragId := range fragIds {
			fmt.Fprintf(os.Stderr, "checking '%s'\n", fragId)
			if frag, found := cache[fragId]; found {
				fmt.Fprintf(os.Stderr, "Invalidate '%s' -> '%s'\n", objId, fragId)
				frag.valid = false
			}
		}
	}
}

// Declare a dependency between an object with ID 'objId' and a
// fragment with id 'fragId'. When Invalidate is called with 'objId',
// all fragments with a Declared dependency will be updated.
//
func Depends(fragId, objId string) {
	if _, ok := depend[objId]; !ok {
		depend[objId] = make(map[string]bool)
	}
	depend[objId][fragId] = true
	fmt.Fprintf(os.Stderr, "Depends[%s] = %v\n", objId, depend[objId])
}

/*

func init() {
   http.Handle("/_fragment/", func ...)
   http.Handle("/js/fragments.js", func ...)
}

*/
