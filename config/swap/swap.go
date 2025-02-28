package swap

import (
	"fmt"
	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"
	"github.com/home-assistant/os-agent/utils/lineinfile"
	logging "github.com/home-assistant/os-agent/utils/log"
	"os"
	"regexp"
	"strconv"
	"strings"
)

const (
	objectPath     = "/io/hass/os/Config/Swap"
	ifaceName      = "io.hass.os.Config.Swap"
	swapPath       = "/etc/default/haos-swapfile"
	swappinessPath = "/etc/sysctl.d/15-swappiness.conf"
)

var (
	optSwapSize      string
	optSwappiness    int
	swapFileEditor   = lineinfile.LineInFile{FilePath: swapPath}
	swappinessEditor = lineinfile.LineInFile{FilePath: swappinessPath}
)

type swap struct {
	conn  *dbus.Conn
	props *prop.Properties
}

// Read swappiness from kernel procfs. If it fails, log errors and return 60
// as it's usual kernel default.
func readKernelSwappiness() int {
	content, err := os.ReadFile("/proc/sys/vm/swappiness")
	if err != nil {
		logging.Error.Printf("Failed to read kernel swappiness: %s", err)
		return 60
	}

	swappiness, err := strconv.Atoi(strings.TrimSpace(string(content)))
	if err != nil {
		logging.Error.Printf("Failed to parse kernel swappiness: %s", err)
		return 60
	}

	return swappiness
}

func getSwapSize() string {
	found, err := swapFileEditor.Find(`^SWAPSIZE=`, "", true)
	if found == nil || err != nil {
		return ""
	}

	matches := regexp.MustCompile(`^SWAPSIZE=(.*)`).FindStringSubmatch(*found)
	if len(matches) > 1 {
		return matches[1]
	}

	return ""
}

func getSwappiness() int {
	found, err := swappinessEditor.Find(`^vm.swappiness\s*=`, "", true)
	if found == nil || err != nil {
		return readKernelSwappiness()
	}

	matches := regexp.MustCompile(`^vm.swappiness\s*=\s*(\d+)`).FindStringSubmatch(*found)
	if len(matches) > 1 {
		if swappiness, err := strconv.Atoi(matches[1]); err == nil {
			return swappiness
		}
	}

	return readKernelSwappiness()
}

func setSwapSize(c *prop.Change) *dbus.Error {
	swapSize, ok := c.Value.(string)
	if !ok {
		return dbus.MakeFailedError(fmt.Errorf("invalid type for swap size"))
	}

	re := regexp.MustCompile(`^\d+([KMG]?(i?B)?)?$`)
	if !re.MatchString(swapSize) {
		return dbus.MakeFailedError(fmt.Errorf("invalid swap size format"))
	}

	params := lineinfile.NewPresentParams(fmt.Sprintf("SWAPSIZE=%s", swapSize))
	params.Regexp, _ = regexp.Compile(`^[#\s]*SWAPSIZE=`)

	if err := swapFileEditor.Present(params); err != nil {
		return dbus.MakeFailedError(fmt.Errorf("failed to set swap size: %w", err))
	}

	return nil
}

func setSwappiness(c *prop.Change) *dbus.Error {
	swappiness, ok := c.Value.(int32)
	if !ok {
		return dbus.MakeFailedError(fmt.Errorf("swappiness must be int32, got %T", c.Value))
	}

	if swappiness < 0 || swappiness > 100 {
		return dbus.MakeFailedError(fmt.Errorf("swappiness must be between 0 and 100"))
	}

	params := lineinfile.NewPresentParams(fmt.Sprintf("vm.swappiness=%d", swappiness))
	params.Regexp, _ = regexp.Compile(`^[#\s]*vm.swappiness\s*=`)

	if err := swappinessEditor.Present(params); err != nil {
		return dbus.MakeFailedError(fmt.Errorf("failed to set swappiness: %w", err))
	}

	return nil
}

func InitializeDBus(conn *dbus.Conn) {
	d := swap{
		conn: conn,
	}

	optSwapSize = getSwapSize()
	optSwappiness = getSwappiness()

	propsSpec := map[string]map[string]*prop.Prop{
		ifaceName: {
			"SwapSize": {
				Value:    optSwapSize,
				Writable: true,
				Emit:     prop.EmitTrue,
				Callback: setSwapSize,
			},
			"Swappiness": {
				Value:    optSwappiness,
				Writable: true,
				Emit:     prop.EmitTrue,
				Callback: setSwappiness,
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
