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

func init() {
	layout, _ = frag.Parse(tLayout)
}

func fHome(C *frag.Cache, args []string) frag.Fragment {
	return frag.Text(tHome)
}

func fPage(C *frag.Cache, args []string) frag.Fragment {
	return layout.RenderFn(func(w io.Writer, id string, mode frag.Mode) {
		if id == "body" && mode == frag.Recursive {
			C.Get(args[1]).Render(w, C, mode)
		}
	})
}

func main() {
	frag.Register("page", fPage)
	frag.Register("home", fHome)
	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		frag.Render(w, "page "+req.URL.Path[1:])
	})
	http.ListenAndServe(":8080", nil)
}
