package system

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/natefinch/atomic"
	"golang.org/x/crypto/ssh"

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

// validateSSHAuthKey checks that newKey is a single well-formed OpenSSH
// authorized_keys entry and returns it in trimmed form. This is a safety
// check for the file format, not policy: any entry the OpenSSH parser
// accepts (including options) passes. Control characters are rejected so a
// single call can never write more than one line.
func validateSSHAuthKey(newKey string) (string, error) {
	key := strings.TrimSpace(newKey)
	if key == "" {
		return "", errors.New("SSH authorized key is empty")
	}
	for _, r := range key {
		if r < 0x20 || r == 0x7f {
			return "", errors.New("SSH authorized key contains control characters")
		}
	}

	_, _, _, rest, err := ssh.ParseAuthorizedKey([]byte(key))
	if err != nil {
		return "", fmt.Errorf("invalid SSH authorized key: %w", err)
	}
	if len(rest) > 0 {
		return "", errors.New("unexpected data after SSH authorized key")
	}

	return key, nil
}

func addSSHAuthKey(path string, newKey string) error {
	key, err := validateSSHAuthKey(newKey)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("failed to create SSH configuration directory: %w", err)
	}

	content, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read SSH authorized keys file: %w", err)
	}
	if len(content) > 0 && !bytes.HasSuffix(content, []byte("\n")) {
		content = append(content, '\n')
	}
	content = append(content, key...)
	content = append(content, '\n')

	if err := atomic.WriteFile(path, bytes.NewReader(content)); err != nil {
		return fmt.Errorf("failed to write SSH authorized keys file: %w", err)
	}
	// atomic.WriteFile keeps the permissions of an existing file, so tighten
	// them explicitly (files created before this change were 0644).
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("failed to set SSH authorized keys file permissions: %w", err)
	}

	return nil
}

func (d system) AddSSHAuthKey(newKey string) *dbus.Error {
	if err := addSSHAuthKey(sshAuthKeyFileName, newKey); err != nil {
		logging.Error.Printf("Failed to add SSH authorized key: %s", err)
		return dbus.MakeFailedError(err)
	}

	logging.Info.Printf("New SSH authentication key added for user root.")

	return nil
}

func (d system) ClearSSHAuthKeys() *dbus.Error {
	if err := os.Remove(sshAuthKeyFileName); err != nil && !os.IsNotExist(err) {
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
	default:
		return dbus.MakeFailedError(fmt.Errorf("unsupported driver: %s (only 'overlayfs' is currently supported)", backend))
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
