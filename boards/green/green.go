package green

import (
	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"

	"github.com/home-assistant/os-agent/utils/led"
	logging "github.com/home-assistant/os-agent/utils/log"
)

const (
	objectPath = "/io/hass/os/Boards/Green"
	ifaceName  = "io.hass.os.Boards.Green"
)

var (
	ledPower    led.LED = led.LED{Name: "power", DefaultTrigger: "default-on"}
	ledActivity led.LED = led.LED{Name: "activity", DefaultTrigger: "activity"}
	ledUser     led.LED = led.LED{Name: "user", DefaultTrigger: "heartbeat"}
)

type green struct {
	conn  *dbus.Conn
	props *prop.Properties
}

func getTriggerLED(led led.LED) bool {
	value, err := led.GetTrigger()
	if err != nil {
		logging.Error.Print(err)
	}
	return value != "none"
}

func setTriggerLED(led led.LED, c *prop.Change) *dbus.Error {
	logging.Info.Printf("Set Green %s LED to %t", led.Name, c.Value)
	err := led.SetTrigger(c.Value.(bool))
	if err != nil {
		return dbus.MakeFailedError(err)
	}
	return nil
}

func setPowerLED(c *prop.Change) *dbus.Error {
	return setTriggerLED(ledPower, c)
}

func setActivityLED(c *prop.Change) *dbus.Error {
	return setTriggerLED(ledActivity, c)
}

func setUserLED(c *prop.Change) *dbus.Error {
	return setTriggerLED(ledUser, c)
}

func InitializeDBus(conn *dbus.Conn) {
	d := green{
		conn: conn,
	}

	// Init base value
	ledPowerValue := getTriggerLED(ledPower)
	ledActivityValue := getTriggerLED(ledActivity)
	ledUserValue := getTriggerLED(ledUser)

	propsSpec := map[string]map[string]*prop.Prop{
		ifaceName: {
			"PowerLED": {
				Value:    ledPowerValue,
				Writable: true,
				Emit:     prop.EmitTrue,
				Callback: setPowerLED,
			},
			"ActivityLED": {
				Value:    ledActivityValue,
				Writable: true,
				Emit:     prop.EmitTrue,
				Callback: setActivityLED,
			},
			"UserLED": {
				Value:    ledUserValue,
				Writable: true,
				Emit:     prop.EmitTrue,
				Callback: setUserLED,
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
}
