# Agent for Home Assistant OS

## Compile

```
go build -ldflags "-X main.version="
```


## Tests

```sh
# gdbus introspect --system --dest io.homeassistant.os --object-path /io/homeassistant/os
```
