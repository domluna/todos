package main

import (
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"testing"
)

func Test_Foo(t *testing.T) {
	tags()
}

func Test_Basics(t *testing.T) {
	// need this for name
	f, err := ioutil.TempFile("", "foo")
	if err != nil {
		t.Fatal(err)
	}

	filename := f.Name()
	f.Close()
	os.Remove(filename)

	ts, err := loadTodos(filename)
	if err != nil {
		t.Fatalf("loadTodos() -- no file: got %s, want nil", err)
	}

	ts = rm(ts, "foo")
	if len(ts) != 0 {
		t.Fatalf("add -- same name: expected 0 todo, got %d", len(ts))
	}

	ts = add(ts, newTodo("foo", "fooing some fools", "$HOME"))
	ts = add(ts, newTodo("foo", "fooing some fools", "$HOME"))
	if len(ts) != 1 {
		t.Fatalf("add -- same name: expected 1 todo, got %d", len(ts))
	}

	home := os.ExpandEnv("$HOME")

	ts = add(ts, newTodo("bar", "go to the bar", home))

	ts = rm(ts, "bar")
	if len(ts) != 1 {
		t.Fatalf("rm: expected 2 todo, got %d", len(ts))
	}

	if ts[0].Name != "foo" {
		t.Fatalf("rm -- deleted wrong todo, expected foo, got %s", ts[0].Name)
	}

	// change the pwd
	ts.do("foo")
	wd, _ := os.Getwd()
	if wd != home {
		t.Fatalf("go: expected directory %s, got %s", home, wd)
	}

	err = saveTodos(filename, ts)
	if err != nil {
		t.Fatalf("saveTodos: got %s, want nil", err)
	}

	tx, err := loadTodos(filename)
	if err != nil {
		t.Fatalf("loadTodos: got %s, want nil", err)
	}

	for i := range tx {
		if !reflect.DeepEqual(tx[i], ts[i]) {
			t.Errorf("expected %v, got %v", ts[i], tx[i])
		}
	}
}

var file = `hello there world
// TODO: some comments thingy over here.
// Some more things todo. 
// Even more things todo, wow!.
//
// Something not part of the TODO!
func foo() int {
	return 0
}

// TODO(batman): finish this
func bar() {
}

// TODO: first
// TODO: second
// TODO(domluna): another todo here
`

var tagTable = []struct {
	content string
	tags    []*tag
}{
	{
		content: `hello there world
// TODO: some comments thingy over here.
// Some more things todo. 
// Even more things todo, wow!
//
// Something not part of the TODO!
func foo() int {
	return 0
}

// TODO(batman): finish this
func bar() {
}

// TODO: first
// TODO: second
// TODO(domluna): last todo
func foobar() {
}
`,
		tags: []*tag{
			&tag{
				lineNum: 2,
				desc:    "TODO: some comments thingy over here.\nSome more things todo.\nEven more things todo, wow!",
			},
			&tag{
				lineNum: 11,
				desc:    "TODO(batman): finish this",
			},
			&tag{
				lineNum: 15,
				desc:    "TODO: first",
			},
			&tag{
				lineNum: 16,
				desc:    "TODO: second",
			},
			&tag{
				lineNum: 17,
				desc:    "TODO(domluna): last todo",
			},
		},
	},
}

func Test_Tags(t *testing.T) {
	tags, err := findTags(strings.NewReader(tagTable[0].content))
	if err != nil {
		t.Fatal(err)
	}

	for i := range tags {
		got := tags[i]
		want := tagTable[0].tags[i]
		if !reflect.DeepEqual(got, want) {
			t.Errorf("comparing tags: got %v, want %v", got, want)
		}
	}
}
