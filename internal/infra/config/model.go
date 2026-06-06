package config

// StaticConfig is the root configuration loaded from config.yaml (via viper).
type StaticConfig struct {
	// Log holds system log configuration.
	Log LogConfig `mapstructure:"log"`

	// APRS holds APRS-IS connection configuration shared by all stations.
	APRS APRSConfig `mapstructure:"aprs"`

	// Beacon holds fixed-point beaconing configuration.
	Beacon BeaconConfig `mapstructure:"beacon"`

	// Traccar holds Traccar polling configuration.
	Traccar TraccarConfig `mapstructure:"traccar"`
}

// LogConfig configures the zap logger and its rotating file sinks.
type LogConfig struct {
	File struct {
		All string `mapstructure:"all"`
		Err string `mapstructure:"err"`
	} `mapstructure:"file"`
	MaxSize    int  `mapstructure:"max_size"`
	MaxBackups int  `mapstructure:"max_backups"`
	MaxAge     int  `mapstructure:"max_age"`
	Compress   bool `mapstructure:"compress"`
}

// APRSServer is a single APRS-IS server endpoint candidate.
type APRSServer struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

// APRSConfig configures the APRS-IS uplink shared by all stations.
type APRSConfig struct {
	// Servers is the ordered list of APRS-IS servers to try.
	Servers []APRSServer `mapstructure:"servers"`
	// Software and Version are advertised in the APRS-IS login line. They let
	// the user masquerade as a particular client; when empty the application
	// defaults (meta.Name / meta.Version) are used.
	Software string `mapstructure:"software"`
	Version  string `mapstructure:"version"`
	// Tocall is the AX.25 destination ("software" TOCALL) placed in every
	// transmitted packet. When empty the application default (meta.ToCall) is
	// used. This too lets the user masquerade as a particular client.
	Tocall string `mapstructure:"tocall"`
	// RetryTimes is how many times the underlying client reconnects on link drop.
	RetryTimes int `mapstructure:"retry_times"`
}

// Station is the set of presentation attributes shared by beacons and
// Traccar devices when building an APRS position report.
type Station struct {
	Callsign string `mapstructure:"callsign"`
	// Symbol is the two-rune APRS symbol, e.g. "/[" (jogger) or "\\L".
	// In YAML a backslash table id must be written as "\\".
	Symbol  string `mapstructure:"symbol"`
	Comment string `mapstructure:"comment"`
	// Info, when non-empty, is sent as an additional APRS status report.
	Info string `mapstructure:"info"`
}

// BeaconStation is a fixed-point station reported on a timer.
type BeaconStation struct {
	Station   `mapstructure:",squash"`
	Latitude  float64 `mapstructure:"latitude"`
	Longitude float64 `mapstructure:"longitude"`
}

// BeaconConfig configures fixed-point periodic beaconing.
type BeaconConfig struct {
	Enabled bool `mapstructure:"enabled"`
	// Interval is the resend period in seconds.
	Interval int             `mapstructure:"interval"`
	Stations []BeaconStation `mapstructure:"stations"`
}

// TraccarDevice is a single Traccar tracked device mapped to a callsign.
type TraccarDevice struct {
	Station  `mapstructure:",squash"`
	URL      string `mapstructure:"url"`
	Account  string `mapstructure:"account"`
	Password string `mapstructure:"password"`
	Device   string `mapstructure:"device"`
	// SkipOld drops positions older than ExpireSeconds instead of reporting them.
	SkipOld bool `mapstructure:"skip_old"`
}

// TraccarConfig configures Traccar polling and the report triggers.
type TraccarConfig struct {
	Enabled bool `mapstructure:"enabled"`
	// Interval is the polling period in seconds.
	Interval int `mapstructure:"interval"`
	// RequestTimeout is the per-request HTTP timeout in seconds.
	RequestTimeout int `mapstructure:"request_timeout"`
	// ExpireSeconds: positions older than this are considered stale.
	ExpireSeconds int `mapstructure:"expire_seconds"`
	// MinUpdateSeconds forces a report when this long has elapsed since the last.
	MinUpdateSeconds int `mapstructure:"min_update_seconds"`
	// MaxDistanceMeters triggers a report when movement exceeds this distance.
	MaxDistanceMeters float64         `mapstructure:"max_distance_meters"`
	Devices           []TraccarDevice `mapstructure:"devices"`
}
