package apparmor

import (
	"fmt"
	"os/exec"
	"regexp"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"

	logging "github.com/home-assistant/os-agent/utils/log"
)

const (
	objectPath        = "/io/homeassistant/os/AppArmor"
	ifaceName         = "io.homeassistant.os.AppArmor"
	appArmorParserCmd = "apparmor_parser"
)

type apparmor struct {
	conn *dbus.Conn
}

func getAppArmorVersion() string {
	cmd := exec.Command(appArmorParserCmd, "--version")

	out, err := cmd.CombinedOutput()
	if err != nil {
		logging.Critical.Panic(err)
	}

	re := regexp.MustCompile("version ([0-9.]*)")
	found := re.FindSubmatch(out)
	if len(found) < 1 {
		logging.Error.Fatalln("Can't read version from parser!")
	}

	return string(found[1])
}

func (d apparmor) LoadProfile(profilePath string, cachePath string) (bool, *dbus.Error) {
	logging.Info.Printf("Load AppArmor profile '%s'.", profilePath)

	cmd := exec.Command(appArmorParserCmd, "--replace", "--write-cache", "--cache-loc", cachePath, profilePath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false, dbus.MakeFailedError(fmt.Errorf("Can't load profile '%s': %s", profilePath, err))
	}

	logging.Info.Printf("Load profile '%s': %s", profilePath, out)
	return true, nil
}

func (d apparmor) UnloadProfile(profilePath string, cachePath string) (bool, *dbus.Error) {
	logging.Info.Printf("Unload AppArmor profile '%s'.", profilePath)

	cmd := exec.Command(appArmorParserCmd, "--remove", "--write-cache", "--cache-loc", cachePath, profilePath)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return false, dbus.MakeFailedError(fmt.Errorf("Can't unload profile '%s': %s", profilePath, err))
	}

	logging.Info.Printf("Unload profile '%s': %s", profilePath, out)
	return true, nil
}

func InitializeDBus(conn *dbus.Conn) {
	d := apparmor{
		conn: conn,
	}

	propsSpec := map[string]map[string]*prop.Prop{
		ifaceName: {
			"ParserVersion": {
				Value:    getAppArmorVersion(),
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
