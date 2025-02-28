package lineinfile

import (
	"regexp"
	"strings"
	"testing"
)

const (
	contentNoNTP = `[Time]
FallbackNTP=time.cloudflare.com
# Speed-up boot as first attempt is done before network is up
ConnectionRetrySec=10
`
	contentNTPSet = `[Time]
NTP=ntp.example.com
FallbackNTP=time.cloudflare.com
# Speed-up boot as first attempt is done before network is up
ConnectionRetrySec=10
`
	contentNTPCommented = `[Time]
#NTP=ntp.example.com
FallbackNTP=time.cloudflare.com
# Speed-up boot as first attempt is done before network is up
ConnectionRetrySec=10
`
	contentNTPNotAfter = `NTP=ntp.example.com
[Time]
FallbackNTP=time.cloudflare.com
ConnectionRetrySec=10
`
)

func TestFindExisting(t *testing.T) {
	lines := strings.Split(contentNTPSet, "\n")
	result := processFind(`^\s*(NTP=).*$`, `\[Time\]`, lines)
	if result == nil {
		t.Errorf("Expected a result, got nil")
		return
	}
	t.Logf("Result: %s", *result)
	expected := "NTP=ntp.example.com"
	if *result != expected {
		t.Errorf("Expected %s, got %s", expected, *result)
	}
}

func TestFindMissing(t *testing.T) {
	lines := strings.Split(contentNoNTP, "\n")
	result := processFind(`^\s*(NTP=).*$`, `\[Time\]`, lines)
	if result != nil {
		t.Errorf("Expected nil, got %s", *result)
	}
}

func TestFindMissingNotAfter(t *testing.T) {
	lines := strings.Split(contentNTPNotAfter, "\n")
	result := processFind(`^\s*(NTP=).*$`, `\[Time\]`, lines)
	if result != nil {
		t.Errorf("Expected nil, got %s", *result)
	}
}

func TestFindEmptyAfter(t *testing.T) {
	content := "KEY=value\nKEY2=value2"
	lines := strings.Split(content, "\n")
	result := processFind(`^KEY=`, "", lines)
	if *result != "KEY=value" {
		t.Errorf("Expected a result, got nil")
		return
	}
}

func TestPresentSimple(t *testing.T) {
	params := NewPresentParams("NTP=ntp2.example.com")
	params.Regexp, _ = regexp.Compile(`^\s*#?\s*(NTP=).*$`)
	params.After = `\[Time\]`
	lines := strings.Split(contentNTPSet, "\n")
	processed, _ := processPresent(lines, params)
	result := strings.Join(processed, "\n")
	expected := strings.Replace(contentNTPSet, "NTP=ntp.example.com", "NTP=ntp2.example.com", 1)
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestPresentNotAfter(t *testing.T) {
	params := NewPresentParams("NTP=ntp2.example.com")
	params.Regexp, _ = regexp.Compile(`^\s*#?\s*(NTP=).*$`)
	params.After = `\[Time\]`
	lines := strings.Split(contentNTPNotAfter, "\n")
	processed, _ := processPresent(lines, params)
	result := strings.Join(processed, "\n")
	expected := `NTP=ntp.example.com
[Time]
NTP=ntp2.example.com
FallbackNTP=time.cloudflare.com
ConnectionRetrySec=10
`
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestPresentCommented(t *testing.T) {
	params := NewPresentParams("NTP=ntp.example.com")
	params.Regexp, _ = regexp.Compile(`^\s*#?\s*(NTP=).*$`)
	params.After = `\[Time\]`
	lines := strings.Split(contentNTPCommented, "\n")
	processed, _ := processPresent(lines, params)
	result := strings.Join(processed, "\n")
	if result != contentNTPSet {
		t.Errorf("Expected %s, got %s", contentNTPSet, result)
	}
}

func TestPresentAdded(t *testing.T) {
	params := NewPresentParams("NTP=ntp.example.com")
	params.Regexp, _ = regexp.Compile(`^\s*#?\s*(NTP=).*$`)
	params.After = `\[Time\]`
	lines := strings.Split(contentNoNTP, "\n")
	processed, _ := processPresent(lines, params)
	result := strings.Join(processed, "\n")
	expected := contentNTPSet
	if result != expected {
		t.Errorf("Expected %s, got %s", contentNTPSet, result)
	}
}

func TestPresentAppendedEOF(t *testing.T) {
	params := NewPresentParams("NTP=ntp.example.com")
	params.Regexp, _ = regexp.Compile(`^\s*#?\s*(NTP=).*$`)
	params.After = `EOF`
	lines := strings.Split(contentNoNTP, "\n")
	processed, _ := processPresent(lines, params)
	result := strings.Join(processed, "\n")
	expected := contentNoNTP + "\nNTP=ntp.example.com"
	if result != expected {
		t.Errorf("Expected %s, got %s", contentNTPSet, result)
	}
}

func TestPresentEmptyContent(t *testing.T) {
	params := NewPresentParams("NTP=ntp.example.com")
	params.Regexp, _ = regexp.Compile(`^\s*#?\s*(NTP=).*$`)
	params.After = `EOF`
	var lines []string
	processed, _ := processPresent(lines, params)
	result := strings.Join(processed, "\n")
	expected := "NTP=ntp.example.com"
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestPresentNoBeforeAfterRegex(t *testing.T) {
	params := NewPresentParams("NTP=ntp.example.com")
	params.Regexp, _ = regexp.Compile(`^\s*#?\s*(NTP=).*$`)
	content := "NTP=ntp.example.com"
	lines := strings.Split(content, "\n")
	processed, _ := processPresent(lines, params)
	result := strings.Join(processed, "\n")
	if result != content {
		t.Errorf("Expected %s, got %s", content, result)
	}
}

func TestPresentNoRegexp(t *testing.T) {
	params := NewPresentParams("NTP=ntp.example.com")
	params.After = `\[Time\]`
	lines := strings.Split(contentNTPSet, "\n")
	_, err := processPresent(lines, params)
	if err == nil {
		t.Errorf("Expected an error, got nil")
	}
}

func TestAbsent(t *testing.T) {
	params := NewAbsentParams()
	params.Regexp, _ = regexp.Compile(`^\s*(NTP=).*$`)
	params.After = `\[Time\]`
	lines := strings.Split(contentNTPSet, "\n")
	processed, _ := processAbsent(lines, params)
	result := strings.Join(processed, "\n")
	expected := `[Time]
FallbackNTP=time.cloudflare.com
# Speed-up boot as first attempt is done before network is up
ConnectionRetrySec=10
`
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestAbsentNotAfter(t *testing.T) {
	params := NewAbsentParams()
	params.Regexp, _ = regexp.Compile(`^\s*(NTP=).*$`)
	params.After = `\[Time\]`
	lines := strings.Split(contentNTPNotAfter, "\n")
	processed, _ := processAbsent(lines, params)
	result := strings.Join(processed, "\n")
	expected := contentNTPNotAfter
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestAbsentNoRegexp(t *testing.T) {
	params := NewAbsentParams()
	params.After = `\[Time\]`
	lines := strings.Split(contentNTPSet, "\n")
	_, err := processAbsent(lines, params)
	if err == nil {
		t.Errorf("Expected an error, got nil")
	}
}

func TestAbsentNoBeforeAfterRegex(t *testing.T) {
	params := NewAbsentParams()
	params.Regexp, _ = regexp.Compile(`^\s*#?\s*(NTP=).*$`)
	content := "NTP=ntp.example.com"
	lines := strings.Split(content, "\n")
	processed, _ := processAbsent(lines, params)
	result := strings.Join(processed, "\n")
	expected := ""
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}
