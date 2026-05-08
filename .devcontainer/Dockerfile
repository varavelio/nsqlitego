FROM alpine:3.23.3 AS fetcher

WORKDIR /fetcher

RUN \
  apk add --quiet curl unzip && \
  # Fetch VDL
  curl -fsSL https://get.varavel.com/vdl | VERSION=v0.5.0-alpha.5 INSTALL_DIR=/fetcher QUIET=true sh && \
  chmod +x ./vdl && \
  # Fetch dprint
  curl -fsSL https://dprint.dev/install.sh | sh && \
  mv /root/.dprint/bin/dprint ./dprint && chmod +x ./dprint && \
  # Fetch task
  sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d -b /fetcher v3.50.0 && \
  chmod +x ./task

# Use golang as the base image
FROM golang:1.26-trixie

# Install golangci-lint
COPY --from=golangci/golangci-lint:v2.12.1 /usr/bin/golangci-lint /usr/local/bin/golangci-lint

# Install tools from the fetcher stage
COPY --from=fetcher /fetcher/vdl /usr/local/bin/vdl
COPY --from=fetcher /fetcher/dprint /usr/local/bin/dprint
COPY --from=fetcher /fetcher/task /usr/local/bin/task

# Set environment variables
ENV DEBIAN_FRONTEND="noninteractive"

RUN set -e && \
  # Update and install system dependencies
  apt-get update -qq && \
  apt-get install -yqq --no-install-recommends \
  ca-certificates wget curl zip unzip p7zip-full tzdata git tree ripgrep jq && \
  # Final setup: git config, and cleanup
  git config --global --add safe.directory '*' && \
  rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Add the startup script on every bash session
RUN rm -f /root/.bashrc
COPY .devcontainer/.bashrc /root/.bashrc

# Command just to keep the container running
CMD ["sleep", "infinity"]
