# Contributing

## Setup

Go 1.23+ and Docker. Then:

```bash
make build      # ./bin/sandbox
make test
make image      # rebuild the sandbox image
```

## Before opening a pull request

- `make test` passes and `gofmt -l .` prints nothing
- New behaviour has a test in the same package
- The change works against a real container, not just in unit tests — `./bin/sandbox --debug bash -lc 'echo ok'`

## Scope

This tool isolates your *identity* from an agent, and does it with one Docker
backend. Things deliberately left out are listed under **Not implemented** in the
[README](README.md) — network isolation, credential proxying, approval prompts,
policy engines, alternative backends. A PR adding one of those needs to argue for
the complexity first, so open an issue before writing the code.

Smaller changes — a runtime in the Dockerfile, a clearer error message, a bug fix,
docs — just send the PR.

## Commits

One logical change per commit. Subject line in the imperative, under 72
characters, prefixed with the area when it helps: `docker: retry engine start`.
