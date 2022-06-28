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
	conn   *dbus.Conn
	optLED bool
}

func getStatusLED() bool {
	// FIXME: read current LED state out of sysfs
	return false
}

func setStatusLED(c *prop.Change) *dbus.Error {
	logging.Info.Printf("Set Yellow Status LED to %t", c.Value)
	// FIXME: set new LED state out of sysfs
	//optLED = c.Value.(bool)
	return nil
}

func InitializeDBus(conn *dbus.Conn) {
	d := yellow{
		conn:   conn,
		optLED: getStatusLED(),
	}

	propsSpec := map[string]map[string]*prop.Prop{
		ifaceName: {
			"StatusLED": {
				Value:    d.optLED,
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
