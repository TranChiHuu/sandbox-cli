// Package cli wires the command surface to the sandbox runtime.
package cli

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/spf13/cobra"

	"github.com/tranchihuu/sandbox-cli/internal/config"
	"github.com/tranchihuu/sandbox-cli/internal/docker"
	"github.com/tranchihuu/sandbox-cli/internal/logger"
)

// Version is overridden at build time via -ldflags.
var Version = "dev"

// NewRoot builds the sandbox command tree.
func NewRoot() *cobra.Command {
	var debug bool

	root := &cobra.Command{
		Use:   "sandbox <agent> [args...]",
		Short: "Run AI coding agents in an isolated container",
		Long: "sandbox runs an AI coding CLI inside a container that sees only the\n" +
			"current directory, mounted at /workspace. The host home directory —\n" +
			"credentials, SSH keys, cloud config — is never exposed.",
		Example: "  sandbox claude\n  sandbox codex\n  sandbox gemini\n  sandbox bash",
		Args:    cobra.MinimumNArgs(1),
		RunE:    func(cmd *cobra.Command, args []string) error { return run(cmd, args, debug) },
		// main prints errors itself; run() silences usage once args are known good.
		SilenceErrors: true,
	}
	root.Flags().SetInterspersed(false)
	root.Flags().BoolVar(&debug, "debug", false, "log docker commands, container name, and mounts")

	// Redirect into ~/.sandbox/Dockerfile to add a language runtime the default omits.
	root.AddCommand(&cobra.Command{
		Use:   "dockerfile",
		Short: "Print the Dockerfile used to build the sandbox image",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, _ []string) {
			fmt.Fprint(cmd.OutOrStdout(), docker.DefaultDockerfile())
		},
	})

	root.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print the sandbox version",
		Args:  cobra.NoArgs,
		Run:   func(cmd *cobra.Command, _ []string) { fmt.Fprintln(cmd.OutOrStdout(), Version) },
	})
	return root
}

func run(cmd *cobra.Command, args []string, debug bool) error {
	cmd.SilenceUsage = true // past arg validation: failures from here aren't usage errors
	ctx := cmd.Context()

	// A broken config file must not block a session: warn and use defaults.
	cfg, cfgErr := config.Load()
	log := logger.New(cfg.LogLevel, debug)
	if cfgErr != nil {
		log.Warn("using default configuration", "error", cfgErr)
	}
	log.Debug("config", "image", cfg.Image, "log_level", cfg.LogLevel)

	// A custom Dockerfile is how users add language runtimes the default image omits.
	// Failing to read one they wrote must not silently fall back to the embedded one.
	custom, err := config.LoadDockerfile()
	if err != nil {
		return err
	}
	if custom != "" {
		log.Debug("using custom Dockerfile from ~/.sandbox")
	}

	sb := docker.New(cfg.Image, custom, log)
	if err := sb.Available(ctx); err != nil {
		return err
	}

	// ponytail: the project is the cwd, not the enclosing git root. Walk up to
	// .git if mounting a subdirectory of a repo turns out to be the common case.
	project, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("detect project directory: %w", err)
	}

	if !sb.ImageExists(ctx) {
		fmt.Fprintf(os.Stderr, "sandbox: building %s (first run only)\n", sb.Image())
		if err := sb.BuildImage(ctx); err != nil {
			return err
		}
	}

	// The container shares our terminal, so Ctrl-C reaches the agent directly.
	// Ignore it here so we stay alive to reap the container instead of orphaning it.
	signal.Notify(make(chan os.Signal, 1), os.Interrupt)

	code, err := sb.Run(ctx, project, args)
	if err != nil {
		return err
	}
	log.Debug("session ended", "exit_code", code)
	if code != 0 {
		os.Exit(code)
	}
	return nil
}
