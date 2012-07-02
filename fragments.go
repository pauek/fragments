/*

Package fragments provides a simple way to cache pieces of web text. A
fragment is an object which can be rendered and can inform about
children (has interface Fragment). The package provides two types of
simple fragments: Text and Templates.

Here is an example usage of the package:

   fragments.Register("salute", func(C *Cache, args []string) fragments.Fragment {
      return fragments.Text("hello, " + args[1] + "\n")
   })
   fragments.Register("pair-salute", func(C *Cache, a []string) fragments.Fragment {
      s := fmt.Sprintf("salute 1: {salute %s}salute 2: {salute %s}", a[1], a[2])
      f, _ := fragments.Parser{"{", "}"}.Parse(s)
      return f
   })
   ...
   fragments.Render(os.Stdout, "salute pauek")
   fragments.Render(os.Stdout, "pair-salute pauek other")

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
	Recursive    = Mode(0)
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

type tmplItem struct {
	text     string
	isAction bool
}

// Template

type Template []tmplItem

type Parser struct {
	Ldelim, Rdelim string
}

var ParseError = errors.New("Unmatched delimiters")

func (p Parser) Parse(s string) (Template, error) {
	t := []tmplItem{}
	lsz, rsz := len(p.Ldelim), len(p.Rdelim)
	for {
		i := strings.Index(s, p.Ldelim)
		if i == -1 {
			break
		}
		t = append(t, tmplItem{s[:i], false})
		s = s[i+lsz:]
		j := strings.Index(s, p.Rdelim)
		if j == -1 {
			return nil, ParseError
		}
		t = append(t, tmplItem{s[:j], true})
		s = s[j+rsz:]
	}
	if len(s) > 0 {
		t = append(t, tmplItem{s, false})
	}
	return Template(t), nil
}

func (t Template) Exec(w io.Writer, fn func(action string)) {
	for _, item := range t {
		if item.isAction {
			fn(item.text)
		} else {
			w.Write([]byte(item.text))
		}
	}
}

func (t Template) Render(w io.Writer, C *Cache, mode Mode) {
	t.Exec(w, func(id string) {
		if mode == Recursive {
			C.Render(w, id)
		}
	})
}

func (t Template) EachChild(fn func(id string)) {
	for _, item := range t {
		if item.isAction {
			fn(item.text)
		}
	}
}

func (t Template) RenderFn(fn func(_w io.Writer, _id string, _m Mode)) RenderFn {
	return RenderFn(func(w io.Writer, C *Cache, m Mode) {
		t.Exec(w, func(id string) { fn(w, id, m) })
	})
}

// RenderFn

type RenderFn func(w io.Writer, C *Cache, mode Mode)

func (f RenderFn) Render(w io.Writer, C *Cache, m Mode) { f(w, C, m) }
func (f RenderFn) EachChild(fn func(id string))         {}

// Cache

type cacheItem struct {
	frag  Fragment
	valid bool
	stamp time.Time
}

type Cache struct {
	cache    map[string]*cacheItem
	registry map[string]Generator
	depends  map[string]map[string]bool
}

type Generator func(C *Cache, args []string) Fragment

func NewCache() *Cache {
	C := new(Cache)
	C.cache = make(map[string]*cacheItem)
	C.registry = make(map[string]Generator)
	C.depends = make(map[string]map[string]bool)
	return C
}

func (C *Cache) get(id string) *cacheItem {
	f, ok := C.cache[id]
	if !ok || !f.valid {
		f = &cacheItem{
			frag:  C.generate(id),
			valid: true,
			stamp: time.Now(),
		}
		C.cache[id] = f
	}
	return f
}

func (C *Cache) Get(id string) Fragment {
	return C.get(id).frag
}

func (C *Cache) generate(id string) Fragment {
	args := strings.Split(id, " ")
	gen, ok := C.registry[args[0]]
	if !ok {
		return Text(fmt.Sprintf(`No generator "%s"`, args[0]))
	}
	return gen(C, args)
}

func (C *Cache) Register(id string, gen Generator) {
	C.registry[id] = gen
}

func Static(f Fragment) Generator {
	return func(C *Cache, args []string) Fragment {
		return f
	}
}

func StaticText(text string) Generator {
	return Static(Text(text))
}

func (C *Cache) Render(w io.Writer, id string) {
	fmt.Fprintf(w, `<div fragment="%s">`, id)
	C.get(id).frag.Render(w, C, Recursive)
	fmt.Fprintf(w, `</div>`)
}

type ListItem struct {
	id, text string
}

func (C *Cache) ListDiff(id string, since time.Time) (list []ListItem) {
	list = []ListItem{{id: id}}
	item := C.get(id)
	if item.stamp.After(since) {
		var b bytes.Buffer
		item.frag.Render(&b, C, NonRecursive)
		list[0].text = b.String()
	}
	item.frag.EachChild(func(id string) {
		sublist := C.List(id)
		list = append(list, sublist...)
	})
	return
}

func (C *Cache) List(id string) []ListItem {
	var zerotime time.Time
	return C.ListDiff(id, zerotime)
}

func (C *Cache) Depends(fid string, oids ...string) {
	for _, oid := range oids {
		if C.depends[oid] == nil {
			C.depends[oid] = make(map[string]bool)
		}
		C.depends[oid][fid] = true
	}
}

func (C *Cache) Invalidate(oid string) {
	for fid, _ := range C.depends[oid] {
		C.get(fid).valid = false
	}
}

// Defaults

var DefaultCache *Cache
var DefaultParser = Parser{"{{", "}}"}

func init() {
	DefaultCache = NewCache()
}

func Get(id string) Fragment {
	return DefaultCache.Get(id)
}

func Render(w io.Writer, id string) {
	DefaultCache.Render(w, id)
}

func List(id string) []ListItem {
	return DefaultCache.List(id)
}

func ListDiff(id string, since time.Time) []ListItem {
	return DefaultCache.ListDiff(id, since)
}

func Register(id string, gen Generator) {
	DefaultCache.Register(id, gen)
}

func Depends(fid string, oids ...string) {
	DefaultCache.Depends(fid, oids...)
}

func Invalidate(oid string) {
	DefaultCache.Invalidate(oid)
}

func Parse(s string) (Template, error) {
	return DefaultParser.Parse(s)
}
