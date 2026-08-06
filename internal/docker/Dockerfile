FROM node:26-alpine

RUN apk add --no-cache \
      bash git curl ca-certificates less ripgrep \
      python3 py3-pip \
      build-base

# pnpm via npm, not corepack: Node 26 no longer bundles corepack.
RUN npm install -g \
      pnpm \
      @anthropic-ai/claude-code \
      @openai/codex \
      @google/gemini-cli \
    && npm cache clean --force

# Isolated HOME: nothing from the host home is ever mounted here.
ENV HOME=/home/node
ENV PATH=/home/node/.local/bin:$PATH

RUN mkdir -p /home/node /workspace && chown -R node:node /home/node /workspace

# ponytail: fixed uid 1000 (node) matches the common single-user Linux host so
# files written to /workspace stay owned by the developer. Add --user passthrough
# if someone runs on a host with a different uid.
USER node
WORKDIR /workspace
