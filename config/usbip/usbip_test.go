package usbip

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildConfig(t *testing.T) {
	got := buildConfig("192.168.1.10", "1-1.2", 5000, "My Device")
	expected := "HOST=192.168.1.10\nBUSID=1-1.2\nPORT=5000\nNAME=My Device\n"
	if got != expected {
		t.Errorf("buildConfig() = %q, want %q", got, expected)
	}
}

func TestBuildConfigDefaultPort(t *testing.T) {
	got := buildConfig("host", "1-1", 0, "")
	if !strings.Contains(got, "PORT=3240\n") {
		t.Errorf("buildConfig() with port 0 should default to 3240, got %q", got)
	}
}

func TestValidateIdentifier(t *testing.T) {
	valid := []string{
		"a1b2c3",
		"123e4567-e89b-12d3-a456-426614174000",
		"device_1.conf",
	}
	for _, id := range valid {
		if err := validateIdentifier(id); err != nil {
			t.Errorf("validateIdentifier(%q) returned error: %v", id, err)
		}
	}

	invalid := []string{
		"",
		".",
		"..",
		"a/b",
		"../etc/passwd",
		`a\b`,
		"with space",
		"new\nline",
	}
	for _, id := range invalid {
		if err := validateIdentifier(id); err == nil {
			t.Errorf("validateIdentifier(%q) expected error, got nil", id)
		}
	}
}

func TestWriteListRemove(t *testing.T) {
	dir := t.TempDir()
	old := configDir
	configDir = filepath.Join(dir, "remote-devices")
	defer func() { configDir = old }()

	d := usbip{}

	// List on a missing directory returns an empty slice, not an error.
	ids, derr := d.ListRemoteDevices()
	if derr != nil {
		t.Fatalf("List() on missing dir returned error: %v", derr)
	}
	if len(ids) != 0 {
		t.Fatalf("List() on missing dir = %v, want empty", ids)
	}

	// Write creates the file.
	if derr := d.WriteRemoteDevice("dev-1", "10.0.0.1", "2-1", 0, "Label"); derr != nil {
		t.Fatalf("Write() returned error: %v", derr)
	}

	content, err := os.ReadFile(filepath.Join(configDir, "dev-1.conf"))
	if err != nil {
		t.Fatalf("reading written config: %v", err)
	}
	expected := "HOST=10.0.0.1\nBUSID=2-1\nPORT=3240\nNAME=Label\n"
	if string(content) != expected {
		t.Errorf("config content = %q, want %q", string(content), expected)
	}

	// Write is an upsert: overwrite the same identifier.
	if derr := d.WriteRemoteDevice("dev-1", "10.0.0.2", "2-1", 0, "Label"); derr != nil {
		t.Fatalf("Write() (update) returned error: %v", derr)
	}
	content, _ = os.ReadFile(filepath.Join(configDir, "dev-1.conf"))
	if !strings.Contains(string(content), "HOST=10.0.0.2\n") {
		t.Errorf("update did not change host, got %q", string(content))
	}

	if derr := d.WriteRemoteDevice("dev-2", "10.0.0.3", "3-1", 0, ""); derr != nil {
		t.Fatalf("Write() returned error: %v", derr)
	}

	ids, derr = d.ListRemoteDevices()
	if derr != nil {
		t.Fatalf("List() returned error: %v", derr)
	}
	// List() returns identifiers sorted; assert order directly.
	if len(ids) != 2 || ids[0] != "dev-1" || ids[1] != "dev-2" {
		t.Errorf("List() = %v, want [dev-1 dev-2]", ids)
	}

	// Remove deletes the file.
	if derr := d.RemoveRemoteDevice("dev-1"); derr != nil {
		t.Fatalf("Remove() returned error: %v", derr)
	}
	if _, err := os.Stat(filepath.Join(configDir, "dev-1.conf")); !os.IsNotExist(err) {
		t.Errorf("file dev-1.conf should have been removed")
	}

	// Removing a non-existent identifier is a no-op success.
	if derr := d.RemoveRemoteDevice("dev-1"); derr != nil {
		t.Errorf("Remove() of missing identifier should succeed, got %v", derr)
	}
}

func TestWriteValidation(t *testing.T) {
	dir := t.TempDir()
	old := configDir
	configDir = filepath.Join(dir, "remote-devices")
	defer func() { configDir = old }()

	d := usbip{}

	if derr := d.WriteRemoteDevice("../escape", "host", "1-1", 0, ""); derr == nil {
		t.Error("Write() with traversal identifier should fail")
	}
	if derr := d.WriteRemoteDevice("dev", "", "1-1", 0, ""); derr == nil {
		t.Error("Write() with empty host should fail")
	}
	if derr := d.WriteRemoteDevice("dev", "host", "", 0, ""); derr == nil {
		t.Error("Write() with empty busID should fail")
	}
	if derr := d.WriteRemoteDevice("dev", "host", "1-1", 0, "bad\nname"); derr == nil {
		t.Error("Write() with newline in name should fail")
	}
}
