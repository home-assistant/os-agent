package rpi

import "testing"

func TestExtractVersion(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"Wed 14 Apr 2021 01:09:33 PM UTC (1618412973)", "1618412973"},
		{"  Wed 14 Apr 2021 (1618412973)  ", "1618412973"},
		{"000138c0", "000138c0"},
		{"", ""},
	}
	for _, c := range cases {
		got := extractVersion(c.in)
		if got != c.want {
			t.Errorf("extractVersion(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestParseSectionBootloaderOnly(t *testing.T) {
	out := `BOOTLOADER: up to date
   CURRENT: Wed 14 Apr 2021 01:09:33 PM UTC (1618412973)
    LATEST: Wed 14 Apr 2021 01:09:33 PM UTC (1618412973)
   RELEASE: default
`
	c, l := parseSection(out, "BOOTLOADER")
	if c != "Wed 14 Apr 2021 01:09:33 PM UTC (1618412973)" {
		t.Errorf("bootloader current = %q", c)
	}
	if l != "Wed 14 Apr 2021 01:09:33 PM UTC (1618412973)" {
		t.Errorf("bootloader latest = %q", l)
	}
	c, l = parseSection(out, "VL805")
	if c != "" || l != "" {
		t.Errorf("VL805 should be empty, got %q / %q", c, l)
	}
}

func TestParseSectionWithVL805(t *testing.T) {
	out := `BOOTLOADER: up to date
   CURRENT: Wed 14 Apr 2021 (1618412973)
    LATEST: Wed 14 Apr 2021 (1618412973)
   RELEASE: default

   VL805_FW: Dedicated VL805 EEPROM
      VL805: update available
    CURRENT: 000138c0
     LATEST: 000138c2
`
	blC, blL := parseSection(out, "BOOTLOADER")
	if blC != "Wed 14 Apr 2021 (1618412973)" || blL != "Wed 14 Apr 2021 (1618412973)" {
		t.Errorf("bootloader = %q / %q", blC, blL)
	}
	vlC, vlL := parseSection(out, "VL805")
	if vlC != "000138c0" || vlL != "000138c2" {
		t.Errorf("VL805 = %q / %q", vlC, vlL)
	}
}

func TestHasOutputLine(t *testing.T) {
	blocked := `BOOTLOADER: up to date
   CURRENT: Fri Jan  9 16:12:13 UTC 2026 (1767975133)
    LATEST: Fri Jan  9 16:12:13 UTC 2026 (1767975133)

  FLASHROM: no
   BLOCKED: yes
`
	notBlocked := `BOOTLOADER: update available
   CURRENT: Wed 14 Apr 2021 (1618412973)
    LATEST: Fri Jan  9 2026 (1767975133)

  FLASHROM: no
   BLOCKED: no
`
	if !hasOutputLine(blocked, "BLOCKED: yes") {
		t.Error(`hasOutputLine(blocked, "BLOCKED: yes") = false, want true`)
	}
	if hasOutputLine(notBlocked, "BLOCKED: yes") {
		t.Error(`hasOutputLine(notBlocked, "BLOCKED: yes") = true, want false`)
	}
	if hasOutputLine("BOOTLOADER: up to date\n", "BLOCKED: yes") {
		t.Error("hasOutputLine with no BLOCKED line = true, want false")
	}
}

func TestComposeVersions(t *testing.T) {
	cases := []struct {
		name                       string
		blCur, blLat, vlCur, vlLat string
		wantCurrent, wantLatest    string
	}{
		{
			name:        "bootloader update, no VL805",
			blCur:       "1618412973",
			blLat:       "1700000000",
			wantCurrent: "1618412973",
			wantLatest:  "1700000000",
		},
		{
			name:        "everything up to date, no VL805",
			blCur:       "1700000000",
			blLat:       "1700000000",
			wantCurrent: "1700000000",
			wantLatest:  "1700000000",
		},
		{
			name:        "bootloader current, VL805 update",
			blCur:       "1756466619",
			blLat:       "1756466619",
			vlCur:       "00ce405",
			vlLat:       "00ce407",
			wantCurrent: "1756466619-00ce405",
			wantLatest:  "1756466619-00ce407",
		},
		{
			name:        "bootloader and VL805 both update - bootloader wins",
			blCur:       "1618412973",
			blLat:       "1700000000",
			vlCur:       "00ce405",
			vlLat:       "00ce407",
			wantCurrent: "1618412973",
			wantLatest:  "1700000000",
		},
		{
			name:        "both up to date with VL805 present",
			blCur:       "1700000000",
			blLat:       "1700000000",
			vlCur:       "00ce407",
			vlLat:       "00ce407",
			wantCurrent: "1700000000",
			wantLatest:  "1700000000",
		},
		{
			name:  "empty bootloader returns empty",
			blCur: "",
			blLat: "",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cur, lat := composeVersions(c.blCur, c.blLat, c.vlCur, c.vlLat)
			if cur != c.wantCurrent || lat != c.wantLatest {
				t.Errorf("compose(%q,%q,%q,%q) = (%q,%q), want (%q,%q)",
					c.blCur, c.blLat, c.vlCur, c.vlLat, cur, lat, c.wantCurrent, c.wantLatest)
			}
		})
	}
}
