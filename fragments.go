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

func (C *Cache) Each(fn func (id string, f *Fragment)) {
	for id, item := range C.cache {
		fn(id, item.frag)
	}
}

func (C *Cache) Get(typ, id string) (*Fragment, error) {
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
			depends: deps,
		}
		for _, oid := range deps {
			depends(item, oid)
		}
		C.cache[fid] = item
	}
	return item.frag, nil
}

type getFn func(typ, id string) (*Fragment, error)

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
	s := string(*f)
	for {
		i := strings.Index(s, "{{")
		if i == -1 {
			break
		}
		w.Write([]byte(s[:i]))
		j := strings.Index(s, "}}")
		if j == -1 {
			return fmt.Errorf("Execute: unmatched '{{'")
		}
		typ, id := SplitID(s[i+2 : j])
		f, err := fn(typ, id)
		if err != nil {
			return fmt.Errorf("Execute: Cannot Get '%s:%s': %s", typ, id, err)
		}
		if err := C.exec(f, w, fn); err != nil {
			return err
		}
		s = s[j+2:]
	}
	w.Write([]byte(s))
	return nil
}

func (C *Cache) Execute(w io.Writer, f *Fragment) error {
	return C.exec(f, w, func (typ, id string) (*Fragment, error) {
		return C.Get(typ, id)
	})
}

func (C *Cache) ExecuteTemplate(w io.Writer, tmpl *template.Template, v Values) error {
	frag, err := PreRender(tmpl, v)
	if err != nil {
		return err
	}
	return C.Execute(w, frag)
}


type Layers map[string]*Cache

func (C *Cache) ExecuteLayers(f *Fragment, w io.Writer, layers Layers) error {
	return C.exec(f, w, func (typ, id string) (*Fragment, error) {
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
		return layer.Get(typ, id)
	})
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
	err := C.ExecuteLayers(f, &b, lyrs); 
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
			item.valid = false;
		}
	}
}

/* 
 TODO:

 - Diff: Given a fragment, return a diff list of its children.

 - Remove framents (by date?)

 - If a Cache is GCd, its items might be referenced in 'deps', 
   so there is a memory leak here.

 - Set a limit in bytes for the cache (?)

*/