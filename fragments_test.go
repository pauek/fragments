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
		Func: func(id string, data interface{}) (Fragment, []string, error) {
			f := Fragment(fmt.Sprintf("[%s] This is item %s", data, id))
			return f, nil, nil
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
