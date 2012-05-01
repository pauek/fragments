package fragments

import (
	"bytes"
	"fmt"
	"testing"
)

func TestPreRender(t *testing.T) {
	f, err := PreRender(`Hola {{.user}}, que tal {{fragment .typ .id}}`, Values{
		"user": "pauek",
		"typ":  "jarl",
		"id":   "demor",
	})
	if err != nil {
		t.Errorf("PreRender error: %s\n", err)
	}
	if string(*f) != "Hola pauek, que tal {{jarl:demor}}" {
		t.Errorf("Wrong output")
	}
	C := NewCache("blah")
	var b bytes.Buffer
	err = C.Execute(f, &b)
	if err == nil {
		t.Errorf("There should be an error")
	}
}

func TestGet(t *testing.T) {
	Add("item", Generator{
		Func: func(id string, data interface{}) (*Fragment, []string, error) {
			f := Fragment(fmt.Sprintf("[%s] This is item %s", data, id))
			return &f, nil, nil
		},
	})
	C := NewCache("hi, there")
	f, err := C.Get("item", "number two")
	if err != nil {
		t.Errorf("Cannot get item")
	}
	res := "[hi, there] This is item number two"
	if string(*f) != res {
		t.Errorf("Wrong output")
	}
	var b bytes.Buffer
	err = C.Execute(f, &b)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if b.String() != res {
		t.Errorf("Wrong output")
	}
}

func TestExecute(t *testing.T) {
	Add("test", Generator{
		Func: func(id string, data interface{}) (*Fragment, []string, error) {
			f := Fragment("test(" + id + ")")
			return &f, nil, nil
		},
	})
	C := NewCache(nil)
	f := Fragment(`This is it: {{test:blah}}`)
	var b bytes.Buffer
	if err := C.Execute(&f, &b); err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if b.String() != "This is it: test(blah)" {
		t.Errorf("Wrong output: %s", b.String())
	}
}

func TestExecute2(t *testing.T) {
	Add("a", Generator{
		Func: func(id string, data interface{}) (*Fragment, []string, error) {
			f, err := PreRender(`whoa, a frag {{fragment "b" .id}}`, Values{
				"id": id,
			})
			if err != nil {
				return nil, nil, err
			}
			return f, nil, nil
		},
	})
	Add("b", Generator{
		Func: func(id string, data interface{}) (*Fragment, []string, error) {
			f := Fragment(`like "b with ` + id + `"`)
			return &f, nil, nil
		},
	})
	C := NewCache(nil)
	f := Fragment(`before {{a:hey}} middle {{a:ho}} after`)
	var b bytes.Buffer
	if err := C.Execute(&f, &b); err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	s := `before whoa, a frag like "b with hey" middle `
	s += `whoa, a frag like "b with ho" after`
	if b.String() != s {
		t.Errorf("Wrong output")
	}
}
