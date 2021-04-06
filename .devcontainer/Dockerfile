# [Choice] Go version: 1, 1.15, 1.14
ARG VARIANT=1
FROM mcr.microsoft.com/vscode/devcontainers/go:${VARIANT}

# [Option] Install Node.js
ARG INSTALL_NODE="true"
ARG NODE_VERSION="lts/*"
RUN if [ "${INSTALL_NODE}" = "true" ]; then su vscode -c "source /usr/local/share/nvm/nvm.sh && nvm install ${NODE_VERSION} 2>&1"; fi

# [Optional] Uncomment this section to install additional OS packages.
RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
    && apt-get -y install --no-install-recommends \
        dbus \
        libglib2.0-bin \
    && rm -rf /var/lib/apt/lists/*


# [Optional] Uncomment the next line to use go get to install anything else you need
RUN go get -x github.com/coreos/go-systemd/daemon
RUN go get -x github.com/fntlnz/mountinfo
RUN go get -x github.com/godbus/dbus

# [Optional] Uncomment this line to install global node packages.
# RUN su vscode -c "source /usr/local/share/nvm/nvm.sh && npm install -g <your-package-here>" 2>&1
