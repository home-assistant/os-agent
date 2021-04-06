package main

import (
	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/prop"
	logging "github.com/home-assistant/os-agent/utils/log"
)

func base_properties(conn *dbus.Conn) {
	propsSpec := map[string]map[string]*prop.Prop{
		busName: {
			"Version": {
				version,
				false,
				prop.EmitInvalidates,
				nil,
			},
		},
	}

	_, err := prop.Export(conn, objectPath, propsSpec)
	if err != nil {
		logging.Critical.Panic(err)
	}
}
