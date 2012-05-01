package fragments

import (
	"bytes"
	"fmt"
	tmpl "html/template"
	"io"
	"time"
)

type Values map[string]interface{}

type Fragment struct {
	typ       string
	id        string
	valid     bool
	text      string
	tmpl      *tmpl.Template
	timestamp time.Time
	err       error // Keeps the error returned by the GenFunc
}

// Write the fragment's text into an io.Writer
func (frag *Fragment) Execute(env *Environ, w io.Writer) error {
	if frag.tmpl == nil {
		return fmt.Errorf("No template!")
	}
	return env.Execute(frag.tmpl, w, nil)
}

func (frag *Fragment) Valid() bool {
	return frag.err == nil && frag.valid
}

func (frag *Fragment) Error() error {
	return frag.err
}

// Produce a string with the fragment
func (frag *Fragment) Render(env *Environ) (text string, err error) {
	var b bytes.Buffer
	err = frag.Execute(env, &b)
	if err != nil {
		return "", err
	}
	return b.String(), nil
}

// Generators

type GenFunc func(id string) (string, []string, error)

type Generator struct {
	Layer string
	Func  GenFunc
}

var generators = make(map[string]Generator)

func Add(id string, gen Generator) {
	generators[id] = gen
}

// Environments

type Layer struct {
	id        string
	fragments map[string]*Fragment
}

type Environ struct {
	layers map[string]*Layer
}

func NewEnviron() *Environ {
	E := &Environ{
		layers: make(map[string]*Layer),
	}
	root := &Layer{
		id:        "",
		fragments: make(map[string]*Fragment),
	}
	E.layers[""] = root
	return E
}

func (E *Environ) NewLayer(id string) *Layer {
	neu := &Layer{
		id:        id,
		fragments: make(map[string]*Fragment),
	}
	E.layers[id] = neu
	return neu
}

func fmap(E *Environ) map[string]interface{} {
	return map[string]interface{}{
		"fragment": func(typ, id string) (string, error) {
			frag := E.Get(typ, id)
			if frag == nil {
				return "", fmt.Errorf("ERROR: Not found: '%s:%s'", typ, id)
			}
			text, err := frag.Render(E)
			if err != nil {
				return "", fmt.Errorf("ERROR: Cannot render '%s:%s': %s", typ, id, err)
			}
			return text, nil
		},
	}
}

// Produce a fragment using the GenFunc
func (E *Environ) make(typ, id string) *Fragment {
	frag := &Fragment{typ: typ, id: id}
	generator, ok := generators[typ]
	if !ok {
		panic(fmt.Sprintf("No generator found for type '%s'", typ))
	}
	text, _, err := generator.Func(id)
	// TODO: handle deps
	if err != nil {
		frag.err = err
		return frag
	}
	tmpl, err := tmpl.New(typ + ":" + id).Funcs(fmap(E)).Parse(text)
	if err != nil {
		frag.err = fmt.Errorf("Error Parsing text: %s", err)
	}
	frag.tmpl = tmpl
	frag.valid = true
	frag.timestamp = time.Now()
	return frag
}

func (E *Environ) Get(typ, id string) *Fragment {
	generator, ok1 := generators[typ]
	if !ok1 {
		// panic(fmt.Sprintf("No generator found for type '%s'", typ))
		return nil
	}
	layer, ok2 := E.layers[generator.Layer]
	if !ok2 {
		layer = E.NewLayer(generator.Layer)
	}
	fragment, ok3 := layer.fragments[typ+":"+id]
	if !ok3 {
		fragment = E.make(typ, id)
	}
	return fragment
}

func PreRender(text string, values Values) (string, error) {
	fmap := map[string]interface{}{
		"fragment": func (typ, id interface{}) tmpl.HTML {
			return tmpl.HTML(fmt.Sprintf(`{{fragment "%s" "%s"}}`, typ, id))
		},
	}
	tmpl, err := tmpl.New("").Funcs(fmap).Parse(text)
	if err != nil {
		return "", nil
	}
	var b bytes.Buffer
	err = tmpl.Execute(&b, values)
	if err != nil {
		return "", fmt.Errorf("Cannot Execute: %s", err)
	}
	return b.String(), nil
}

func (E *Environ) Execute(t *tmpl.Template, w io.Writer, data interface{}) error {
	return t.Funcs(fmap(E)).Execute(w, data)
}
