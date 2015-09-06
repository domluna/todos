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
	"path"
	"path/filepath"
	"strings"
	"time"
)

const usageMsg = "" +
	`Usage of the todos CLI:

Add todos with a description and path. If the -path
flag is not present, the current directory will be
used. T is the name of the todo:
	todos -desc="write more tests" add T

Remove a todo with a name T from the list:
	todos rm T

List all todos:
	todos list
	todos list

Show a random todo:
	todos rand
	todos rand

List all the todo tags in the current directory. A tag
is defined and either "TODO:" or "TODO(somename):"
	todos tags

Remove all todos:
	todos clear
`

func usage() {
	fmt.Fprintln(os.Stderr, usageMsg)
	fmt.Fprintln(os.Stderr, "Flags:")
	flag.PrintDefaults()
	os.Exit(2)
}

var (
	todoPath = flag.String("path", "", "path of the todo")
	todoDesc = flag.String("desc", "", "description of the todo")
)

var todosFile = os.ExpandEnv("$HOME/.todos")

func main() {
	flag.Usage = usage
	flag.Parse()

	if flag.NFlag() == 0 && flag.NArg() == 0 {
		usage()
	}

	err := parseFlags()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		fmt.Fprintln(os.Stderr, `For usage information, run "todo -help"`)
		os.Exit(2)
	}

	// load up the current todos
	ts, err := loadTodos(todosFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(2)
	}

	cmd := flag.Arg(0)
	switch cmd {
	case "add":
		todoName := flag.Arg(1)

		var ok bool
		ts, ok = add(ts, newTodo(todoName, *todoDesc, *todoPath))
		if !ok {
			fmt.Fprintf(os.Stderr, "todo with name \"%s\" already exists\n", todoName)
		}
	case "rm":
		todoName := flag.Arg(1)

		var ok bool
		ts, ok = rm(ts, todoName)
		if !ok {
			fmt.Fprintf(os.Stderr, "no todo with name \"%s\"\n", todoName)
		}
	case "ls":
		ts.ls()
	case "rand":
		if len(ts) < 1 {
			fmt.Fprintln(os.Stderr, `no todos left, try adding one with "todo add"`)
			os.Exit(2)
		}
		todo := ts.random()
		fmt.Println(todo)
	case "work":
		// 	todoName := flag.Arg(1)
		// 	err := ts.workOn(todoName)
		// 	if err != nil {
		// 		fmt.Fprintf(os.Stderr, "%v\n", err)
		// 		os.Exit(2)
		// 	}
		//
		// 	fmt.Fprintf(os.Stdin, "$(cd %s)", os.ExpandEnv("$HOME"))
		fmt.Fprintln(os.Stderr, `"todo work" is currently a WIP`)
	// 		os.Exit(2)
	case "tags":
		tags()
	case "clear":
		ts = make(todoSlice, 0)
	case "count":
		fmt.Fprintf(os.Stdout, "%d todo(s) left\n", len(ts))
	default:
		fmt.Fprintf(os.Stderr, "command \"%s\" not valid\n", cmd)
		usage()
	}

	// save changes
	err = saveTodos(todosFile, ts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(2)
	}

}

func parseFlags() error {
	if *todoPath == "" {
		dir, _ := os.Getwd()
		*todoPath = dir
	}

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

// Long form of todo
func (t *Todo) String() string {
	s := ""
	layout := "Mon Jan 2 2006"
	s = fmt.Sprintf("Name: %s\n", t.Name)
	s += fmt.Sprintf("Created: %s\n", t.Created.Format(layout))
	s += fmt.Sprintf("Path: %s\n", t.Path)
	s += fmt.Sprintf("Description: %s", t.Desc)
	return s
}

func add(ts todoSlice, t *Todo) (todoSlice, bool) {
	for _, tt := range ts {
		if tt.Name == t.Name {
			return ts, false
		}
	}
	return append(ts, t), true

}

func rm(ts todoSlice, name string) (todoSlice, bool) {
	for i, tt := range ts {
		if tt.Name == name {
			// https://github.com/golang/go/wiki/SliceTricks
			return append(ts[:i], ts[i+1:]...), true
		}
	}
	return ts, false
}

// list lists the active todos.
// TODO: better date formatting
func (ts todoSlice) ls() {
	if len(ts) < 1 {
		fmt.Println("No todos! Better find something!")
		return
	}
	for _, t := range ts {
		fmt.Printf("%s\n", t)
	}
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

// workOn changes to the directory of the todo with
// given name.
func (ts todoSlice) workOn(name string) error {
	var path string
	for _, t := range ts {
		if t.Name == name {
			path = t.Path
		}
	}

	if path == "" {
		return errors.New(fmt.Sprintf("todo %s not found", name))
	}

	fmt.Println(path)
	return os.Chdir(path)
}

// tag represents a TODO tag in a file.
type tag struct {
	lineNum int
	desc    string
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
func tags() error {
	dir, _ := os.Getwd()
	fmt.Printf("Searching for tags ...\n\n")
	return filepath.Walk(dir, func(fpath string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if fi.IsDir() && path.Base(dir) != fi.Name() {
			return filepath.SkipDir
		}

		if strings.HasPrefix(fi.Name(), ".") || fi.IsDir() {
			return nil
		}

		f, err := os.Open(fpath)
		if err != nil {
			return err
		}
		defer f.Close()

		tags, err := findTags(f)
		if err != nil {
			return err
		}

		fmt.Printf("*** %s ***\n\n", fi.Name())
		s := ""
		for _, t := range tags {
			s += fmt.Sprintf("line: %d\n", t.lineNum)
			s += fmt.Sprintf("%s\n", t.desc)
			s += "\n"
		}
		fmt.Print(s)
		return nil
	})
}

func findTags(r io.Reader) ([]*tag, error) {
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
						lineNum: lineNum,
						desc:    line[idx:],
					}
				}
			} else if trimmed == "" { // empty comment line
				tags = append(tags, tt)
				tt = nil
			} else if idx != -1 { // another TODO tag in same comment block
				tags = append(tags, tt)
				tt = &tag{
					lineNum: lineNum,
					desc:    line[idx:],
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
