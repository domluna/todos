todos
=====

todos is a todo list for the terminal.

```sh
$ go get github.com/domluna/todos
```

### Usage

```
Usage of the todos CLI:

Add a new todo with a description and path. If no path
is given, the current directory will be used. T is the
name of the todo:
	todos -desc="write more tests" new T

Remove a todo with a name T from the list:
	todos rm T

List all todos:
	todos ls

Show a random todo:
	todos rand

List all the todo tags in the current directory. A tag
is defined and either "TODO:" or "TODO(somename):"
	todos tags

Remove all todos:
	todos clear

Flags:
  -desc string
    	description of the todo
  -path string
    	path of the todo
```

### Binaries

TODO

