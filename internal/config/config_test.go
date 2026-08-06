package config

import (
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, contents string) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())

	dir, err := Dir()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, fileName), []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadMissingFileIsNotAnError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	cfg, err := Load()
	if err != nil {
		t.Fatalf("missing config should not error: %v", err)
	}
	if cfg != Default() {
		t.Errorf("got %+v, want defaults %+v", cfg, Default())
	}
}

// Fields absent from the file must keep their default rather than becoming "".
func TestLoadPartialFileKeepsDefaults(t *testing.T) {
	write(t, `{"log_level":"debug"}`)

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want debug", cfg.LogLevel)
	}
	if cfg.Image != DefaultImage {
		t.Errorf("Image = %q, want default %q", cfg.Image, DefaultImage)
	}
}

func TestLoadOverridesImage(t *testing.T) {
	write(t, `{"image":"custom:tag"}`)

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Image != "custom:tag" {
		t.Errorf("Image = %q, want custom:tag", cfg.Image)
	}
}

// A typo in the config file must not leave the caller with a zero-valued image.
func TestLoadMalformedReturnsDefaultsAndError(t *testing.T) {
	write(t, `{"image": }`)

	cfg, err := Load()
	if err == nil {
		t.Fatal("malformed config should report an error")
	}
	if cfg != Default() {
		t.Errorf("got %+v, want defaults %+v", cfg, Default())
	}
}

func TestLoadDockerfileMissingIsEmpty(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	got, err := LoadDockerfile()
	if err != nil {
		t.Fatalf("missing Dockerfile should not error: %v", err)
	}
	if got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestLoadDockerfileReadsCustomFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dir, err := Dir()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	const want = "FROM alpine\n"
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte(want), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := LoadDockerfile()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
