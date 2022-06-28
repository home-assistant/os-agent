package yellow

import (
	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"

	logging "github.com/home-assistant/os-agent/utils/log"
)

const (
	objectPath = "/io/hass/os/Boards/Yellow"
	ifaceName  = "io.hass.os.Boards.Yellow"
)

type yellow struct {
	conn *dbus.Conn
}

func getStatusLED() bool {
	return false
}

func setStatusLED(c *prop.Change) *dbus.Error {
	logging.Info.Printf("Set Yellow Status LED to %t", c.Value)

	return nil
}

func InitializeDBus(conn *dbus.Conn) {
	d := yellow{
		conn: conn,
	}

	propsSpec := map[string]map[string]*prop.Prop{
		ifaceName: {
			"StatusLED": {
				Value:    getStatusLED(),
				Writable: true,
				Emit:     prop.EmitTrue,
				Callback: setStatusLED,
			},
		},
	}

	props, err := prop.Export(conn, objectPath, propsSpec)
	if err != nil {
		logging.Critical.Panic(err)
	}

	err = conn.Export(d, objectPath, ifaceName)
	if err != nil {
		logging.Critical.Panic(err)
	}

	node := &introspect.Node{
		Name: objectPath,
		Interfaces: []introspect.Interface{
			introspect.IntrospectData,
			prop.IntrospectData,
			{
				Name:       ifaceName,
				Methods:    introspect.Methods(d),
				Properties: props.Introspection(ifaceName),
			},
		},
	}

	err = conn.Export(introspect.NewIntrospectable(node), objectPath, "org.freedesktop.DBus.Introspectable")
	if err != nil {
		logging.Critical.Panic(err)
	}

	logging.Info.Printf("Exposing object %s with interface %s ...", objectPath, ifaceName)
}
