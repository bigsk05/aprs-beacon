package packet

import (
	"strings"
	"testing"

	"aprs-beacon/internal/meta"
)

func TestFormatLatitude(t *testing.T) {
	cases := []struct {
		in   float64
		want string
	}{
		{30.876111, "3052.57N"},
		{-30.876111, "3052.57S"},
		{0, "0000.00N"},
	}
	for _, c := range cases {
		if got := formatLatitude(c.in); got != c.want {
			t.Errorf("formatLatitude(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestFormatLongitude(t *testing.T) {
	cases := []struct {
		in   float64
		want string
	}{
		{121.895277, "12153.72E"},
		{-121.895277, "12153.72W"},
	}
	for _, c := range cases {
		if got := formatLongitude(c.in); got != c.want {
			t.Errorf("formatLongitude(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestFormatAltitude(t *testing.T) {
	cases := []struct {
		in   float64
		want string
	}{
		{0, "000000"},
		{1234, "001234"},
		{-50, "-00050"},
	}
	for _, c := range cases {
		if got := formatAltitude(c.in); got != c.want {
			t.Errorf("formatAltitude(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestBuildFixedBeacon(t *testing.T) {
	pos := Position{
		Callsign:  "BA4QFV-O",
		Latitude:  30.876111,
		Longitude: 121.895277,
		Symbol:    "\\L",
		Comment:   "hello",
		Speed:     NoValue,
		Course:    NoValue,
		Altitude:  NoValue,
	}
	out := pos.Build()
	if len(out) != 1 {
		t.Fatalf("expected 1 packet (no Info), got %d", len(out))
	}
	want := "BA4QFV-O>" + meta.ToCall + ",TCPIP*:!3052.57N\\12153.72ELhello"
	if out[0] != want {
		t.Errorf("Build() = %q, want %q", out[0], want)
	}
}

func TestBuildMovingWithStatus(t *testing.T) {
	pos := Position{
		Callsign:  "BD8CMN-5",
		Latitude:  30.5,
		Longitude: 121.0,
		Symbol:    "/[",
		Comment:   "mobile",
		Info:      "on 145.1",
		Speed:     12, // knots
		Course:    90,
		Altitude:  328.0839895, // feet ~ 100m already in feet here
	}
	out := pos.Build()
	if len(out) != 2 {
		t.Fatalf("expected 2 packets (position + status), got %d", len(out))
	}
	if !strings.HasPrefix(out[0], "BD8CMN-5>"+meta.ToCall+",TCPIP*:!3030.00N/12100.00E[") {
		t.Errorf("position prefix wrong: %q", out[0])
	}
	if !strings.Contains(out[0], "090/012") {
		t.Errorf("course/speed extension missing: %q", out[0])
	}
	if !strings.Contains(out[0], "/A=000328") {
		t.Errorf("altitude extension wrong: %q", out[0])
	}
	wantStatus := "BD8CMN-5>" + meta.ToCall + ",TCPIP*:>on 145.1"
	if out[1] != wantStatus {
		t.Errorf("status = %q, want %q", out[1], wantStatus)
	}
}

func TestBuildTocallOverride(t *testing.T) {
	pos := Position{
		Callsign:  "BA4QFV-O",
		Latitude:  30.876111,
		Longitude: 121.895277,
		Symbol:    "\\L",
		Tocall:    "APRS",
		Comment:   "hi",
		Info:      "status",
		Speed:     NoValue,
		Course:    NoValue,
		Altitude:  NoValue,
	}
	out := pos.Build()
	if len(out) != 2 {
		t.Fatalf("expected 2 packets, got %d", len(out))
	}
	for _, p := range out {
		if !strings.HasPrefix(p, "BA4QFV-O>APRS,TCPIP*:") {
			t.Errorf("explicit Tocall not honoured: %q", p)
		}
	}
}
