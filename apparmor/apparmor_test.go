package apparmor

import (
	"strings"
	"testing"
)

func TestCheckProfileNamesValid(t *testing.T) {
	tests := []struct {
		name  string
		names []string
	}{
		{"single profile", []string{"local_example"}},
		{"with child profile", []string{"local_example", "local_example///usr/bin/child"}},
		{"with hat", []string{"local_example", "local_example//hat"}},
		{"several children", []string{"local_example", "local_example//a", "local_example//b", "local_example//c"}},
		{"child listed first", []string{"local_example//child", "local_example"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := checkProfileNames("/etc/apparmor.d/hassio/local_example", tt.names); err != nil {
				t.Fatalf("expected profile file to be valid, got error: %s", err)
			}
		})
	}
}

func TestCheckProfileNamesInvalid(t *testing.T) {
	tests := []struct {
		name    string
		names   []string
		wantErr string
	}{
		{"no profile", nil, "does not define profile 'local_example'"},
		{"wrong name", []string{"example"}, "unexpected profile 'example'"},
		{"additional profile", []string{"local_example", "docker-default"}, "unexpected profile 'docker-default'"},
		{"additional profile first", []string{"hassio-supervisor", "local_example"}, "unexpected profile 'hassio-supervisor'"},
		{"child of other profile", []string{"local_example", "docker-default//hat"}, "unexpected profile 'docker-default//hat'"},
		{"prefix without separator", []string{"local_example", "local_example2"}, "unexpected profile 'local_example2'"},
		{"only children", []string{"local_example//child"}, "does not define profile 'local_example'"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkProfileNames("/etc/apparmor.d/hassio/local_example", tt.names)
			if err == nil {
				t.Fatal("expected profile file to be rejected")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("expected error containing %q, got %q", tt.wantErr, err)
			}
		})
	}
}
