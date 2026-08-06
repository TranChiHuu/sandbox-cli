// Package docker turns sandbox intent into docker CLI invocations.
//
// It shells out to the docker binary rather than using the Docker SDK: the SDK
// would add a large dependency tree and its own attach/TTY plumbing, while
// `docker run -it` already gives us interactive terminals for free.
package docker

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"golang.org/x/term"
)

const (
	// Workdir is the in-container mount point for the project.
	Workdir = "/workspace"
	// Home is the isolated in-container HOME.
	Home = "/home/node"
	// HomeVolume persists the isolated HOME (agent logins, caches) across runs
	// without exposing the host home directory.
	HomeVolume = "sandbox-cli-home"
)

//go:embed Dockerfile
var embeddedDockerfile string

// Sandbox runs commands in a container. It holds the settings that vary rather
// than reading globals, so callers stay in control of image and logging.
type Sandbox struct {
	image      string
	dockerfile string
	log        *slog.Logger
}

// New returns a Sandbox using the given image. An empty dockerfile means the
// embedded one; a nil logger discards output.
func New(image, dockerfile string, log *slog.Logger) *Sandbox {
	if dockerfile == "" {
		dockerfile = embeddedDockerfile
	}
	if log == nil {
		log = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return &Sandbox{image: image, dockerfile: dockerfile, log: log}
}

// Image is the image this sandbox runs.
func (s *Sandbox) Image() string { return s.image }

// DefaultDockerfile is the embedded Dockerfile, for users to copy and edit.
func DefaultDockerfile() string { return embeddedDockerfile }

// engineStartTimeout bounds how long we wait for a cold engine to accept
// connections. Docker Desktop and OrbStack both boot a VM, so this is slow.
const engineStartTimeout = 90 * time.Second

// Available makes docker usable: it reports a missing install, and starts the
// engine itself if it is installed but not running.
func (s *Sandbox) Available(ctx context.Context) error {
	path, err := exec.LookPath("docker")
	if err != nil {
		return errors.New("docker not found in PATH — install Docker Desktop or OrbStack: https://docs.docker.com/get-docker/")
	}
	s.log.Debug("docker cli", "path", path)

	if engineUp(ctx) {
		s.log.Debug("docker engine already running")
		return nil
	}
	app, err := startEngine(ctx)
	if err != nil {
		return err
	}
	s.log.Debug("started docker engine", "app", app)
	return waitForEngine(ctx)
}

func engineUp(ctx context.Context) bool {
	return exec.CommandContext(ctx, "docker", "info").Run() == nil
}

// startEngine launches the installed container engine. On macOS engines ship as
// app bundles we can open; on Linux the daemon is root-owned systemd territory,
// so we hand the user the command instead of trying to sudo behind their back.
// It returns the name of the app it started.
func startEngine(ctx context.Context) (string, error) {
	if runtime.GOOS != "darwin" {
		return "", errors.New("docker is installed but not running — start it with `sudo systemctl start docker`")
	}
	for _, app := range macEngineApps(ctx) {
		if _, err := os.Stat("/Applications/" + app + ".app"); err != nil {
			continue
		}
		if err := exec.CommandContext(ctx, "open", "-a", app).Run(); err == nil {
			fmt.Fprintf(os.Stderr, "sandbox: starting %s", app)
			return app, nil
		}
	}
	return "", errors.New("docker is installed but not running — start your container engine and retry")
}

// macEngineApps lists candidate engines, preferring the one named by the active
// docker context: booting the wrong VM costs the user a minute and a lot of RAM.
func macEngineApps(ctx context.Context) []string {
	apps := []string{"Docker", "OrbStack", "Rancher Desktop", "Podman Desktop"}
	out, err := exec.CommandContext(ctx, "docker", "context", "show").Output()
	if err != nil {
		return apps
	}
	byContext := map[string]string{
		"orbstack":        "OrbStack",
		"desktop-linux":   "Docker",
		"rancher-desktop": "Rancher Desktop",
	}
	if app, ok := byContext[strings.TrimSpace(string(out))]; ok {
		return append([]string{app}, apps...)
	}
	return apps
}

func waitForEngine(ctx context.Context) error {
	deadline := time.NewTimer(engineStartTimeout)
	defer deadline.Stop()
	tick := time.NewTicker(time.Second)
	defer tick.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			fmt.Fprintln(os.Stderr)
			return errors.New("timed out waiting for docker to start")
		case <-tick.C:
			if engineUp(ctx) {
				fmt.Fprintln(os.Stderr, " ready")
				return nil
			}
			fmt.Fprint(os.Stderr, ".")
		}
	}
}

// ImageExists reports whether the sandbox image is present locally.
func (s *Sandbox) ImageExists(ctx context.Context) bool {
	return exec.CommandContext(ctx, "docker", "image", "inspect", s.image).Run() == nil
}

// BuildImage builds the sandbox image from the embedded Dockerfile. The
// Dockerfile needs no build context, so it is piped in over stdin.
func (s *Sandbox) BuildImage(ctx context.Context) error {
	s.log.Debug("docker build", "image", s.image)

	cmd := exec.CommandContext(ctx, "docker", "build", "-t", s.image, "-")
	cmd.Stdin = strings.NewReader(s.dockerfile)
	cmd.Stdout = os.Stderr // build chatter must not pollute the agent's stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("build image %s: %w", s.image, err)
	}
	return nil
}

// runArgs builds the argv for `docker` that runs argv inside the sandbox with
// project mounted at Workdir. Split out from Run so it can be tested.
func runArgs(image, project, name string, tty bool, argv []string) []string {
	args := []string{
		"run", "--rm", "-i", "--init",
		"--name", name,
		"-v", project + ":" + Workdir,
		"-v", HomeVolume + ":" + Home,
		"-w", Workdir,
	}
	if tty {
		args = append(args, "-t")
	}
	return append(append(args, image), argv...)
}

// containerName derives a name that is readable in `docker ps` and unique per
// session, so concurrent sandboxes in the same project do not collide.
func containerName(project string, pid int) string {
	base := nonAlphanumeric.ReplaceAllString(filepath.Base(project), "-")
	return fmt.Sprintf("sandbox-%s-%d", strings.Trim(base, "-"), pid)
}

var nonAlphanumeric = regexp.MustCompile(`[^a-zA-Z0-9_.-]+`)

// validateProject refuses mounts that would defeat the sandbox. Mounting the
// home directory or the filesystem root would hand the agent the credentials
// and keys this tool exists to hide, so the runtime blocks it rather than
// trusting the caller to have chosen a sane directory.
func validateProject(project string) error {
	abs, err := filepath.Abs(project)
	if err != nil {
		return fmt.Errorf("resolve project directory %q: %w", project, err)
	}
	if abs == string(filepath.Separator) {
		return errors.New("refusing to mount the filesystem root as a project")
	}
	if home, err := os.UserHomeDir(); err == nil && abs == filepath.Clean(home) {
		return errors.New("refusing to mount your home directory — run sandbox from inside a project")
	}
	return nil
}

// Run executes argv inside the sandbox, wiring the current stdio through so the
// session feels native. It returns the child's exit code alongside any error.
func (s *Sandbox) Run(ctx context.Context, project string, argv []string) (int, error) {
	if err := validateProject(project); err != nil {
		return 1, err
	}
	tty := term.IsTerminal(int(os.Stdin.Fd()))
	name := containerName(project, os.Getpid())
	args := runArgs(s.image, project, name, tty, argv)

	s.log.Debug("runtime",
		"image", s.image, "mount", project+" -> "+Workdir,
		"home", Home, "home_volume", HomeVolume, "tty", tty)
	s.log.Debug("container", "name", name)
	s.log.Debug("docker command", "argv", append([]string{"docker"}, args...))

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr

	err := cmd.Run()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode(), nil // the agent failed, not the sandbox
	}
	if err != nil {
		return 1, fmt.Errorf("docker run: %w", err)
	}
	return 0, nil
}
