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

Download the latest Debian package from OS Agent GitHub release page at:

<https://github.com/home-assistant/os-agent/releases/latest>

Next, install (or update) the downloaded Debian package using:

```shell
sudo dpkg -i os-agent_1.0.0_linux_x86_64.deb
```

Note: Replace the `deb` file in the above example with the file you
have downloaded from the releases page.

You can test if the installation was successful by running:

```bash
busctl introspect --system io.hass.os /io/hass/os
```

This should **not** return an error. If you get an object introspection
with `io.hass.os`, `interface` etc. OS Agent is working as expected.

## Uninstall

To remove OS Agent from your system use the Debian packaging system:

```shell
sudo dpkg -r os-agent
```

## Development

### Compile

```shell
go build -ldflags "-X main.version="
```

### Tests

```shell
gdbus introspect --system --dest io.hass.os --object-path /io/hass/os
gdbus call --system --dest io.hass.os --object-path /io/hass/os/Boards/Yellow --method org.freedesktop.DBus.Properties.Set io.hass.os.Boards.Yellow PowerLED "<false>"
```
