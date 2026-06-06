// Package beacon reports fixed-point stations on a timer.
//
// A beacon has a static position; the service simply re-sends every configured
// station's position report each time it runs. There is no change detection.
package beacon

import (
	"context"
	"fmt"

	"aprs-beacon/internal/infra/config"
	"aprs-beacon/internal/packet"
)

// Logger is the logging subset the service needs.
type Logger interface {
	Info(context.Context, ...any)
	Error(context.Context, ...any)
}

// Sender abstracts the APRS-IS uplink (implemented by *aprsis.Manager).
type Sender interface {
	SendPosition(pos packet.Position) error
}

// Service sends fixed-point beacons.
type Service struct {
	cfg config.BeaconConfig
	tx  Sender
	log Logger
}

// NewService constructs the beacon service.
func NewService(cfg config.BeaconConfig, tx Sender, log Logger) *Service {
	return &Service{cfg: cfg, tx: tx, log: log}
}

// Run sends every configured beacon once. It is intended to be invoked
// periodically by the scheduler. A failure on one station is logged and does
// not stop the others.
func (s *Service) Run() {
	ctx := context.Background()
	for _, st := range s.cfg.Stations {
		pos := packet.Position{
			Callsign:  st.Callsign,
			Latitude:  st.Latitude,
			Longitude: st.Longitude,
			Symbol:    st.Symbol,
			Comment:   st.Comment,
			Info:      st.Info,
			Speed:     packet.NoValue,
			Course:    packet.NoValue,
			Altitude:  packet.NoValue,
		}
		if err := s.tx.SendPosition(pos); err != nil {
			s.log.Error(ctx, fmt.Sprintf("beacon: send %s failed: %v", st.Callsign, err))
			continue
		}
		s.log.Info(ctx, fmt.Sprintf("beacon: %s sent", st.Callsign))
	}
}
