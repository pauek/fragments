package fragments

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"strings"
	"time"
)

type Fragment string
type Values map[string]interface{}

type Traverser struct {
	Text  func(text string)
	Child func(typ, id string) error
}

func (F *Fragment) Traverse(T Traverser) error {
	s := string(*F)
	for {
		i := strings.Index(s, "{{")
		if i == -1 {
			break
		}
		if T.Text != nil {
			T.Text(s[:i])
		}
		j := strings.Index(s, "}}")
		if j == -1 {
			return fmt.Errorf("Traverse: unmatched '{{'")
		}
		typ, id := SplitID(s[i+2 : j])
		if T.Child != nil {
			if err := T.Child(typ, id); err != nil {
				return fmt.Errorf("Traverse: %s", err)
			}
		}
		s = s[j+2:]
	}
	if T.Text != nil {
		T.Text(s)
	}
	return nil
}

func (F *Fragment) Clean() *Fragment {
	r := strings.NewReplacer("\n", "", "  ", " ", "   ", " ", "\t", " ")
	*F = Fragment(r.Replace(string(*F)))
	return F
}

func (F *Fragment) Stubs() string {
	var b bytes.Buffer
	F.Traverse(Traverser{
		Text: func(text string) {
			fmt.Fprintf(&b, text)
		},
		Child: func(typ, id string) error {
			fmt.Fprintf(&b, `<div fragment="%s:%s"></div>`, typ, id)
			return nil
		},
	})
	return b.String()
}

// Registry

type GenFunc func(id string, data interface{}) (*Fragment, []string, error)

type Generator struct {
	Func  GenFunc
	Layer string
}

var generators = make(map[string]Generator)

func Add(typ string, gen Generator) {
	generators[typ] = gen
}

// Fragments

type CacheItem struct {
	frag      *Fragment
	valid     bool
	timestamp time.Time
	depends   []string
}

type Cache struct {
	data  interface{}
	cache map[string]*CacheItem
}

func NewCache(data interface{}) *Cache {
	return &Cache{
		data:  data,
		cache: make(map[string]*CacheItem),
	}
}

func (C *Cache) Each(fn func(id string, f *Fragment)) {
	for id, item := range C.cache {
		fn(id, item.frag)
	}
}

func (C *Cache) Get(typ, id string) (*Fragment, error) {
	item, err := C.get(typ, id)
	if err != nil {
		return nil, err
	}
	return item.frag, nil
}

func (C *Cache) get(typ, id string) (*CacheItem, error) {
	fid := typ + ":" + id
	item, ok := C.cache[fid]
	if !ok || !item.valid {
		gen, ok2 := generators[typ]
		if !ok2 {
			return nil, fmt.Errorf("Type '%s' not found", typ)
		}
		// TODO: handle deps
		frag, deps, err := gen.Func(id, C.data)
		if err != nil {
			msg := "Generation error for '%s:%s': %s\n"
			return nil, fmt.Errorf(msg, typ, id, err)
		}
		item = &CacheItem{
			frag:      frag,
			valid:     true,
			timestamp: time.Now(),
			depends:   deps,
		}
		for _, oid := range deps {
			depends(item, oid)
		}
		C.cache[fid] = item
	}
	return item, nil
}

type getFn func(typ, id string) (*Fragment, error)

func get(C *Cache) getFn {
	return func(typ, id string) (*Fragment, error) {
		return C.Get(typ, id)
	}
}

type Layers map[string]*Cache

func getLayers(C *Cache, layers Layers) getFn {
	return func(typ, id string) (*Fragment, error) {
		gen, ok := generators[typ]
		if !ok {
			return nil, fmt.Errorf("Generator for '%s' not found", typ)
		}
		layer := C
		if gen.Layer != "" {
			if layer, ok = layers[gen.Layer]; !ok {
				return nil, fmt.Errorf("Layer '%s' not found", gen.Layer)
			}
		}
		item, err := layer.get(typ, id)
		if err != nil {
			return nil, err
		}
		return item.frag, nil
	}
}

func SplitID(fid string) (typ, id string) {
	k := strings.Index(fid, ":")
	if k == -1 {
		typ, id = fid, ""
	} else {
		typ, id = fid[:k], fid[k+1:]
	}
	return
}

func (C *Cache) exec(f *Fragment, w io.Writer, fn getFn) error {
	return f.Traverse(Traverser{
		Text: func(text string) {
			fmt.Fprint(w, text)
		},
		Child: func(typ, id string) error {
			f, err := fn(typ, id)
			if err != nil {
				return fmt.Errorf("exec: Cannot Get '%s:%s': %s", typ, id, err)
			}
			fmt.Fprintf(w, `<div fragment="%s:%s">`, typ, id)
			if err := C.exec(f, w, fn); err != nil {
				return err
			}
			fmt.Fprintf(w, `</div>`)
			return nil
		},
	})
}

type DiffItem struct {
	Id, Html string
	Stamp time.Time
}

func (C *Cache) getlist(f *Fragment, fn getFn) (list []*DiffItem, err error) {
	f.Traverse(Traverser{
		Child: func(typ, id string) error {
			f, err := fn(typ, id)
			if err != nil {
				return fmt.Errorf("getlist: Cannot Get '%s:%s': %s", typ, id, err)
			}
			list = append(list, &DiffItem{
				Id:   typ + ":" + id,
				Html: f.Clean().Stubs(),
			})
			L, err := C.getlist(f, fn)
			if err != nil {
				return err
			}
			list = append(list, L...)
			return nil
		},
	})
	return
}

func (C *Cache) Execute(w io.Writer, f *Fragment) error {
	return C.exec(f, w, get(C))
}

func (C *Cache) ExecuteTemplate(w io.Writer, tmpl *template.Template, v Values) error {
	frag, err := PreRender(tmpl, v)
	if err != nil {
		return err
	}
	return C.Execute(w, frag)
}

func (C *Cache) ExecuteLayers(f *Fragment, w io.Writer, layers Layers) error {
	return C.exec(f, w, getLayers(C, layers))
}

func (C *Cache) GetList(f *Fragment) (list []*DiffItem, err error) {
	return C.getlist(f, get(C))
}

func (C *Cache) GetListLayers(f *Fragment, layers Layers) (list []*DiffItem, err error) {
	return C.getlist(f, getLayers(C, layers))
}

func (C *Cache) Render(f *Fragment) (string, error) {
	var b bytes.Buffer
	if err := C.Execute(&b, f); err != nil {
		return "", err
	}
	return b.String(), nil
}

func (C *Cache) RenderLayers(f *Fragment, lyrs Layers) (string, error) {
	var b bytes.Buffer
	err := C.ExecuteLayers(f, &b, lyrs)
	if err != nil {
		return "", err
	}
	return b.String(), nil
}

func Parse(text string) (*template.Template, error) {
	fmap := map[string]interface{}{
		"fragment": func(ids ...interface{}) string {
			sids := make([]string, len(ids))
			for i, id := range ids {
				sids[i] = fmt.Sprintf("%v", id)
			}
			return fmt.Sprintf("{{%s}}", strings.Join(sids, ":"))
		},
	}
	t, err := template.New("").Funcs(fmap).Parse(text)
	if err != nil {
		return nil, fmt.Errorf("Parse error: %s", err)
	}
	return t, nil
}

// Execute a template, embedding all values, except
// references to fragments
func PreRender(t *template.Template, v Values) (*Fragment, error) {
	var b bytes.Buffer
	err := t.Execute(&b, v)
	if err != nil {
		return nil, fmt.Errorf("Exec error: %s", err)
	}
	f := Fragment(b.String())
	return &f, nil
}

// Invalidation

var deps = make(map[string][]*CacheItem)

func depends(item *CacheItem, id string) {
	deps[id] = append(deps[id], item)
}

func Invalidate(id string) {
	if itemlist, ok := deps[id]; ok {
		for _, item := range itemlist {
			item.valid = false
		}
	}
}

/* 
 TODO:

 - Diff: Given a fragment, return a diff list of its children.

 - Remove framents (by date?)

 - If a Cache is GCd, its items might be referenced in 'deps', 
   so there is a "memory leak" here.

 - Set a limit in bytes for the cache (?)

*/
