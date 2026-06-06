// Package packet builds APRS-IS report strings.
//
// aprsutils parses packets but does not build them, so this package owns the
// formatting of an outgoing uncompressed position report (and an optional
// status report) ready to hand to client.SendPacket.
package packet

import (
	"fmt"
	"strings"

	"aprs-beacon/internal/meta"
)

// NoValue marks a numeric field (Speed/Course/Altitude) as absent, so it is
// omitted from the generated packet. Use it for fixed beacons.
const NoValue = -1

// Position describes an outgoing APRS position report.
type Position struct {
	// Callsign is the source callsign with optional SSID, e.g. "BD8CMN-5".
	Callsign string
	// Latitude and Longitude are in decimal degrees (north/east positive).
	Latitude  float64
	Longitude float64
	// Symbol is the two-rune APRS symbol: table id + symbol code, e.g. "/[" or "\\L".
	Symbol string
	// Tocall is the AX.25 destination (TOCALL/"software") placed after ">".
	// It lets the user masquerade as a particular client. When empty it
	// defaults to meta.ToCall.
	Tocall string
	// Comment is appended to the position report.
	Comment string
	// Info, when non-empty, produces an additional status report packet.
	Info string

	// Speed (knots), Course (degrees) and Altitude (feet) are optional.
	// Set any of them to NoValue to omit. Course/Speed are emitted together
	// as the APRS "CSE/SPD" extension whenever either is present.
	Speed    float64
	Course   float64
	Altitude float64
}

// tocall returns the AX.25 destination to use: the configured Tocall, or
// meta.ToCall when none was set. This is the TOCALL the receiving network sees
// (and what identifies the sending "software").
func (p Position) tocall() string {
	if p.Tocall != "" {
		return p.Tocall
	}
	return meta.ToCall
}

// symbolRunes splits a two-rune symbol into (table, code), tolerating malformed
// input by falling back to the primary table and a dot.
func symbolRunes(symbol string) (table, code rune) {
	r := []rune(symbol)
	switch len(r) {
	case 0:
		return '/', '.'
	case 1:
		return '/', r[0]
	default:
		return r[0], r[1]
	}
}

// courseSpeedExtension renders the optional "CSE/SPD" field. It is included
// whenever either course or speed is present; the missing component is sent
// as zero, per common APRS practice.
func (p Position) courseSpeedExtension() string {
	hasCourse := p.Course != NoValue
	hasSpeed := p.Speed != NoValue
	if !hasCourse && !hasSpeed {
		return ""
	}
	course := 0.0
	if hasCourse {
		course = p.Course
	}
	speed := 0.0
	if hasSpeed {
		speed = p.Speed
	}
	return fmt.Sprintf("%s/%s", formatCourse(course), formatSpeed(speed))
}

// altitudeExtension renders the optional "/A=NNNNNN" field.
func (p Position) altitudeExtension() string {
	if p.Altitude == NoValue {
		return ""
	}
	return "/A=" + formatAltitude(p.Altitude)
}

// Build renders the position report and, if Info is set, a trailing status
// report. The returned slice contains one or two ready-to-send packet strings
// (without the trailing CRLF, which the client adds).
func (p Position) Build() []string {
	table, code := symbolRunes(p.Symbol)
	tocall := p.tocall()

	// Uncompressed position: !DDMM.mmN<table>DDDMM.mmE<code>...
	var b strings.Builder
	b.WriteString(p.Callsign)
	b.WriteByte('>')
	b.WriteString(tocall)
	b.WriteString(",TCPIP*:!")
	b.WriteString(formatLatitude(p.Latitude))
	b.WriteRune(table)
	b.WriteString(formatLongitude(p.Longitude))
	b.WriteRune(code)
	b.WriteString(p.courseSpeedExtension())
	b.WriteString(p.altitudeExtension())
	b.WriteString(p.Comment)

	packets := []string{b.String()}

	if p.Info != "" {
		packets = append(packets, fmt.Sprintf("%s>%s,TCPIP*:>%s", p.Callsign, tocall, p.Info))
	}
	return packets
}
