package starlarkeval

import (
	"bytes"
	"fmt"
	"io"

	"go.starlark.net/starlark"
)

type Interpreter struct {
	// Global state
	globals *starlark.StringDict
	// Thread context
	thread *starlark.Thread
	// Last eval error
	evalErr *starlark.EvalError
	// reporter
	reporter Reporter
}

// Reporter is implemented by *testing.T.
type Reporter func(format string, args ...interface{})

func NewInterpreter(reporter Reporter) *Interpreter {
	interpreter := &Interpreter{
		reporter: reporter,
		thread: &starlark.Thread{
			Print: func(_ *starlark.Thread, msg string) {
				reporter(msg)
			},
		},
	}

	interpreter.globals = &starlark.StringDict{}

	return interpreter
}

func (i *Interpreter) GetGlobal(name string) starlark.Value {
	globals := *i.globals
	return globals[name]
}

func (i *Interpreter) handleDepset(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	// always expect a single list argument for depsets
	if len(args) != 1 {
		return nil, fmt.Errorf("depset() expected a single positional argument of type List")
	}
	return args[0], nil
}

func (i *Interpreter) Exec(filename string, src io.Reader) error {
	data, err := io.ReadAll(src)
	if err != nil {
		return err
	}
	state, err := starlark.ExecFile(i.thread, filename, bytes.NewReader(data), *i.globals)
	i.globals = &state
	if evalErr, ok := err.(*starlark.EvalError); ok {
		fmt.Printf("EvalErr: %v\n", evalErr)
		i.evalErr = evalErr
	}
	return err
}
