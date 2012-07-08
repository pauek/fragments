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
	"io/ioutil"
	_ "log"
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
		} else {
			inDiv(w, id, nil)
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

func (t Template) RenderFn(fn func(_w io.Writer, _id string)) RenderFn {
	return RenderFn(func(w io.Writer, C *Cache, mode Mode) {
		t.Exec(w, func(id string) {
			if mode == Recursive {
				fn(w, id)
			}
		})
	})
}

// RenderFn

type RenderFn func(w io.Writer, C *Cache, mode Mode)

func (f RenderFn) Render(w io.Writer, C *Cache, m Mode) { f(w, C, m) }
func (f RenderFn) EachChild(fn func(id string))         {}

// Cache

type cacheItem struct {
	frag  Fragment
	stamp time.Time
}

type Cache struct {
	cache    map[string]*cacheItem
	valid    map[string]bool
	registry map[string]Generator
	depends  map[string]map[string]bool
}

type Generator func(C *Cache, args []string) Fragment

func NewCache() *Cache {
	C := new(Cache)
	C.cache = make(map[string]*cacheItem)
	C.valid = make(map[string]bool)
	C.registry = make(map[string]Generator)
	C.depends = make(map[string]map[string]bool)
	return C
}

func (C *Cache) get(id string) *cacheItem {
	f, ok := C.cache[id]
	if !ok || !C.valid[id] {
		C.valid[id] = true
		f = &cacheItem{
			frag:  C.generate(id),
			stamp: time.Now(),
		}
		C.cache[id] = f
	} else {
		// log.Printf("Hit: %q", id)
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

func inDiv(w io.Writer, id string, fn func()) {
	fmt.Fprintf(w, `<div fragment="%s">`, id)
	if fn != nil {
		fn()
	}
	fmt.Fprintf(w, `</div>`)
}

func (C *Cache) Render(w io.Writer, id string) {
	inDiv(w, id, func () {
		C.get(id).frag.Render(w, C, Recursive)
	})
}

type ListItem struct {
	Id, Html string
	Stamp time.Time
}

func (C *Cache) listDiff(id string, since time.Time) (list []ListItem) {
	item := C.get(id)
	list = []ListItem{{Id: id, Stamp: item.stamp}}
	if item.stamp.After(since) {
		var b bytes.Buffer
		inDiv(&b, id, func () {
			item.frag.Render(&b, C, NonRecursive)
		})
		list[0].Html = b.String()
	}
	item.frag.EachChild(func(id string) {
		sublist := C.listDiff(id, since)
		list = append(list, sublist...)
	})
	return
}

func addRoot(id string, list []ListItem) []ListItem {
	root := ListItem{Id: "_root", Stamp: time.Now()}
	root.Html = fmt.Sprintf(`<div fragment="%s"></div>`, id)
	return append([]ListItem{root}, list...)
}

func (C *Cache) ListDiff(id string, since time.Time) (list []ListItem) {
	return addRoot(id, C.listDiff(id, since))
}

func (C *Cache) List(id string) []ListItem {
	var zerotime time.Time
	return C.ListDiff(id, zerotime)
}

// Declare dependencies: fragment fid depends on all objects in the
// oids list
//
func (C *Cache) Depends(fid string, oids ...string) {
	for _, oid := range oids {
		if C.depends[oid] == nil {
			C.depends[oid] = make(map[string]bool)
		}
		C.depends[oid][fid] = true
	}
}

// Invalidate fragment with ID fid (if it exists)
//
func (C *Cache) Invalidate(fid string) {
	if _, found := C.valid[fid]; found {
		C.valid[fid] = false
	}
}

// Invalidate all fragments associated with object oid
//
func (C *Cache) Touch(oid string) {
	for fid := range C.depends[oid] {
		C.Invalidate(fid)
	}
}

// Default Parser

var DefaultParser = Parser{"{% ", " %}"}

func Parse(s string) (Template, error) {
	return DefaultParser.Parse(s)
}

func MustParse(s string) (t Template) {
	if t, err := Parse(s); err == nil {
		return t
	}
	panic(fmt.Sprintf("Cannot Parse"))
}

func MustParseFile(filename string) (t Template) {
	if data, err := ioutil.ReadFile(filename); err == nil {
		return MustParse(string(data))
	}
	panic(fmt.Sprintf("Cannot Read File '%s'", filename))
}
