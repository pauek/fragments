package fragments

import (
	"bytes"
	"fmt"
	"testing"
)

func TestOneFragment(t *testing.T) {
	env := NewEnviron(nil)

	msg := "A simple fragment with id = '%s'"
	Add("myfragment", Generator{
		Func: func(e *Environ, id string) (string, []string, error) {
			return fmt.Sprintf(msg, id), nil, nil
		},
	})

	for _, s := range []string{"a", "a b", "a b c"} {
		fa := env.Get("myfragment", s)
		if fa == nil {
			t.Fail()
			continue
		}
		str, err := fa.Render(env)
		if err != nil {
			t.Errorf("Render failed: %s", err)
		}
		if str != fmt.Sprintf(msg, s) {
			t.Errorf("Wrong output '%s'", str)
		}
	}
}

var people = map[string]struct{ name, city string }{
	"1": {"Pauek", "a"},
	"2": {"NyanCat", "b"},
}
var cities = map[string]string{
	"a": "Barcelona",
	"b": "4chan",
}

func TestMultipleFragments(t *testing.T) {
	env := NewEnviron(nil)
	txt := `My name is {{.name}} and I live in {{fragment "city" .city}}`
	Add("people", Generator{
		Func: func(e *Environ, id string) (text string, depends []string, err error) {
			p, ok := people[id]
			if !ok {
				return "", nil, fmt.Errorf("id '%s' not found", id)
			}
			text, err = PreRender(txt, Values{
				"name": p.name,
				"city": p.city,
			})
			if err != nil {
				t.Errorf("Template doesn't PreRender: %s", err)
			}
			depends = []string{"people:" + id}
			return
		},
	})

	// get fragment
	f := env.Get("people", "1")
	if f == nil {
		t.Errorf("Fragment is null!")
		return
	}

	// execute without city
	res, err := f.Render(env)
	if err == nil {
		t.Errorf("There should be an error")
	}

	Add("city", Generator{
		Func: func(e *Environ, id string) (text string, depends []string, err error) {
			city, ok := cities[id]
			if !ok {
				return "", nil, fmt.Errorf("City '%s' not found", id)
			}
			text = fmt.Sprintf("%s", city)
			depends = []string{"cities"}
			return text, depends, nil
		},
	})

	f = env.Get("people", "1")
	res, err = f.Render(env)
	if err != nil {
		t.Errorf("Render gave an error")
	}
	if res != "My name is Pauek and I live in Barcelona" {
		t.Errorf("Wrong output!")
	}
}

// Test invalidation
func TestInvalidation(t *testing.T) {
	var a = 1
	env := NewEnviron(nil)
	Add("a", Generator{
		Func: func(e *Environ, id string) (string, []string, error) {
			return fmt.Sprintf("Value = %d", a), []string{"a"}, nil
		},
	})
	test := func(expect string) {
		f := env.Get("a", "")
		res, err := f.Render(env)
		if err != nil {
			t.Errorf("Should not give error")
		}
		if res != expect {
			t.Errorf("Wrong output")
		}
	}

	test("Value = 1")
	a = 2
	test("Value = 1")
	Invalidate("a")
	test("Value = 2")
}

// Test layers
func TestLayers(t *testing.T) {
	txt := `The value is {{fragment "data" .id}}`
	Add("page", Generator{
		Func: func(env *Environ, id string) (string, []string, error) {
			text, err := PreRender(txt, Values{
				"id": id,
			})
			if err != nil {
				return "", nil, err
			}
			return text, nil, nil
		},
	})
	Add("data", Generator{
		Layer: "data",
		Func: func(env *Environ, id string) (string, []string, error) {
			return fmt.Sprintf("[%s] %v", id, env.Data), nil, nil
		},
	})
	env := NewEnviron(nil)
	e1 := NewEnviron(env)
	e1.NewLayer("data")
	e1.Data = 1

	content := `Here is the page: {{fragment "page" .id}}`
	tmpl, err := Parse(content)
	if err != nil {
		t.Errorf("Cannot parse")
	}
	var b1 bytes.Buffer
	err = e1.Execute(tmpl, &b1, Values{
		"id": "xxx",
	})
	if err != nil {
		t.Errorf("Error: %s\n", err)
	}
	if b1.String() != "Here is the page: The value is [xxx] 1" {
		t.Errorf("Wrong output")
	}

	e2 := NewEnviron(env)
	e2.NewLayer("user")
	e2.Data = "nyan"

	var b2 bytes.Buffer
	err = e2.Execute(tmpl, &b2, Values{
		"id": "yyy",
	})
	if err != nil {
		t.Errorf("Error: %s\n", err)
	}
	if b2.String() != "Here is the page: The value is [yyy] nyan" {
		t.Errorf("Wrong output")
	}
	fmt.Printf("%d\n", len(env.layers[""].fragments))
	fmt.Printf("%d\n", len(e1.layers[""].fragments))
	fmt.Printf("%d\n", len(e2.layers[""].fragments))
	fmt.Printf("%d\n", len(e1.layers["data"].fragments))
	fmt.Printf("%d\n", len(e2.layers["data"].fragments))
}
