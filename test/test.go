package main

import (
	frag "fragments"
	"net/http"
	"io"
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

func init() {
	layout, _ = frag.Parse(tLayout, "{{", "}}")
}

type Page string


func fHome(args []string) frag.Renderer {
	return frag.Text(tHome)
}

func fPage(args []string) frag.Renderer {
	return frag.RenderFunc(func (w io.Writer) {
		layout.Exec(w, func(id string) {
			frag.Render(w, args[0])
		})
	})
}

func main() {
	frag.Register("page", fPage)
	frag.Register("home", fHome)
	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		frag.Render(w, "page " + req.URL.Path[1:])
	})
	http.ListenAndServe(":8080", nil)
}
