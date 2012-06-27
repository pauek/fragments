/*

Package fragments provides a simple way to cache pieces of (web)
text. A fragment is just an object which can be rendered (i.e., has
interface Renderer). The package provides three types of fragment:
Text, Ids and Templates.

Here is an example usage of the package:

   fragments.Register("salute", func(args []string) fragments.Renderer {
      return fragments.Text("hello, " + args[0] + "\n")
   })
   fragments.Register("pair-salute", func(a []string) fragments.Renderer {
      s := fmt.Sprintf("salute 1: {salute %s}salute 2: {salute %s}", a[0], a[1])
      f, _ := fragments.Parse(s, "{", "}")
      return f
   })
   ...
   fragments.Get("salute pauek").Render(os.Stdout)
   fragments.Get("pair-salute pauek other").Render(os.Stdout)

*/
package fragments

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

// Renderer 
type Renderer interface {
	Render(w io.Writer)
}

type renderFn func(w io.Writer)

func (f renderFn) Render(w io.Writer) { f(w) }

func RenderFunc(fn func(w io.Writer)) Renderer {
	return renderFn(fn)
}

// Id stores ids for fragments.
type Id string
type Text string
type Error string

func (E Error) Render(w io.Writer) {
	w.Write([]byte(E))
}

func (T Text) Render(w io.Writer) {
	w.Write([]byte(T))
}

// Template is a type of fragment: a simple array of strings and Ids.
type Template []interface{}

// Exec executes the template traversing the array and writing pieces
// of text to w and calling function fn for every Id.
func (T Template) Exec(w io.Writer, fn func (id string)) {
	for _, f := range T {
		switch f.(type) {
		case string:
			w.Write([]byte(f.(string)))
		case Id:
			fn(string(f.(Id)))
		}
	}
}

// Render for Templates invokes the Render method on all
// elements of the array
func (T Template) Render(w io.Writer) {
	T.Exec(w, func(id string) { Get(id).Render(w) })
}

func (T Template) String() (s string) {
	for _, f := range T {
		s += fmt.Sprintf("%v", f)
	}
	return s
}

// Returned as a Parse error
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
		text := s[:i]
		if text != "" {
			tmpl = append(tmpl, text)
		}
		id := Id(s[i+lsz : j])
		if id != "" {
			tmpl = append(tmpl, id)
		}
		s = s[j+rsz:]
	}
	if len(s) > 0 {
		tmpl = append(tmpl, s)
	}
	return tmpl, nil
}

// Cache

type cacheitem struct {
	r Renderer
	t time.Time
}

var cache = make(map[string]cacheitem)

// Get a fragment with identifier id, as well as the time of
// generation. If the fragment doesn't exist in the cache, it is
// created with its generator
func GetT(id string) (r Renderer, t time.Time) {
	f, ok := cache[id]
	if ok {
		return f.r, f.t
	}
	r, t = generate(id)
	cache[id] = cacheitem{r, t}
	return
}

// Get is the same as GetT but does no return the time
func Get(id string) Renderer {
	f, _ := GetT(id)
	return f
}

// Render a fragment with id to writer w
func Render(w io.Writer, id string) {
	Get(id).Render(w)
}

func showCache() {
	for id, item := range cache {
		fmt.Printf("%s: %+v\n", id, item)
	}
}

// Registry

// Generator is the type of function that generates new fragments when
// they are requested
type Generator func(args []string) Renderer

var registry = make(map[string]Generator)

func generate(id string) (f Renderer, t time.Time) {
	args := strings.Split(string(id), " ")
	gen, ok := registry[args[0]]
	if !ok {
		return Error(fmt.Sprintf("no generator '%s'", args[0])), time.Now()
	}
	return gen(args[1:]), time.Now()
}

// Register a fragment type. typ is the type name of the fragment, and
// fn is a generator function.
func Register(typ string, fn Generator) {
	registry[typ] = fn
}

// Invalidation