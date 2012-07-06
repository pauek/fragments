package main

import (
	frag "fragments"
	"io"
	"net/http"
)

const tLayout = `<!doctype html><html><head></head><body>{{body}}</body></html>`
const tHome = `<h1>Home</h1>{{hometext}}`
const tHomeText = `<p>This is the home page</p>`

var layout frag.Template

func init() {
	layout, _ = frag.Parse(tLayout)
}

func fHome(C *frag.Cache, args []string) frag.Fragment {
	t, _ := frag.Parse(tHome)
	return t
}

func fPage(C *frag.Cache, args []string) frag.Fragment {
	return layout.RenderFn(func(w io.Writer, id string) {
		if id == "body" {
			C.Render(w, args[1])
		}
	})
}

func main() {
	frag.Register("page", fPage)
	frag.Register("home", fHome)
	frag.Register("hometext", frag.StaticText(tHomeText))
	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		frag.Render(w, "page "+req.URL.Path[1:])
	})
	http.ListenAndServe(":8080", nil)
}
