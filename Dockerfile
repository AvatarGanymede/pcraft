# pcraft Server Dockerfile
#
# Single-stage build that consumes prebuilt artifacts. The release workflow
# (.github/workflows/release.yml) extracts the per-arch release bundle
# (`pcraft-linux-{x64,arm64}.tar.gz`) into the build context, then this file
# just COPYs the native binaries into the runtime layout.
#
# Building this file outside CI (manual `docker build .`) will fail because
# the `bundle/` directory isn't present in the build context. To build
# locally, extract a release tarball into ./ctx/bundle/ alongside
# ./ctx/docker-entrypoint.sh and run:
#   docker build -f Dockerfile ./ctx
#
# Run:
#   docker run -p 38429:38429 -v pcraft-data:/data ghcr.io/avatarganymede/pcraft:latest

FROM debian:bookworm-slim

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        git \
        gh \
        ca-certificates \
        curl \
        gosu \
        tini \
        python3 \
        python3-venv \
        pipx && \
    curl -sL https://aka.ms/InstallAzureCLIDeb | bash && \
    rm -rf /var/lib/apt/lists/* && \
    PIPX_HOME=/opt/pipx PIPX_BIN_DIR=/usr/local/bin pipx install apprise

RUN groupadd -r pcraft && useradd -r -g pcraft -u 1000 -d /data/home -M pcraft

RUN mkdir -p /data/home/.azure && \
    AZURE_CONFIG_DIR=/data/home/.azure az extension add --name azure-devops && \
    chown -R pcraft:pcraft /data/home/.azure

RUN mkdir -p /app/apps/backend/bin /data/worktrees

COPY bundle/bin/pcraft               /app/apps/backend/bin/pcraft
COPY bundle/bin/agentctl-linux-amd64 /app/apps/backend/bin/agentctl-linux-amd64
COPY bundle/bin/agentctl             /usr/local/bin/agentctl
COPY docker-entrypoint.sh            /usr/local/bin/docker-entrypoint.sh

RUN chmod +x \
        /app/apps/backend/bin/pcraft \
        /app/apps/backend/bin/agentctl-linux-amd64 \
        /usr/local/bin/agentctl \
        /usr/local/bin/docker-entrypoint.sh && \
    ln -s /app/apps/backend/bin/pcraft /usr/local/bin/pcraft && \
    chown -R pcraft:pcraft /app /data

VOLUME ["/data"]

ENV PCRAFT_NO_BROWSER=1 \
    PCRAFT_HOME_DIR=/data \
    HOME=/data/home \
    NPM_CONFIG_PREFIX=/data/.npm-global \
    PATH=/data/.npm-global/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin \
    HOSTNAME=0.0.0.0 \
    NODE_ENV=production

WORKDIR /app

EXPOSE 38429

ENTRYPOINT ["tini", "--", "docker-entrypoint.sh"]
CMD ["pcraft", "start", "--backend-port", "38429", "--verbose"]
