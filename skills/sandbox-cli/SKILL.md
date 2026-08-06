---
name: sandbox-cli
description: Use when an agent needs to run untrusted or unreviewed code, install dependencies, or execute a project's tests without exposing the host — installs and drives the `sandbox` CLI, which runs Claude Code, Codex, or Gemini CLI in a Docker container where only the current directory is mounted.
---

# sandbox-cli

Runs an agent (or any command) in a container that mounts the current directory and nothing else. The host home directory — SSH keys, cloud credentials, browser data — is absent, not blocked.

## When to use

- Running a project's tests, build, or `npm install`/`composer install` from unreviewed code
- Handing a repo to Claude Code / Codex / Gemini CLI without host access
- A language runtime the host lacks (PHP, Go, Ruby) — add it to the image instead of the host

Not for: network isolation (network is unrestricted), or hiding secrets that live inside the project (`/workspace` is fully readable).

## Install

```bash
command -v sandbox && sandbox version    # already installed?

git clone https://github.com/TranChiHuu/sandbox-cli.git ~/.local/src/sandbox-cli
make -C ~/.local/src/sandbox-cli install     # go install → $(go env GOPATH)/bin
export PATH="$(go env GOPATH)/bin:$PATH"     # add to shell rc if missing
```

Requires Go 1.23+ and Docker (Docker Desktop, OrbStack, Rancher Desktop). If the engine is down, `sandbox` starts it on macOS; on Linux it prints the `systemctl` command.

## Use

```bash
cd /path/to/project
sandbox claude            # or: codex, gemini, bash, npm, pytest, anything in the image
sandbox claude --resume    # args pass straight through
sandbox bash -lc 'npm ci && npm test'
sandbox --debug claude     # print the docker argv, mounts, container name
```

First run builds the image (a few minutes). After that startup ≈ `docker run`.

## Quick reference

| | |
|---|---|
| `/workspace` | the current directory, read-write |
| `$HOME` (`/home/node`) | empty, isolated, persisted in volume `sandbox-cli-home` |
| host home | never mounted |
| `sandbox dockerfile` | print the image's Dockerfile, to copy and edit |
| `docker volume rm sandbox-cli-home` | reset agent logins/caches |

Refuses to run from `$HOME` or `/` — mounting either defeats the sandbox.

Base image: Alpine + bash, git, curl, less, ripgrep, Node 26, npm, pnpm, Python 3, a C toolchain, and the three agent CLIs. **No per-project language runtimes**, and the container user is non-root so the agent cannot `apk add` one.

## Adding a language runtime

**Detect what the project actually uses — never assume a language.** Check the manifest in the project root:

| Found in project | Runtime | Alpine packages |
|---|---|---|
| `package.json`, `pnpm-lock.yaml` | Node | already in the base image |
| `requirements.txt`, `pyproject.toml` | Python | already in the base image |
| `composer.json` | PHP | see the full example below |
| `go.mod` | Go | `go` |
| `Gemfile` | Ruby | `ruby ruby-dev ruby-bundler` |
| `pom.xml` | Java | `openjdk21 maven` |
| `build.gradle`, `build.gradle.kts` | Java | `openjdk21 gradle` |
| `Cargo.toml` | Rust | `rust cargo` |
| `*.csproj`, `*.sln` | .NET | `dotnet9-sdk` |

A polyglot repo needs every runtime its tests touch. Add `mysql-client`/`postgresql-client` only if the project shells out to `mysql`/`psql`. Then build the image once:

```bash
sandbox dockerfile > ~/.sandbox/Dockerfile        # start from the default
# edit: add packages to an `apk add` line
echo '{"image":"sandbox-php:latest"}' > ~/.sandbox/config.json
sandbox claude                                    # builds and uses it
```

`~/.sandbox/Dockerfile` **replaces** the built-in one entirely — keep the `USER node` and `WORKDIR /workspace` lines, and give the image its own name in `config.json` so it doesn't collide with `sandbox-cli:latest`.

PHP example (Alpine 3.24; the `composer` package depends on `php85`, so match that version or composer drags a second PHP in):

```dockerfile
RUN apk add --no-cache \
      php85 php85-phar php85-mbstring php85-openssl php85-curl \
      php85-dom php85-xml php85-xmlreader php85-xmlwriter php85-tokenizer \
      php85-session php85-fileinfo php85-ctype php85-iconv php85-simplexml \
      php85-pcntl php85-posix php85-zip php85-bcmath php85-sodium \
      php85-pdo php85-pdo_sqlite php85-pdo_mysql php85-pdo_pgsql php85-sqlite3 \
      php85-gd php85-intl php85-pecl-xdebug \
      composer mysql-client \
    && ln -sf /usr/bin/php85 /usr/local/bin/php
```

## Common mistakes

| Symptom | Cause |
|---|---|
| `php`/`go`/`ruby` not found | base image ships no project runtimes — edit `~/.sandbox/Dockerfile` |
| Dockerfile edited but nothing changed | image already built; change `image` in `config.json` or `docker rmi` it |
| Dev server unreachable from the host browser | no port publishing — run the server on the host, or `docker run -p` manually |
| Database connection refused | no DB in the image; use sqlite, or reach the host at `host.docker.internal` |
| Files in the project owned by wrong uid | image is fixed to uid 1000; hosts with another uid need a `--user` passthrough |
