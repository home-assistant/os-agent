package system

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	testKeyEd25519 = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIDXD8u9KB94/l1YukYflKOsO7KzoSEQD4dNNlWY9zaQP test@example.com"
	testKeyEcdsa   = "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBIGeXP8EbMxj8Ws5m7tdN5YR9BryZNyG+L9670o7eSZog4G03n16bs7Yz0oV1J4sWOkhZNUak6g3IM1jnMLFvgE= ecdsa@example"
	testKeyRSA     = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDRX06DJ3LwTht9XexLQPaz8zBE+FQIRvxa+AcltbzYHmOH+H6NxUFCK2KPF26aCiaDCzccO14Q44ObSxk6WsG7vsX9TemQL7MEpag6aBOzllKYOcHOEDi6vkKPVEy8MGM5Z0M2tWIkFuJ6lmTbd9zo2Udw/Tv03iftGVFJe0QjIevcbxZF+MFG7iN+RioyA9cRfW3FVOsgwU6ax2NBEoi44MmAnWOEft1QPML7hYHOUpqiC+hR+UKxCLPSaRbsAkA2nFZVOuNixt501KkM2/JLkVfIXbOk1FbL845AWOaECt3XZzVK1RoPZHDJ9+q8Zq60bNQIjo6MsTKnzrfFRPa3 rsa@example.com"
)

func TestValidateSSHAuthKeyValid(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want string
	}{
		{"plain key", testKeyEd25519, testKeyEd25519},
		{"rsa key", testKeyRSA, testKeyRSA},
		{"surrounding whitespace trimmed", "  " + testKeyEd25519 + "\n", testKeyEd25519},
		{"no comment", strings.Join(strings.Fields(testKeyEd25519)[:2], " "), strings.Join(strings.Fields(testKeyEd25519)[:2], " ")},
		{"with options", `restrict,command="/bin/true" ` + testKeyEd25519, `restrict,command="/bin/true" ` + testKeyEd25519},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateSSHAuthKey(tt.key)
			if err != nil {
				t.Fatalf("expected key to be valid, got error: %s", err)
			}
			if got != tt.want {
				t.Errorf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestValidateSSHAuthKeyInvalid(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{"empty", ""},
		{"whitespace only", " \n"},
		{"garbage", "not a key"},
		{"missing key data", "ssh-ed25519"},
		{"invalid base64", "ssh-ed25519 !!!invalid!!! test@example.com"},
		{"newline injection", testKeyEd25519 + "\n" + testKeyEcdsa},
		{"embedded carriage return", strings.Replace(testKeyEd25519, " test@", "\r test@", 1)},
		{"tab separators", strings.ReplaceAll(testKeyEd25519, " ", "\t")},
		{"oversized entry", testKeyEd25519 + " " + strings.Repeat("a", 3000)},
		{"embedded escape character", strings.Replace(testKeyEd25519, " test@", "\x1b test@", 1)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := validateSSHAuthKey(tt.key); err == nil {
				t.Errorf("expected key %q to be rejected", tt.key)
			}
		})
	}
}

func TestAddSSHAuthKeyCreatesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".ssh", "authorized_keys")

	if err := addSSHAuthKey(path, testKeyEd25519+"\n"); err != nil {
		t.Fatalf("failed to add key: %s", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read authorized keys file: %s", err)
	}
	if string(content) != testKeyEd25519+"\n" {
		t.Errorf("unexpected file content: %q", content)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("failed to stat authorized keys file: %s", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("expected file mode 0600, got %o", info.Mode().Perm())
	}

	dirInfo, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatalf("failed to stat SSH configuration directory: %s", err)
	}
	if dirInfo.Mode().Perm() != 0o700 {
		t.Errorf("expected directory mode 0700, got %o", dirInfo.Mode().Perm())
	}
}

func TestAddSSHAuthKeyAppends(t *testing.T) {
	path := filepath.Join(t.TempDir(), "authorized_keys")

	if err := addSSHAuthKey(path, testKeyEd25519); err != nil {
		t.Fatalf("failed to add first key: %s", err)
	}
	if err := addSSHAuthKey(path, testKeyEcdsa); err != nil {
		t.Fatalf("failed to add second key: %s", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read authorized keys file: %s", err)
	}
	if string(content) != testKeyEd25519+"\n"+testKeyEcdsa+"\n" {
		t.Errorf("unexpected file content: %q", content)
	}
}

func TestAddSSHAuthKeyMissingTrailingNewline(t *testing.T) {
	path := filepath.Join(t.TempDir(), "authorized_keys")
	if err := os.WriteFile(path, []byte(testKeyEd25519), 0o600); err != nil {
		t.Fatalf("failed to create authorized keys file: %s", err)
	}

	if err := addSSHAuthKey(path, testKeyEcdsa); err != nil {
		t.Fatalf("failed to add key: %s", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read authorized keys file: %s", err)
	}
	if string(content) != testKeyEd25519+"\n"+testKeyEcdsa+"\n" {
		t.Errorf("unexpected file content: %q", content)
	}
}

func TestAddSSHAuthKeyInvalidKeyLeavesFileUntouched(t *testing.T) {
	path := filepath.Join(t.TempDir(), "authorized_keys")
	if err := os.WriteFile(path, []byte(testKeyEd25519+"\n"), 0o600); err != nil {
		t.Fatalf("failed to create authorized keys file: %s", err)
	}

	if err := addSSHAuthKey(path, "not a key"); err == nil {
		t.Fatal("expected invalid key to be rejected")
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read authorized keys file: %s", err)
	}
	if string(content) != testKeyEd25519+"\n" {
		t.Errorf("unexpected file content: %q", content)
	}
}

func TestClearSSHAuthKeysRemovesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "authorized_keys")
	if err := os.WriteFile(path, []byte(testKeyEd25519+"\n"), 0o600); err != nil {
		t.Fatalf("failed to create authorized keys file: %s", err)
	}

	if err := clearSSHAuthKeys(path); err != nil {
		t.Fatalf("failed to clear authorized keys: %s", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("expected authorized keys file to be removed, got %v", err)
	}
}

func TestClearSSHAuthKeysMissingFileIsSuccess(t *testing.T) {
	path := filepath.Join(t.TempDir(), "authorized_keys")

	if err := clearSSHAuthKeys(path); err != nil {
		t.Errorf("expected clearing a missing file to succeed, got: %s", err)
	}
}

func TestClearSSHAuthKeysReportsFailure(t *testing.T) {
	// A non-empty directory cannot be removed with os.Remove
	path := filepath.Join(t.TempDir(), "authorized_keys")
	if err := os.MkdirAll(filepath.Join(path, "child"), 0o700); err != nil {
		t.Fatalf("failed to create directory: %s", err)
	}

	if err := clearSSHAuthKeys(path); err == nil {
		t.Error("expected clearing to report the removal failure")
	}
}

func TestAddSSHAuthKeyTightensPermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "authorized_keys")
	// Files created before atomic writes were introduced were 0644
	if err := os.WriteFile(path, []byte(testKeyEd25519+"\n"), 0o644); err != nil { //nolint:gosec
		t.Fatalf("failed to create authorized keys file: %s", err)
	}

	if err := addSSHAuthKey(path, testKeyEcdsa); err != nil {
		t.Fatalf("failed to add key: %s", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("failed to stat authorized keys file: %s", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("expected file mode 0600, got %o", info.Mode().Perm())
	}
}
