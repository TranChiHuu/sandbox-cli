package cli

import (
	"bytes"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestVersionCommand(t *testing.T) {
	var out bytes.Buffer
	root := NewRoot()
	root.SetOut(&out)
	root.SetArgs([]string{"version"})

	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(out.String()); got != Version {
		t.Errorf("version = %q, want %q", got, Version)
	}
}

func TestRequiresAnAgent(t *testing.T) {
	root := NewRoot()
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs(nil)

	if err := root.Execute(); err == nil {
		t.Error("running with no agent should be an error, not a docker session")
	}
}

// Flags meant for the agent must reach the agent. Without
// SetInterspersed(false), cobra treats --resume as an unknown sandbox flag.
func TestAgentFlagsArePassedThrough(t *testing.T) {
	for _, tc := range []struct {
		name  string
		argv  []string
		want  []string
		debug bool
	}{
		{"agent flags", []string{"claude", "--resume", "-p", "hi"}, []string{"claude", "--resume", "-p", "hi"}, false},
		{"sandbox flag first", []string{"--debug", "codex", "--full-auto"}, []string{"codex", "--full-auto"}, true},
		{"agent flag shadows ours", []string{"claude", "--debug"}, []string{"claude", "--debug"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got []string
			root := NewRoot()
			// Replace the runner so the test never launches docker; the flag
			// parsing under test happens before RunE either way.
			root.RunE = func(_ *cobra.Command, args []string) error {
				got = args
				return nil
			}
			root.SetOut(&bytes.Buffer{})
			root.SetArgs(tc.argv)

			if err := root.Execute(); err != nil {
				t.Fatalf("Execute(%q): %v", tc.argv, err)
			}
			if !slices.Equal(got, tc.want) {
				t.Errorf("args = %q, want %q", got, tc.want)
			}
			if debug, _ := root.Flags().GetBool("debug"); debug != tc.debug {
				t.Errorf("debug = %v, want %v", debug, tc.debug)
			}
		})
	}
}
