package fragments

import (
	"testing"
)

var splitcases = []struct{
	fullid, typ, id string
}{
	{"a:b", "a", "b"},
	{"a:b:c", "a", "b:c"},
	{"a", "a", ""},
	{"", "", ""},
}

func TestSplitID(t *testing.T) {
	for _, cas := range splitcases {
		typ, id := SplitID(cas.fullid)
		if typ != cas.typ {
			t.Errorf(`"%s" != "%s"`, typ, cas.typ)
		}
		if id != cas.id {
			t.Errorf(`"%s" != "%s"`, id, cas.id)
		}
	}
}

func TestTraverse(t *testing.T) {
	f1 := Fragment("a{{b:xx}}c{{d:yy}}e")
	expText := []string{"a", "c", "e"}
	expChild := []string{"b:xx", "d:yy"}
	f1.traverse(traverser{
	   Text: func (text string) {
			if text != expText[0] {
				t.Errorf(`"%s" != "%s"`, text, expText[0])
			}
			expText = expText[1:]
		},
	   Child: func (typ, id string) error {
			_typ, _id := SplitID(expChild[0])
			if typ != _typ {
				t.Errorf(`"%s" != "%s"`, typ, _typ)
			}
			if id != _id {
				t.Errorf(`"%s" != "%s"`, id, _id)
			}
			expChild = expChild[1:]
			return nil
		},
	})
}



/*
func TestPreRender(t *testing.T) {
	tmpl, err := Parse(`Hola {{.user}}, que tal {{fragment .typ .id}}`)
	if err != nil {
		t.Errorf("Parse error: %s\n", err)
	}
	f, err := PreRender(tmpl, Values{
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
	err = C.Execute(&b, f)
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
	err = C.Execute(&b, f)
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
	if err := C.Execute(&b, &f); err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if b.String() != "This is it: test(blah)" {
		t.Errorf("Wrong output: %s", b.String())
	}
}

func TestExecute2(t *testing.T) {
	tmpl, err := Parse(`whoa, a frag {{fragment "b" .id}}`)
	if err != nil {
		t.Errorf("Parse error: %s\n", err)
	}
	
	Add("a", Generator{
		Func: func(id string, data interface{}) (*Fragment, []string, error) {
			f, err := PreRender(tmpl, Values{
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
	if err := C.Execute(&b, &f); err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	s := `before whoa, a frag like "b with hey" middle `
	s += `whoa, a frag like "b with ho" after`
	if b.String() != s {
		t.Errorf("Wrong output")
	}
}

func TestLayers(t *testing.T) {
	Add("a", Generator{
		Layer: "A",
		Func: func(id string, data interface{}) (*Fragment, []string, error) {
			f := Fragment(fmt.Sprintf(`[%s] Data is %v`, id, data))
			return &f, nil, nil
		},
	})
	Add("b", Generator{
		Func: func(id string, data interface{}) (*Fragment, []string, error) {
			f := Fragment(`[` + id + `] Fragment A is: '{{a:blah}}'`)
			return &f, nil, nil
		},
	})
	f := Fragment(`{{b:hey}}`)
	C := NewCache(nil)
	e1 := NewCache(1)
	e2 := NewCache(2)

	_, err0 := C.RenderLayers(&f, nil)
	if err0 == nil {
		t.Errorf("There should be an error")
	}
	// TODO: check error is "layer 'A' not found"

	s1, err1 := C.RenderLayers(&f, Layers{"A": e1})
	if err1 != nil {
		t.Errorf("Unexpected error: %s\n", err1)
	}
	if s1 != "[hey] Fragment A is: '[blah] Data is 1'" {
		fmt.Printf("output: %s\n", s1)
		t.Errorf("Wrong output")
	}

	s2, err2 := C.RenderLayers(&f, Layers{"A": e2})
	if err2 != nil {
		t.Errorf("Unexpected error: %s\n", err2)
	}
	if s2 != "[hey] Fragment A is: '[blah] Data is 2'" {
		fmt.Printf("output: %s\n", s2)
		t.Errorf("Wrong output")
	}
	
	// check number of fragments each
	if len(C.cache) != 1 || len(e1.cache) != 1 || len(e2.cache) != 1 {
		t.Errorf("Wrong number of fragments")
	}
}

func TestInvalidation(t *testing.T) {
	var a int = 1

	Add("f", Generator{
	Func: func(id string, data interface{}) (*Fragment, []string, error) {
			f := Fragment(fmt.Sprintf("a = %d", a))
			return &f, []string{"a"}, nil
		},
	})
	
	C := NewCache(nil)
	f := Fragment(`{{f}}`)

	test := func (expected string) {
		s, err := C.Render(&f)
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if s != expected {
			t.Errorf("Wrong output '%s' (should be '%s')", s, expected)
		}
	}

	test("a = 1")
	Invalidate("a")
	test("a = 1")
	a = 2
	test("a = 1")
	Invalidate("a")
	test("a = 2")
}
*/