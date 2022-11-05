package progress

import (
	"io"
)

func NewProgressOutput(out io.Writer) Output {
	return &progressOutput{sf: &rawProgressFormatter{}, out: out, newLines: true}
}

type progressOutput struct {
	sf       formatProgress
	out      io.Writer
	newLines bool
}

// WriteProgress formats progress information from a ProgressReader.
func (out *progressOutput) WriteProgress(prog Progress) error {
	var formatted []byte
	if prog.Message != "" {
		formatted = out.sf.formatStatus(prog.ID, prog.Message)
	} else {
		jsonProgress := JSONProgress{Current: prog.Current, Total: prog.Total, HideCounts: prog.HideCounts, Units: prog.Units}
		formatted = out.sf.formatProgress(prog.ID, prog.Action, &jsonProgress, prog.Aux)
	}
	_, err := out.out.Write(formatted)
	if err != nil {
		return err
	}

	if out.newLines && prog.LastUpdate {
		_, err = out.out.Write(out.sf.formatStatus("", ""))
		return err
	}

	return nil
}
