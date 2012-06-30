package main

import (
	frag "fragments"
	"io"
	"net/http"
)

const tLayout = `
<!doctype html>
<html>
  <head></head>
  <body>{{body}}</body>
</html>
`

const tHome = `
<h1>Home</h1>
<p>This is the home page</p>
`

var layout frag.Template
var cache *frag.Cache

func init() {
	layout, _ = frag.Parse(tLayout, "{{", "}}")
	cache = frag.NewCache()
}

type Page string


func fHome(C *frag.Cache, args []string) frag.Fragment {
	return frag.Text(tHome)
}

func fPage(C *frag.Cache, args []string) frag.Fragment {
	return frag.RenderFunc(func (w io.Writer, C *frag.Cache, m frag.Mode) {
		layout.Each(func (f frag.Fragment) {
			if _, ok := f.(frag.Ref); ok {
				C.Render(w, args[1]) // assume 'body'
				return
			}
			f.Render(w, C, m)
		})
	})
}

func main() {
	cache.Register("page", fPage)
	cache.Register("home", fHome)
	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		cache.Render(w, "page "+req.URL.Path[1:])
	})
	http.ListenAndServe(":8080", nil)
}
