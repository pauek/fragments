/*

Package fragments provides a simple way to cache pieces of (web)
text. A fragment is just an object which can be rendered (i.e., has
interface Renderer). The package provides three types of fragment:
Text, Ids and Templates.

Here is an example usage of the package:

   fragments.Register("salute", func(C *Cache, args []string) fragments.Renderer {
      return fragments.Text("hello, " + args[1] + "\n")
   })
   fragments.Register("pair-salute", func(C *Cache, a []string) fragments.Renderer {
      s := fmt.Sprintf("salute 1: {salute %s}salute 2: {salute %s}", a[1], a[2])
      f, _ := fragments.Parse(s, "{", "}")
      return f
   })
   ...
   fragments.Get("salute pauek").Render(os.Stdout)
   fragments.Get("pair-salute pauek other").Render(os.Stdout)

*/
package fragments

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

type Mode int

const (
	Recursive = Mode(0)
	NonRecursive = Mode(1)
)

type Fragment interface {
	Render(w io.Writer, C *Cache, mode Mode)
	EachChild(fn func(id string))
}

// Text

type Text string

func (t Text) Render(w io.Writer, C *Cache, mode Mode) {
	w.Write([]byte(t))
}

func (t Text) EachChild(fn func(id string)) {}

// Ref

type Ref string

func (r Ref) id() string { return string(r) }

func (r Ref) Render(w io.Writer, C *Cache, mode Mode) {
	C.get(string(r)).frag.Render(w, C, mode)
}

func (r Ref) EachChild(fn func(id string)) { fn(string(r)) }

// Template

type Template []Fragment

func (t Template) Each(fn func (f Fragment)) {
	for _, elem := range t {
		fn(elem)
	}
}

func (t Template) Render(w io.Writer, C *Cache, mode Mode) {
	t.Each(func(f Fragment) { f.Render(w, C, mode) })
}

func (t Template) EachChild(fn func(id string)) {
	t.Each(func(f Fragment) { f.EachChild(fn) })
}

var ParseError = errors.New("Parse: unmatched delimiters")

// Parse some text looking for pieces enclosed within ldelim and
// rdelim and return a Template (or an OpenDelim error)
func Parse(s string, ldelim, rdelim string) (tmpl Template, err error) {
	lsz := len(ldelim)
	rsz := len(rdelim)
	for {
		i := strings.Index(s, ldelim)
		j := strings.Index(s, rdelim)
		if i == -1 && j == -1 {
			break
		} else if i == -1 || i > j {
			return nil, ParseError
		}
		text := Text(s[:i])
		if text != "" {
			tmpl = append(tmpl, text)
		}
		id := Ref(s[i+lsz : j])
		if id != "" {
			tmpl = append(tmpl, id)
		}
		s = s[j+rsz:]
	}
	if len(s) > 0 {
		tmpl = append(tmpl, Text(s))
	}
	return tmpl, nil
}

// Cache

type Cache struct {
	cache    map[string]cacheItem
	registry map[string]Generator
	deps     map[string][]string
}

type Generator func(C *Cache, args []string) Fragment

type cacheItem struct {
	frag  Fragment
	stamp time.Time
	valid bool
}

type ListItem struct{ id, text string }

func NewCache() (C *Cache) {
	C = new(Cache)
	C.cache = make(map[string]cacheItem)
	C.registry = make(map[string]Generator)
	C.deps = make(map[string][]string)
	return C
}

func (C *Cache) get(id string) cacheItem {
	item, ok := C.cache[id]
	if !ok || !item.valid {
		item = C.generate(id)
		C.cache[id] = item
	}
	return item
}

func (C *Cache) generate(id string) (item cacheItem) {
	args := strings.Split(id, " ")
	gen, ok := C.registry[args[0]]
	item.stamp = time.Now()
	item.valid = true
	if !ok {
		item.frag = Text(fmt.Sprintf("No generator '%s'", args[0]))
		return
	}
	item.frag = gen(C, args)
	return
}

func (C *Cache) Register(id string, gen Generator) {
	C.registry[id] = gen
}

func (C *Cache) Render(w io.Writer, id string) {
	C.get(id).frag.Render(w, C, Recursive)
}

func (C *Cache) List(id string) (list []ListItem) {
	var zero time.Time
	return C.DiffList(id, zero)
}

func (C *Cache) DiffList(id string, since time.Time) (list []ListItem) {
	item := C.get(id)
	list = []ListItem{{id: id}}
	if item.stamp.After(since) {
		var b bytes.Buffer
		item.frag.Render(&b, C, NonRecursive)
		list[0].text = b.String()
	} 
	item.frag.EachChild(func (id string) {
		sublist := C.DiffList(id, since)
		list = append(list, sublist...)
	})
	return list
}

func (C *Cache) Depends(id string, ids ...string) {
	C.deps[id] = append(C.deps[id], ids...)
}

func (C *Cache) show() {
	for id, item := range C.cache {
		fmt.Printf("%#v %#v\n", id, item)
	}
}

// fragments for closures

type FragFn func(w io.Writer, C *Cache, mode Mode)

func (f FragFn) Render(w io.Writer, C *Cache, m Mode) { f(w, C, m) }
func (f FragFn) EachChild(fn func(id string))         {}

func RenderFunc(fn func(w io.Writer, C *Cache, m Mode)) Fragment {
	return FragFn(fn)
}

// defaultCache

var defaultCache *Cache

func init() {
	defaultCache = NewCache()
}

func Register(id string, gen Generator) {
	defaultCache.Register(id, gen)
}

func Render(w io.Writer, id string) {
	defaultCache.Render(w, id)
}

func List(id string) []ListItem {
	return defaultCache.List(id)
}

func DiffList(id string, since time.Time) []ListItem {
	return defaultCache.DiffList(id, since)
}
