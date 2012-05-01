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

func (C *Cache) Get(typ, id string) (*Fragment, error) {
	fid := typ + ":" + id
	item, ok := C.cache[fid]
	if !ok || !item.valid {
		gen, ok2 := generators[typ]
		if !ok2 {
			return nil, fmt.Errorf("Type '%s' not found", typ)
		}
		// TODO: handle deps
		frag, _, err := gen.Func(id, C.data)
		if err != nil {
			msg := "Generation error for '%s:%s': %s\n"
			return nil, fmt.Errorf(msg, typ, id, err)
		}
		item = &CacheItem{
			frag:      frag,
			valid:     true,
			timestamp: time.Now(),
		}
		C.cache[fid] = item
	}
	return item.frag, nil
}

func (C *Cache) Execute(f *Fragment, w io.Writer) error {
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
		id := s[i+2 : j]
		parts := strings.Split(id, ":")
		if len(parts) != 2 {
			return fmt.Errorf("Execute: fid is not '<typ>:<id>'")
		}
		typ, id := parts[0], parts[1]
		f, err := C.Get(typ, id)
		if err != nil {
			return fmt.Errorf("Execute: Cannot Get '%s:%s': %s", typ, id, err)
		}
		C.Execute(f, w)
		s = s[j+2:]
	}
	w.Write([]byte(s))
	return nil
}

func PreRender(text string, v Values) (*Fragment, error) {
	fmap := map[string]interface{}{
		"fragment": func(typ, id interface{}) string {
			return fmt.Sprintf("{{%s:%s}}", typ, id)
		},
	}
	t, err := template.New("").Funcs(fmap).Parse(text)
	if err != nil {
		return nil, fmt.Errorf("Parse error: %s", err)
	}
	var b bytes.Buffer
	err = t.Execute(&b, v)
	if err != nil {
		return nil, fmt.Errorf("Exec error: %s", err)
	}
	f := Fragment(b.String())
	return &f, nil
}
