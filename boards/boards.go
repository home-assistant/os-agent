package boards

import (
	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"

	"github.com/home-assistant/os-agent/boards/supervised"
	"github.com/home-assistant/os-agent/boards/yellow"
	logging "github.com/home-assistant/os-agent/utils/log"
)

const (
	objectPath = "/io/hass/os/Boards"
	ifaceName  = "io.hass.os.Boards"
)

type boards struct {
	conn  *dbus.Conn
	props *prop.Properties
}

func InitializeDBus(conn *dbus.Conn, board string) {
	d := boards{
		conn: conn,
	}

	propsSpec := map[string]map[string]*prop.Prop{
		ifaceName: {
			"Board": {
				Value:    board,
				Writable: false,
				Emit:     prop.EmitInvalidates,
				Callback: nil,
			},
		},
	}

	props, err := prop.Export(conn, objectPath, propsSpec)
	if err != nil {
		logging.Critical.Panic(err)
	}
	d.props = props

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

	// Initialize the board
	if board == "yellow" {
		yellow.InitializeDBus(conn)
	} else if board == "supervised" {
		supervised.InitializeDBus(conn)
	} else {
		logging.Warning.Printf("Unknown board %s", board)
	}
}
