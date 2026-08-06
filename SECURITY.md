# Security Policy

## Supported versions

The latest commit on `main`. There are no maintained release branches.

## Reporting a vulnerability

Open a [private security advisory](https://github.com/TranChiHuu/sandbox-cli/security/advisories/new).
Please don't file a public issue for anything that lets code escape the container
or read the host home directory.

Include the `sandbox version` output, your Docker engine and OS, and the smallest
command that reproduces it. Expect a first reply within a week.

## What is in scope

`sandbox` exists to keep an agent from reading your host identity — SSH keys, cloud
credentials, browser data. In scope:

- Anything that exposes the host home directory or host filesystem inside the container
- Container escape through the arguments, config, or Dockerfile handling
- `sandbox` running from a directory it should refuse (`$HOME`, `/`)

## What is not

These are documented limits, not bugs:

- **Network access is unrestricted.** The container reaches the internet and anything
  your machine routes to, including `host.docker.internal`.
- **`/workspace` is fully readable.** A `.env` file in your project is visible to the
  agent by design — this isolates your identity, not your project's own secrets.
- **Docker itself is the boundary.** A kernel or Docker vulnerability belongs upstream.
- **The agent runs unsupervised.** There are no approval prompts or audit logs; an
  agent can do anything to the mounted directory, including deleting it.
