package main

func runGazelle(wd string, args []string) error {
	return run(wd, args)
}

// TestHelp checks that help commands do not panic due to nil flag values.
// Verifies #256.
func TestHelp(t *testing.T) {
	for _, args := range [][]string{
		{"help"},
		{"fix", "-h"},
		{"update", "-h"},
		{"update-repos", "-h"},
	} {
		t.Run(args[0], func(t *testing.T) {
			if err := runGazelle(".", args); err == nil {
				t.Errorf("%s: got success, want flag.ErrHelp", args[0])
			} else if err != flag.ErrHelp {
				t.Errorf("%s: got %v, want flag.ErrHelp", args[0], err)
			}
		})
	}
}
