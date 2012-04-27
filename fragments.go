package fragments

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	tmpl "html/template"
	"os"
	"time"
)

var DebugInfo = true

func log(format string, args ...interface{}) {
	if DebugInfo {
		fmt.Fprintf(os.Stderr, format, args...)
	}
}

type Values map[string]interface{}

type UpdateFunc func(id string) (string, error)

type Ref struct {
	Kid, Id string
}

func (R Ref) String() string {
	return fmt.Sprintf("%s:%s", R.Kid, R.Id)
}

type Fragment struct {
	Ref
	stamp    time.Time
	valid    bool
	text     *tmpl.Template
	children []Ref
	err      error
}

type Kind struct {
	updatefn UpdateFunc
	cache map[string]*Fragment
}


var (
	kinds  = make(map[string]Kind)
	depend = make(map[string]map[Ref]bool)
)


func (frag *Fragment) update(fn UpdateFunc) {
	log("update(%s:%s)\n", frag.Kid, frag.Id)
	// call update function
	res, err := fn(frag.Id)
	if err != nil {
		log("err: %s\n", err)
		frag.err = err
		return
	}
	frag.stamp = time.Now()

	// collect children + insert placeholders
	children := []Ref{}
	fmap := map[string]interface{}{
		"fragment": func(kid, id string) string {
			i := len(children)
			children = append(children, Ref{kid, id})
			log("Child %s of %s: %s:%s\n", i, frag.ID(), kid, id)
			return fmt.Sprintf(`{{fragment "%s" "%s"}}`, kid, id)
		},
	}
	t, err := tmpl.New("frag").Funcs(fmap).Parse(res)
	if err != nil {
		fmt.Println(err)
		panic(fmt.Sprintf("Template for '%s' has errors", frag.Kid))
	}
	var b bytes.Buffer
	t.Execute(&b, nil)
	text := fmt.Sprintf(`<div fragment="%s">%s</div>`, frag.ID(), b.String())

	// compile final template
	frag.text, err = tmpl.New("frag").Parse(text)
	if err != nil {
		frag.err = err
		return
	}
	frag.valid = true
	frag.children = children
}

func (frag *Fragment) ID() string {
	return fmt.Sprintf("%s:%s", frag.Kid, frag.Id)
}

type DiffItem struct{ id, html string }

func (d DiffItem) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{"id": d.id, "html": d.html /*, timestamp */})
}

func (frag *Fragment) Diff(since time.Time) []DiffItem {
	if DebugInfo {
		fmt.Println("Diff:", since)
	}
	D := []DiffItem{}
	if frag.stamp.After(since) {
		stubs := make([]tmpl.HTML, len(frag.children))
		for i := range frag.children {
			ref := frag.children[i]
			child, err := get(ref.Kid, ref.Id)
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
		ref := frag.children[i]
		child, err := get(ref.Kid, ref.Id)
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

func (frag *Fragment) Stub() tmpl.HTML {
	div := fmt.Sprintf("<div fragment=\"%s:%s\"></div>", frag.Kid, frag.Id)
	return tmpl.HTML(div)
}

//////////////////////////////////////////////////////////////////////

// Register a type of fragment. The ID of the type is a prefix of the
// fragment ID. For instance, a fragment 'user:pauek' has type 'user'.
//
func Add(kid string, fn UpdateFunc) {
	kinds[kid] = Kind{
		updatefn: fn,
		cache: make(map[string]*Fragment),
	}
}

// Get a fragment from cache. If it exists and is valid, it is
// returned immediately.  Otherwise it is first updated.
//
func get(kid, id string) (F *Fragment, err error) {
	kind, ok := kinds[kid]
	if !ok {
		panic(fmt.Sprintf("Fragment type '%s' not found", kid))
	}
	if frag, ok := kind.cache[id]; ok {
		if frag.err != nil {
			return frag, frag.err
		}
		if frag.valid {
			return frag, nil
		}
	}
	fn := kind.updatefn
	frag := &Fragment{valid: false}
	frag.Kid = kid
	frag.Id = id
	frag.update(fn)
	kind.cache[id] = frag
	if frag.err != nil {
		return frag, frag.err
	}
	return frag, nil
}

// invalidate all fragments depending on the object with 'id'
//
func Invalidate(id string) {
	log("Invalidate(%s)\n", id)
	if fragrefs, ok := depend[id]; ok {
		for ref := range fragrefs {
			log("Checking '%s:%s'\n", ref.Kid, ref.Id)
			kind, found := kinds[ref.Kid]
			if !found {
				panic(fmt.Sprintf("Fragment type '%s' not found", ref.Kid))
			}
			if frag, found := kind.cache[ref.Id]; found {
				log("Invalidate '%s' -> '%s:%s'\n", id, ref.Kid, ref.Id)
				frag.valid = false
			}
		}
	}
}

// Declare a dependency between an object with ID 'objId' and a
// fragment with id 'fragId'. When Invalidate is called with 'objId',
// all fragments with a Declared dependency will be updated.
//
func Depends(ref Ref, objId string) {
	if _, ok := depend[objId]; !ok {
		depend[objId] = make(map[Ref]bool)
	}
	depend[objId][ref] = true
	log("Depends[%s] = %v\n", objId, depend[objId])
}

func Each(fn func(id string, f *Fragment)) {
	for kid, kind := range kinds {
		for id, f := range kind.cache {
			fn(kid+":"+id, f)
		}
	}
}

func Parse(t *tmpl.Template, text string) (*tmpl.Template, error) {
	var fmap = tmpl.FuncMap{
		"fragment": func () (tmpl.HTML, error) {
			return tmpl.HTML("Error: placeholder function"), nil
		},
	}
	tp, err := t.Funcs(fmap).Parse(text)
	if err != nil {
		return nil, err
	}
	return tp, nil
}

// the payload is passed to any fragment function updating
func Execute(w io.Writer, t *tmpl.Template, v Values, payload interface{}) error {
	var fmap = tmpl.FuncMap{}
	fmap["fragment"] = func(ids ...interface{}) (tmpl.HTML, error) {
		log("==> fragment(%v)\n", ids)
		var kid, id string
		switch len(ids) {
		case 2: 
			kid, id = ids[0].(string), ids[1].(string)
		case 1:
			kid, id = ids[0].(string), ""
		default:
			return "", fmt.Errorf("Wrong number of arguments for 'fragment'")
		}			
		frag, err := get(kid, id)
		if err != nil {
			return "", err
		}
		var b bytes.Buffer
		frag.text.Funcs(fmap).Execute(&b, nil)
		return tmpl.HTML(b.String()), nil
	}
	return t.Funcs(fmap).Execute(w, v)
}

func Render(t *tmpl.Template, v Values, payload interface{}) (string, error) {
	var b bytes.Buffer
	err := Execute(&b, t, v, payload)
	if err != nil {
		return "", err
	}
	return b.String(), nil
}


/*

func init() {
   http.Handle("/_fragment/", func ...)
   http.Handle("/js/fragments.js", func ...)
}

*/
