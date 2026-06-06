// Package traccar polls a Traccar server and reports moving stations to APRS-IS.
//
// For each configured device it fetches the latest position and decides whether
// to transmit, based on two triggers:
//   - movement: distance from the last reported point >= MaxDistanceMeters
//   - time:     elapsed since the last report          >= MinUpdateSeconds
//
// The first observation of a device is always reported, which both seeds the
// per-device state and avoids the legacy "distance from zero" bug.
package traccar

import (
	"context"
	"fmt"
	"sync"
	"time"

	"aprs-beacon/internal/infra/config"
	"aprs-beacon/internal/packet"

	"github.com/APRSCN/aprsutils"
)

// Logger is the logging subset the service needs.
type Logger interface {
	Debug(context.Context, ...any)
	Info(context.Context, ...any)
	Warn(context.Context, ...any)
	Error(context.Context, ...any)
}

// Sender abstracts the APRS-IS uplink (implemented by *aprsis.Manager).
type Sender interface {
	SendPosition(pos packet.Position) error
}

// poller is the subset of *Client the service depends on (eases testing).
type poller interface {
	LatestPosition(ctx context.Context, baseURL, account, password, deviceID string) (Position, error)
}

// deviceState tracks the last reported fix for a device.
type deviceState struct {
	lastLat    float64
	lastLon    float64
	lastReport time.Time
	seen       bool
}

// Service polls Traccar and conditionally reports positions.
type Service struct {
	cfg   config.TraccarConfig
	api   poller
	tx    Sender
	log   Logger
	nowFn func() time.Time

	mu    sync.Mutex
	state map[string]*deviceState // keyed by callsign
}

// NewService constructs the Traccar reporting service.
func NewService(cfg config.TraccarConfig, api poller, tx Sender, log Logger) *Service {
	return &Service{
		cfg:   cfg,
		api:   api,
		tx:    tx,
		log:   log,
		nowFn: time.Now,
		state: make(map[string]*deviceState),
	}
}

// Run polls every configured device once and reports those that meet a trigger.
// Intended to be invoked periodically by the scheduler. Per-device failures are
// logged and isolated.
func (s *Service) Run() {
	ctx, cancel := context.WithTimeout(context.Background(),
		time.Duration(s.cfg.RequestTimeout)*time.Second)
	defer cancel()

	for _, dev := range s.cfg.Devices {
		s.handleDevice(ctx, dev)
	}
}

func (s *Service) handleDevice(ctx context.Context, dev config.TraccarDevice) {
	pos, err := s.api.LatestPosition(ctx, dev.URL, dev.Account, dev.Password, dev.Device)
	if err != nil {
		s.log.Error(ctx, fmt.Sprintf("traccar: poll %s: %v", dev.Callsign, err))
		return
	}

	now := s.nowFn()

	// Drop stale fixes if requested.
	if dev.SkipOld && s.cfg.ExpireSeconds > 0 {
		age := now.Sub(pos.FixTime)
		if age > time.Duration(s.cfg.ExpireSeconds)*time.Second {
			s.log.Debug(ctx, fmt.Sprintf("traccar: %s fix is stale (%s old), skipping", dev.Callsign, age.Truncate(time.Second)))
			return
		}
	}

	st := s.deviceState(dev.Callsign)

	reason, report := s.shouldReport(st, pos, now)
	if !report {
		return
	}

	if err := s.tx.SendPosition(s.buildPosition(dev, pos)); err != nil {
		s.log.Error(ctx, fmt.Sprintf("traccar: send %s: %v", dev.Callsign, err))
		return
	}

	s.recordReport(st, pos, now)
	s.log.Info(ctx, fmt.Sprintf("traccar: %s sent (%s)", dev.Callsign, reason))
}

// shouldReport applies the movement/time triggers. It does not mutate state.
func (s *Service) shouldReport(st *deviceState, pos Position, now time.Time) (reason string, report bool) {
	if !st.seen {
		return "first fix", true
	}

	distKM := aprsutils.CalculateDistanceHaversine(st.lastLat, st.lastLon, pos.Latitude, pos.Longitude)
	distM := distKM * 1000
	if distM >= s.cfg.MaxDistanceMeters {
		return fmt.Sprintf("moved %.0fm", distM), true
	}

	elapsed := now.Sub(st.lastReport)
	if elapsed >= time.Duration(s.cfg.MinUpdateSeconds)*time.Second {
		return fmt.Sprintf("elapsed %s", elapsed.Truncate(time.Second)), true
	}
	return "", false
}

// buildPosition maps a Traccar fix and device presentation into a packet.
// Traccar reports speed in knots and altitude in metres, matching what the
// packet builder expects (speed in knots, altitude converted to feet here).
func (s *Service) buildPosition(dev config.TraccarDevice, pos Position) packet.Position {
	return packet.Position{
		Callsign:  dev.Callsign,
		Latitude:  pos.Latitude,
		Longitude: pos.Longitude,
		Symbol:    dev.Symbol,
		Comment:   dev.Comment,
		Info:      dev.Info,
		Speed:     pos.Speed,
		Course:    pos.Course,
		Altitude:  packet.MetersToFeet(pos.Altitude),
	}
}

func (s *Service) deviceState(callsign string) *deviceState {
	s.mu.Lock()
	defer s.mu.Unlock()
	st, ok := s.state[callsign]
	if !ok {
		st = &deviceState{}
		s.state[callsign] = st
	}
	return st
}

func (s *Service) recordReport(st *deviceState, pos Position, now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	st.lastLat = pos.Latitude
	st.lastLon = pos.Longitude
	st.lastReport = now
	st.seen = true
}
