# Agent for Home Assistant OS

## Compile

```
go build -ldflags "-X main.version="
```


## Tests

```sh
$ gdbus introspect --system --dest io.hass.os --object-path /io/hass/os
```
