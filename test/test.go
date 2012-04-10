package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"time"

	"fragments"
)

const T = `<div id="{{ .id }}" class="comments">
<p>{{ .comment }}</p>
by {{ .user }}
</div>
`

func comments(id string) string {
	cT, _ := template.New("comment").Parse(T)
	var b bytes.Buffer
	cT.Execute(&b, fragments.Values{
		"id":      id,
		"comment": "Blah blah",
		"user":    fragments.Ref("user:pauek"),
	})
	return b.String()
}

func user(id string) string {
	s := `<div class="user">User: %s [%s]</div>`
	return fmt.Sprintf(s, id, fragments.Ref("clock:now"))
}

func clock(id string) (res string) {
	if id == "now" {
		now := time.Now()
		res = fmt.Sprintf("%02d:%02d:%02d", now.Hour(), now.Minute(), now.Second())
	} else {
		res = fmt.Sprintf("[not supported (%s)]", id)
	}
	return
}

func main() {
	fragments.Add("comments", comments)
	fragments.Add("user", user)
	fragments.Add("clock", clock)

	frag := fragments.Get("comments:jarl")
	frag.Render(os.Stdout)
	ago := time.Now().Add(-10 * time.Minute)
	D := frag.Diff(ago)
	J, _ := json.Marshal(D)
	fmt.Println(D)
	fmt.Println(string(J))
}
