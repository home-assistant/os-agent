package udisks2

import (
	"context"
	"fmt"
	"strings"

	"github.com/godbus/dbus/v5"
)

var noOptions = map[string]dbus.Variant{}

func (o *Block) GetDeviceString(ctx context.Context) (*string, error) {

	device, err := o.GetDevice(ctx)
	if err != nil {
		return nil, err
	}

	s := strings.Trim(string(device), "\x00")
	return &s, nil
}

func (m *Manager) ResolveDeviceFromLabel(label string) (*dbus.ObjectPath, error) {
	devspec := map[string]dbus.Variant{"label": dbus.MakeVariant(label)}
	blockObjects, err := m.ResolveDevice(context.Background(), devspec, noOptions)

	if err != nil {
		return nil, err
	}
	if len(blockObjects) != 1 {
		return nil, fmt.Errorf("Expected single block device with file system label \"%s\", found %d", label, len(blockObjects))
	}

	/* Get Partition object of the data partition */
	return &blockObjects[0], nil
}
