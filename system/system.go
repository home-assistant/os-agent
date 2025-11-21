package system

import (
	"fmt"
	"os"
	"strings"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"

	logging "github.com/home-assistant/os-agent/utils/log"
)

const (
	objectPath                = "/io/hass/os/System"
	ifaceName                 = "io.hass.os.System"
	labelDataFileSystem       = "hassos-data"
	labelOverlayFileSystem    = "hassos-overlay"
	kernelCommandLine         = "/mnt/boot/cmdline.txt"
	tmpKernelCommandLine      = "/mnt/boot/.tmp.cmdline.txt"
	sshAuthKeyFileName        = "/root/.ssh/authorized_keys"
	containerdSnapshotterFlag = "/mnt/data/.docker-use-containerd-snapshotter"
)

type system struct {
	conn *dbus.Conn
}

func (d system) ScheduleWipeDevice() (bool, *dbus.Error) {

	data, err := os.ReadFile(kernelCommandLine)
	if err != nil {
		fmt.Println(err)
		return false, dbus.MakeFailedError(err)
	}

	datastr := strings.TrimSpace(string(data))
	datastr += " haos.wipe=1"

	err = os.WriteFile(tmpKernelCommandLine, []byte(datastr), 0644) //nolint:gosec
	if err != nil {
		fmt.Println(err)
		return false, dbus.MakeFailedError(err)
	}

	// Boot is mounted sync on Home Assistant OS, so just rename should be fine.
	err = os.Rename(tmpKernelCommandLine, kernelCommandLine)
	if err != nil {
		fmt.Println(err)
		return false, dbus.MakeFailedError(err)
	}

	logging.Info.Printf("Device will get wiped on next reboot!")
	return true, nil
}

func (d system) AddSSHAuthKey(newKey string) *dbus.Error {

	file, err := os.OpenFile(sshAuthKeyFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logging.Error.Printf("Failed to open SSH authentication file %s: %s", sshAuthKeyFileName, err)
		return dbus.MakeFailedError(err)
	}

	defer file.Close()

	if _, err := file.WriteString(newKey + "\n"); err != nil {
		logging.Error.Printf("Failed to write SSH authentication file: %s.", err)
		return dbus.MakeFailedError(err)
	}

	logging.Info.Printf("New SSH authentication key added for user root.")

	return nil
}

func (d system) ClearSSHAuthKeys() *dbus.Error {
	if err := os.Remove(sshAuthKeyFileName); err != nil && os.IsNotExist(err) {
		logging.Error.Printf("Failed to delete SSH authentication file %s: %s", sshAuthKeyFileName, err)
		return dbus.MakeFailedError(err)
	}

	return nil
}

func (d system) MigrateDockerStorageDriver(backend string) *dbus.Error {
	switch backend {
	case "overlayfs":
		// Write the backend name to the flag file
		err := os.WriteFile(containerdSnapshotterFlag, []byte(backend), 0644) //nolint:gosec
		if err != nil {
			logging.Error.Printf("Failed to write containerd snapshotter flag: %s", err)
			return dbus.MakeFailedError(err)
		}
		logging.Info.Printf("Storage driver set to overlayfs containerd snapshotter")
	case "overlay2":
		// Graph driver -> remove the flag file to disable the snapshotter
		if err := os.Remove(containerdSnapshotterFlag); err != nil && !os.IsNotExist(err) {
			logging.Error.Printf("Failed to remove containerd snapshotter flag: %s", err)
			return dbus.MakeFailedError(err)
		}
		logging.Info.Printf("Storage driver set to overlay2 graph driver")
	default:
		return dbus.MakeFailedError(fmt.Errorf("unsupported driver: %s (only 'overlayfs' and 'overlay2' are supported)", backend))
	}

	return nil
}

func InitializeDBus(conn *dbus.Conn) {
	d := system{
		conn: conn,
	}

	err := conn.Export(d, objectPath, ifaceName)
	if err != nil {
		logging.Critical.Panic(err)
	}

	node := &introspect.Node{
		Name: objectPath,
		Interfaces: []introspect.Interface{
			introspect.IntrospectData,
			{
				Name:    ifaceName,
				Methods: introspect.Methods(d),
			},
		},
	}

	err = conn.Export(introspect.NewIntrospectable(node), objectPath, "org.freedesktop.DBus.Introspectable")
	if err != nil {
		logging.Critical.Panic(err)
	}

	logging.Info.Printf("Exposing object %s with interface %s ...", objectPath, ifaceName)
}
