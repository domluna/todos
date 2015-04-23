todos
=====

todos is a todo list for the terminal.

### Usage

Add todos with a description and path. If the -path
flag is not present, the current directory will be
used. T is the name of the todo:
```sh
todos -desc="write more tests" add T
```

Remove a todo with a name T from the list:
```sh
todos rm T
```

List all todos:
```sh
todos list
```

Show a random todo:
```sh
todos rand
```

List all the todo tags in the current directory. A tag
is defined and either "TODO:" or "TODO(somename):"
```sh
todos tags
```

Remove all todos:
```sh
todos clear
```

### Binaries

TODO

