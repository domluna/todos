// todo
//
// todo add T
// todo rm T
// todo rand
// todo do <name>
// todo list
//
// TODO: finish report
// TODO: think about list display in console
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const usageMsg = "" +
	`Usage of todo CLI:
`

func usage() {
	fmt.Fprintln(os.Stderr, usageMsg)
	fmt.Fprintln(os.Stderr, "Flags:")
	flag.PrintDefaults()
	os.Exit(2)
}

// flags
var (
	path    = flag.String("path", "", "path of the todo")
	desc    = flag.String("desc", "", "description of the todo")
	longOut = flag.Bool("long", false, "list output will be more detailed")
)

var todosFile = os.ExpandEnv("$HOME/.todos")

func main() {
	flag.Usage = usage
	flag.Parse()

	if flag.NFlag() == 0 && flag.NArg() == 0 {
		usage()
	}

	// err := parseFlags()
}

func parseFlags() error {
	return nil
}

type Todo struct {
	// ID `json:`
	Name string `json:"name,omitempty"`

	// path (directory) relevant to the todo
	Path string `json:"path,omitempty"`

	// when the todo was created
	Created time.Time `"json:created,omitempty"`

	// what is the todo about?
	Desc string `json:"description,omitempty"`
}

func newTodo(name, desc, path string) *Todo {
	return &Todo{
		Name:    name,
		Desc:    desc,
		Path:    filepath.Clean(os.ExpandEnv(path)),
		Created: time.Now(),
	}
}

type todoSlice []*Todo

func add(ts todoSlice, t *Todo) todoSlice {
	for _, tt := range ts {
		if tt.Name == t.Name {
			return ts
		}
	}
	return append(ts, t)

}

func rm(ts todoSlice, name string) todoSlice {
	for i, tt := range ts {
		if tt.Name == name {
			// https://github.com/golang/go/wiki/SliceTricks
			return append(ts[:i], ts[i+1:]...)
		}
	}
	return ts
}

// list lists the active todos.
// TODO: this sucks right now
func (ts todoSlice) list() {
	fmt.Println()
	for _, t := range ts {
		s := fmt.Sprintf("Task: %s\n", t.Name)
		s += fmt.Sprintf("Created: %s\n", t.Created)
		s += fmt.Sprintf("Path: %s\n", t.Path)
		s += fmt.Sprintf("Description: %s\n", t.Desc)
		fmt.Print(s)
	}
	fmt.Printf("todos left: %d\n", len(ts))
}

// randomTodo picks a random todo from lists of available
// todos to complete.
//
// This one is just for kicks.
func (ts todoSlice) random() *Todo {
	src := rand.NewSource(time.Now().UnixNano())
	r := rand.New(src)
	ix := r.Intn(len(ts))
	return ts[ix]
}

// do changes to the directory of the todo.
func (ts todoSlice) do(name string) error {
	var path string
	for _, t := range ts {
		if t.Name == name {
			path = t.Path
		}
	}

	if path == "" {
		return errors.New("task \"%s\" not found.")
	}

	return os.Chdir(path)
}

// tag represents a TODO tag in a file.
type tag struct {
	lineNum  int
	desc     string
	filename string
}

// tags looks at files in the current directory for "TODO"
// tags and reports the description as well as the filename and line
// number.
//
// Searches for tags like this:
//  TODO: desc...
//  TODO(somename): desc...
//
// A tag description is ended by
//  1. a new tag
//  2. a empty comment,
//	ex. // TODO: tag here
//	    //
//	    // This will not be added to the description.
//  3. end of comment block
//
// This does NOT check for files recursively.
func (ts todoSlice) tags() {
}

func reportFile(filename string, r io.Reader) ([]*tag, error) {
	scanner := bufio.NewScanner(r)

	var tags []*tag
	var tt *tag
	lineNum := 1

	for scanner.Scan() {
		line := scanner.Text() // removes the trailing '\n'

		if isComment(line) {
			idx := indexTag(line)
			trimmed := trimComment(line)

			if tt == nil { // check for tags
				// start writing todo
				if idx != -1 {
					tt = &tag{
						lineNum:  lineNum,
						desc:     line[idx:],
						filename: filename,
					}
				}
			} else if trimmed == "" { // empty comment line
				tags = append(tags, tt)
				tt = nil
			} else if idx != -1 { // another TODO tag in same comment block
				tags = append(tags, tt)
				tt = &tag{
					lineNum:  lineNum,
					desc:     line[idx:],
					filename: filename,
				}
			} else { // add trimmed current line to current tag
				tt.desc += ("\n" + trimmed)
			}

		} else {
			// end of comment block, append current tag
			if tt != nil {
				tags = append(tags, tt)
				tt = nil
			}
		}
		lineNum++
	}

	return tags, scanner.Err()
}

// comment identifiers
var commentIdents = []string{"//", "#"}

func isComment(line string) bool {
	for _, ci := range commentIdents {
		if strings.HasPrefix(line, ci) {
			return true
		}
	}
	return false
}

func trimComment(line string) string {
	var s string
	for _, ci := range commentIdents {
		if strings.HasPrefix(line, ci) {
			s = strings.TrimPrefix(line, ci)
			break
		}
	}
	return strings.TrimSpace(s)
}

var tagIdents = []string{"TODO:", "TODO("}

func indexTag(line string) int {
	for _, ti := range tagIdents {
		ix := strings.Index(line, ti)
		if ix != -1 {
			return ix
		}
	}
	return -1
}

func loadTodos(filename string) (todoSlice, error) {
	buf, err := readFile(filename)
	if err != nil {
		return nil, err
	}

	var ts todoSlice
	r := bytes.NewReader(buf)

	err = json.NewDecoder(r).Decode(&ts)
	if err != nil {
		return nil, err
	}
	return ts, nil
}

func saveTodos(filename string, ts todoSlice) error {
	f, err := os.Create(filename) // deletes the old file
	if err != nil {
		return err
	}
	defer f.Close()

	buf, err := json.MarshalIndent(&ts, "", "\t")
	if err != nil {
		return err
	}
	_, err = f.Write(buf)
	return err
}

func readFile(filename string) ([]byte, error) {
	if _, err := os.Stat(filename); err == nil {
		return ioutil.ReadFile(filename)
	}

	_, err := os.Create(filename)
	if err != nil {
		return nil, err
	}

	// initial json, empty list
	return []byte("[]"), nil
}
