package udisks2

import (
	"context"
	"fmt"
	"strings"

	"github.com/godbus/dbus/v5"
)

type UDisks2Helper struct {
	conn    *dbus.Conn
	manager *Manager
}

func NewUDisks2(conn *dbus.Conn) UDisks2Helper {
	busObj := conn.Object("org.freedesktop.UDisks2", "/org/freedesktop/UDisks2/Manager")
	manager := NewManager(busObj)

	d := UDisks2Helper{
		conn:    conn,
		manager: manager,
	}

	return d
}

var noOptions = map[string]dbus.Variant{}

func (u UDisks2Helper) GetDeviceFromLabel(label string) (*string, error) {
	devspec := map[string]dbus.Variant{"label": dbus.MakeVariant(label)}
	blockObjects, err := u.manager.ResolveDevice(context.Background(), devspec, noOptions)

	if err != nil {
		return nil, err
	}
	if len(blockObjects) != 1 {
		return nil, fmt.Errorf("Expected single block device with file system label \"%s\", found %d", label, len(blockObjects))
	}

	/* Get Partition object of the data partition */
	busObjectBlock := u.conn.Object("org.freedesktop.UDisks2", blockObjects[0])
	partition := NewPartition(busObjectBlock)
	table, err := partition.GetTable(context.Background())
	if err != nil {
		return nil, err
	}

	/* Get Block device of partition table */
	busObjectParentBlock := u.conn.Object("org.freedesktop.UDisks2", table)
	parentBlock := NewBlock(busObjectParentBlock)

	device, err := parentBlock.GetDevice(context.Background())
	if err != nil {
		return nil, err
	}

	s := strings.Trim(string(device), "\x00")
	return &s, nil
}

func (u UDisks2Helper) FormatDeviceWithSinglePartition(devicePath string, uuid string, name string) error {
	devspec := map[string]dbus.Variant{"path": dbus.MakeVariant(devicePath)}
	blockObjects, err := u.manager.ResolveDevice(context.Background(), devspec, noOptions)

	if err != nil {
		return err
	}
	if len(blockObjects) != 1 {
		return fmt.Errorf("Expected single block device with device path \"%s\", found %d", devicePath, len(blockObjects))
	}

	blockObjectPath := blockObjects[0]
	fmt.Printf("Formatting device %s\n", devicePath)

	busObjectParentBlock := u.conn.Object("org.freedesktop.UDisks2", blockObjectPath)
	parentBlock := NewBlock(busObjectParentBlock)
	err = parentBlock.Format(context.Background(), "gpt", noOptions)
	if err != nil {
		return err
	}

	parentPartitionTable := NewPartitionTable(busObjectParentBlock)
	createdPartition, err :=
		parentPartitionTable.CreatePartition(context.Background(), 0, 0,
			uuid, name, noOptions)
	if err != nil {
		return err
	}
	fmt.Printf("New partition D-Bus object %s\n", createdPartition)

	return nil
}
