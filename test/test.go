package main

import (
	"fmt"
	frag "fragments"
	"os"
)

func main() {
	frag.Register("salute", func(args []string) frag.Renderer {
		return frag.Text("hello, " + args[0] + "\n")
	})
	frag.Register("pair-salute", func(a []string) frag.Renderer {
		s := fmt.Sprintf("salute 1: {salute %s}salute 2: {salute %s}", a[0], a[1])
		f, _ := frag.Parse(s, "{", "}")
		return f
	})
	frag.Get("salute pauek").Render(os.Stdout)
	frag.Get("pair-salute pauek other").Render(os.Stdout)
}