# Agent for Home Assistant OS

This is the OS Agent for Home Assistant. It is used for Home Assistant
OS and Home Assistant Supervised installation types and it allows the
Home Assistant Supervisor to communicate with the host operating system.

## Installation & Update

### Using the Home Assistant Operating System

The OS Agent is pre-installed with the Home Assistant Operating System.

Updates are part of the Home Assistant Operating System updates, which
the Home Assistant UI will offer to upgrade to when there is a new version
available.

### Using Home Assistant Supervised on Debian

1. Install the required dependancy's using the following command:

```bash
sudo apt install \
udisks2 \
libglib2.0-bin -y
```

2. Download the and install (or update) the Debian package using the following commands:

```bash
wget https://github.com/home-assistant/os-agent/releases/latest/download/os-agent_1.0.0_linux_x86_64.deb
sudo dpkg -i os-agent_1.0.0_linux_x86_64.deb
```

Note: Replace the `deb` file in the above example with the latest
version found on [Releases Page](https://github.com/home-assistant/os-agent/releases/latest/).

3. You can test if the installation was successful by running:

```bash
gdbus introspect --system --dest io.hass.os --object-path /io/hass/os
```

## Development

### Compile

```shell
go build -ldflags "-X main.version="
```

### Tests

```shell
gdbus introspect --system --dest io.hass.os --object-path /io/hass/os
```
