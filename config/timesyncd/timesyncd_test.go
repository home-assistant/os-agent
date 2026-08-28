package timesyncd

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/home-assistant/os-agent/utils/lineinfile"
)

func useTempConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "timesyncd.conf")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil { //nolint:gosec
		t.Fatal(err)
	}
	orig := configFile
	configFile = lineinfile.LineInFile{FilePath: path}
	t.Cleanup(func() { configFile = orig })
	return path
}

func readConfig(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

func TestUpdateNTPAdds(t *testing.T) {
	path := useTempConfig(t, "[Time]\nFallbackNTP=time.cloudflare.com\n")

	if err := updateTimesyncdConfigProperty("NTP", []string{"pool.ntp.org", "time.google.com"}); err != nil {
		t.Fatal(err)
	}

	expected := "[Time]\nNTP=pool.ntp.org time.google.com\nFallbackNTP=time.cloudflare.com\n"
	if got := readConfig(t, path); got != expected {
		t.Errorf("Expected %q, got %q", expected, got)
	}
}

func TestUpdateNTPReplaces(t *testing.T) {
	path := useTempConfig(t, "[Time]\nNTP=old.example.com\nFallbackNTP=time.cloudflare.com\n")

	if err := updateTimesyncdConfigProperty("NTP", []string{"new.example.com"}); err != nil {
		t.Fatal(err)
	}

	expected := "[Time]\nNTP=new.example.com\nFallbackNTP=time.cloudflare.com\n"
	if got := readConfig(t, path); got != expected {
		t.Errorf("Expected %q, got %q", expected, got)
	}
}

func TestUpdateNTPEmptyRemovesLine(t *testing.T) {
	path := useTempConfig(t, "[Time]\nNTP=old.example.com\nFallbackNTP=time.cloudflare.com\n")

	if err := updateTimesyncdConfigProperty("NTP", []string{}); err != nil {
		t.Fatal(err)
	}

	expected := "[Time]\nFallbackNTP=time.cloudflare.com\n"
	if got := readConfig(t, path); got != expected {
		t.Errorf("Expected %q, got %q", expected, got)
	}
	if servers := getNTPServers(); len(servers) != 0 {
		t.Errorf("Expected no servers, got %v", servers)
	}
}

func TestUpdateNTPEmptyWithoutLine(t *testing.T) {
	path := useTempConfig(t, "[Time]\nFallbackNTP=time.cloudflare.com\n")

	if err := updateTimesyncdConfigProperty("NTP", nil); err != nil {
		t.Fatal(err)
	}

	expected := "[Time]\nFallbackNTP=time.cloudflare.com\n"
	if got := readConfig(t, path); got != expected {
		t.Errorf("Expected %q, got %q", expected, got)
	}
}

func TestUpdateFallbackNTPEmptyRemovesLine(t *testing.T) {
	path := useTempConfig(t, "[Time]\nNTP=pool.ntp.org\nFallbackNTP=time.cloudflare.com\n")

	if err := updateTimesyncdConfigProperty("FallbackNTP", []string{}); err != nil {
		t.Fatal(err)
	}

	expected := "[Time]\nNTP=pool.ntp.org\n"
	if got := readConfig(t, path); got != expected {
		t.Errorf("Expected %q, got %q", expected, got)
	}
}

func TestGetServersRoundTrip(t *testing.T) {
	useTempConfig(t, "[Time]\n")

	if err := updateTimesyncdConfigProperty("NTP", []string{"pool.ntp.org", "time.google.com"}); err != nil {
		t.Fatal(err)
	}
	if err := updateTimesyncdConfigProperty("FallbackNTP", []string{"time.cloudflare.com"}); err != nil {
		t.Fatal(err)
	}

	expected := []string{"pool.ntp.org", "time.google.com"}
	if got := getNTPServers(); !reflect.DeepEqual(got, expected) {
		t.Errorf("Expected %v, got %v", expected, got)
	}
	expectedFallback := []string{"time.cloudflare.com"}
	if got := getFallbackNTPServers(); !reflect.DeepEqual(got, expectedFallback) {
		t.Errorf("Expected %v, got %v", expectedFallback, got)
	}
}
