package fragments

import (
	"fmt"
	"testing"
)

func TestOneFragment(t *testing.T) {
	env := NewEnviron()

	msg := "A simple fragment with id = '%s'"
	Add("myfragment", Generator{
		Func: func(id string) (string, []string, error) {
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
	env := NewEnviron()
	txt := `My name is {{.name}} and I live in {{fragment "city" .city}}`
	Add("people", Generator{
		Func: func(id string) (text string, depends []string, err error) {
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
		Func: func(id string) (text string, depends []string, err error) {
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

// Test Layers
