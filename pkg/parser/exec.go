package parser

import (
	"io"

	"github.com/amenzhinsky/go-memexec"
	"github.com/stackb/scala-gazelle/pkg/procutil"
)

// ExecJS runs the embedded js runtime interpreter
func ExecJS(dir string, args, env []string, in io.Reader, stdout, stderr io.Writer) (int, error) {
	exe, err := memexec.New(nodeExe)
	if err != nil {
		return 1, err
	}
	defer exe.Close()

	cmd := exe.Command(args...)
	cmd.Stdin = in
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Env = env
	cmd.Dir = dir
	err = cmd.Run()

	return procutil.CmdExitCode(cmd, err), err
}
