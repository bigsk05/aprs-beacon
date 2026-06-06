package packet

import (
	"fmt"
	"math"
)

// Unit conversion factors.
const (
	knotsPerKPH  = 1.0 / 1.852 // km/h -> knots
	knotsPerMPH  = 0.868976    // mph  -> knots
	feetPerMeter = 3.280839895 // m    -> feet
)

// KPHToKnots converts kilometres per hour to knots.
func KPHToKnots(kph float64) float64 { return kph * knotsPerKPH }

// MPHToKnots converts miles per hour to knots.
func MPHToKnots(mph float64) float64 { return mph * knotsPerMPH }

// MetersToFeet converts metres to feet.
func MetersToFeet(m float64) float64 { return m * feetPerMeter }

// formatLatitude renders a latitude as APRS DDMM.mmN/S (8 chars), e.g. "3052.56N".
func formatLatitude(lat float64) string {
	dir := "N"
	if lat < 0 {
		dir = "S"
	}
	lat = math.Abs(lat)
	deg := int(lat)
	mins := (lat - float64(deg)) * 60
	return fmt.Sprintf("%02d%05.2f%s", deg, mins, dir)
}

// formatLongitude renders a longitude as APRS DDDMM.mmE/W (9 chars), e.g. "12156.92E".
func formatLongitude(lon float64) string {
	dir := "E"
	if lon < 0 {
		dir = "W"
	}
	lon = math.Abs(lon)
	deg := int(lon)
	mins := (lon - float64(deg)) * 60
	return fmt.Sprintf("%03d%05.2f%s", deg, mins, dir)
}

// formatCourse renders a course in degrees as a 3-digit field (000-360).
// Values outside [0,360) are normalised to 000.
func formatCourse(deg float64) string {
	if deg < 0 || deg >= 360 {
		deg = 0
	}
	return fmt.Sprintf("%03d", int(math.Round(deg)))
}

// formatSpeed renders a speed in knots as a 3-digit field, clamped to 999.
func formatSpeed(knots float64) string {
	v := int(math.Round(math.Abs(knots)))
	if v > 999 {
		v = 999
	}
	return fmt.Sprintf("%03d", v)
}

// formatAltitude renders an altitude in feet as the 6-char APRS A= field value,
// clamped to the representable range.
func formatAltitude(feet float64) string {
	if feet > 999999 {
		feet = 999999
	}
	if feet < -99999 {
		feet = -99999
	}
	v := int(math.Round(feet))
	if v < 0 {
		return fmt.Sprintf("-%05d", -v)
	}
	return fmt.Sprintf("%06d", v)
}
