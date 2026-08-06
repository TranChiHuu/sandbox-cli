package docker

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestRunArgs(t *testing.T) {
	got := runArgs("img:1", "/p/proj", "sandbox-proj-7", true, []string{"claude", "--resume"})
	want := []string{
		"run", "--rm", "-i", "--init",
		"--name", "sandbox-proj-7",
		"-v", "/p/proj:/workspace",
		"-v", "sandbox-cli-home:/home/node",
		"-w", "/workspace",
		"-t",
		"img:1", "claude", "--resume",
	}
	if !slices.Equal(got, want) {
		t.Errorf("runArgs mismatch\n got: %q\nwant: %q", got, want)
	}
}

// Passing -t when stdin is not a terminal makes docker refuse to start at all.
func TestRunArgsOmitsTTYWhenNotATerminal(t *testing.T) {
	got := runArgs("img:1", "/p/proj", "n", false, []string{"claude"})
	if slices.Contains(got, "-t") {
		t.Errorf("got -t without a terminal: %q", got)
	}
	if got[len(got)-1] != "claude" || got[len(got)-2] != "img:1" {
		t.Errorf("agent argv must follow the image: %q", got)
	}
}

// The whole point of the tool: only the project and the throwaway HOME volume
// are ever mounted.
func TestRunArgsMountsNothingElse(t *testing.T) {
	got := runArgs("img:1", "/p/proj", "n", false, nil)

	var mounts []string
	for i, a := range got {
		if a == "-v" || a == "--mount" || a == "--volume" {
			mounts = append(mounts, got[i+1])
		}
	}
	want := []string{"/p/proj:/workspace", "sandbox-cli-home:/home/node"}
	if !slices.Equal(mounts, want) {
		t.Errorf("unexpected mounts\n got: %q\nwant: %q", mounts, want)
	}
}

func TestContainerName(t *testing.T) {
	for _, tc := range []struct {
		project, want string
	}{
		{"/p/proj", "sandbox-proj-7"},
		{"/p/My Project (v2)", "sandbox-My-Project-v2-7"},
		{"/p/...", "sandbox-...-7"},
	} {
		if got := containerName(tc.project, 7); got != tc.want {
			t.Errorf("containerName(%q) = %q, want %q", tc.project, got, tc.want)
		}
	}
}

// docker rejects names outside [a-zA-Z0-9][a-zA-Z0-9_.-]*, so a project
// directory with spaces or unicode must not produce an unusable name.
func TestContainerNameIsDockerSafe(t *testing.T) {
	got := containerName("/p/héllo wörld!", 7)
	for _, r := range got {
		if !strings.ContainsRune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_.-", r) {
			t.Fatalf("containerName produced illegal rune %q in %q", r, got)
		}
	}
}

func TestValidateProjectRejectsHomeAndRoot(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home directory")
	}
	for _, project := range []string{"/", home, home + string(filepath.Separator)} {
		if err := validateProject(project); err == nil {
			t.Errorf("validateProject(%q) allowed a mount that exposes the host", project)
		}
	}
}

func TestValidateProjectAllowsSubdirectory(t *testing.T) {
	if err := validateProject(t.TempDir()); err != nil {
		t.Errorf("validateProject rejected a normal project: %v", err)
	}
}

func TestNewUsesEmbeddedDockerfileByDefault(t *testing.T) {
	if got := New("img", "", nil).dockerfile; got != embeddedDockerfile {
		t.Error("empty dockerfile should fall back to the embedded one")
	}
	if !strings.Contains(embeddedDockerfile, "FROM node:") {
		t.Error("embedded Dockerfile does not look like a Dockerfile")
	}
}

// A user who writes ~/.sandbox/Dockerfile must actually get their image, not ours.
func TestNewHonoursCustomDockerfile(t *testing.T) {
	const custom = "FROM alpine\nRUN apk add --no-cache go\n"
	if got := New("img", custom, nil).dockerfile; got != custom {
		t.Errorf("dockerfile = %q, want the custom one", got)
	}
}
