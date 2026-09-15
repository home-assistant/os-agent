//go:build integration

package apparmor

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeProfile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestValidateProfileWithParser(t *testing.T) {
	dir := t.TempDir()
	valid := writeProfile(t, dir, "local_example", "#include <tunables/global>\nprofile local_example flags=(attach_disconnected) {\n  #include <abstractions/base>\n  profile child {\n  }\n  ^hat {\n  }\n}\n")
	smuggled := writeProfile(t, dir, "local_smuggle", "#include <tunables/global>\nprofile local_smuggle {\n  #include <abstractions/base>\n}\n\n  profile docker-default flags=(complain) {\n  #include <abstractions/base>\n}\n")
	barePath := writeProfile(t, dir, "local_bare", "#include <tunables/global>\nprofile local_bare {\n}\n/usr/bin/foo {\n}\n")
	syntax := writeProfile(t, dir, "local_syntax", "profile local_syntax {\n  this is not valid,\n")
	wrongName := writeProfile(t, dir, "local_wrong", "profile something_else {\n}\n")
	trailingSpace := writeProfile(t, dir, "local_space", "profile \"local_space \" {\n}\n")

	if err := validateProfile(context.Background(), valid); err != nil {
		t.Fatalf("valid profile rejected: %s", err)
	}
	for _, tt := range []struct{ path, want string }{
		{smuggled, "unexpected profile 'docker-default'"},
		{barePath, "unexpected profile '/usr/bin/foo'"},
		{syntax, "can't parse profile"},
		{wrongName, "unexpected profile 'something_else'"},
		{trailingSpace, "unexpected profile 'local_space '"},
	} {
		err := validateProfile(context.Background(), tt.path)
		if err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: expected error containing %q, got %v", filepath.Base(tt.path), tt.want, err)
		} else {
			t.Logf("%s: %s", filepath.Base(tt.path), err)
		}
	}
}
