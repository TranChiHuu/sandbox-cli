# Sandbox CLI

## Vision

Sandbox CLI is a secure runtime for AI coding agents.

Instead of running Claude Code, Codex CLI, Gemini CLI, Cursor, etc. directly on the host machine, Sandbox CLI launches them inside an isolated container while exposing only the current project directory.

The goal is to provide a nearly identical developer experience while preventing AI agents from accessing the developer's credentials, SSH keys, cloud configuration, browser data, or other personal files.

---

# Core Principles

1. Mount the project, not the developer's identity.
2. AI should not know it is inside a sandbox.
3. Existing AI CLIs should work without modification.
4. Security should be enforced by the runtime, not by prompts.
5. The runtime should be provider-agnostic.

---

# MVP Scope

Build a Go CLI named `sandbox`.

Example:

```bash
sandbox claude
sandbox codex
sandbox gemini
```

The CLI should:

- Detect the current project directory.
- Start or reuse a Docker container.
- Mount only the current project to `/workspace`.
- Set the working directory to `/workspace`.
- Launch the requested AI CLI inside the container.
- Attach stdin/stdout/stderr so the experience feels native.
- Remove the container when the session ends (configurable later).

---

# Container Requirements

The container should contain:

- Git
- Node.js
- npm
- pnpm
- Go
- Python
- bash
- curl
- Claude Code CLI
- Codex CLI
- Gemini CLI (optional initially)

The container should behave like a normal Linux development environment.

---

# Filesystem

Allowed:

/workspace

Blocked:

~/.aws
~/.ssh
~/.config
~/Documents
~/Downloads
Entire HOME directory

HOME inside the container should be isolated.

Example:

HOME=/tmp/home

---

# Architecture

sandbox CLI

↓

Sandbox Manager

↓

Docker

↓

AI CLI

The CLI should separate responsibilities into packages.

Example:

cmd/
internal/cli/
internal/docker/
internal/runtime/
internal/config/
internal/logger/

Keep the code modular so Docker can later be replaced with Firecracker or Kubernetes.

---

# CLI UX

Examples

Run Claude

sandbox claude

Run Codex

sandbox codex

Show version

sandbox version

Show help

sandbox --help

---

# Docker Requirements

If the sandbox image does not exist:

Automatically build it.

If a container is already running:

Reuse it.

Otherwise:

Create a new container.

---

# Configuration

Store configuration under:

~/.sandbox/

Future configuration:

- Docker image name
- Container name
- Runtime backend
- Cleanup policy
- Log level

Do not implement every option yet.

---

# Logging

Use structured logging.

Debug mode should print:

- Docker commands
- Container ID
- Mounted directory
- Runtime information

---

# Future Features (Not Yet)

Do not implement these now.

Authentication Broker

Command Broker

Policy Engine

Audit Logs

Firecracker backend

Kubernetes backend

Approval prompts

Network isolation

Credential proxy

SSH proxy

Git proxy

AWS proxy

---

# Coding Style

Use idiomatic Go.

Avoid unnecessary abstractions.

Keep interfaces small.

Separate business logic from Docker logic.

Every package should have a single responsibility.

Prefer composition over inheritance.

Avoid global state.

Return errors instead of panicking.

Write clean, maintainable code.

---

# Deliverables

The project should include:

- Go module
- Cobra CLI
- Docker integration
- Dockerfile for sandbox image
- Automatic image build
- Automatic container creation
- Interactive terminal support
- Graceful shutdown
- README
- Makefile
- Basic unit tests

The implementation should prioritize simplicity and maintainability over advanced features.