// Package aprsis manages APRS-IS uplink connections.
//
// It wraps github.com/APRSCN/aprsutils/client, giving each callsign its own
// lazily-established, reused connection. The aprsutils client owns
// reconnection and heartbeat internally, so the manager only needs to create,
// connect once, cache and finally close clients.
package aprsis

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"aprs-beacon/internal/infra/config"
	"aprs-beacon/internal/meta"
	"aprs-beacon/internal/packet"

	"github.com/APRSCN/aprsutils"
	"github.com/APRSCN/aprsutils/client"
)

// Logger is the subset of logging the manager needs. It matches both the
// scaffold logger.Logger and aprsutils.Logger, keeping wiring trivial.
type Logger interface {
	Debug(context.Context, ...any)
	Info(context.Context, ...any)
	Warn(context.Context, ...any)
	Error(context.Context, ...any)
}

// Manager owns one APRS-IS client per callsign and serialises sends per client.
type Manager struct {
	cfg    config.APRSConfig
	log    Logger
	aprLog aprsutils.Logger

	mu      sync.Mutex
	clients map[string]*client.Client
}

// NewManager builds a manager from APRS config. aprLog is passed to the
// underlying aprsutils client; it may be the same value as log if it satisfies
// aprsutils.Logger, otherwise pass aprsutils.NewLogger().
//
// Empty Software/Version/Tocall fields are filled with the application
// defaults from the meta package, so the rest of the manager can use them
// unconditionally.
func NewManager(cfg config.APRSConfig, log Logger, aprLog aprsutils.Logger) *Manager {
	if cfg.Software == "" {
		cfg.Software = meta.Name
	}
	if cfg.Version == "" {
		cfg.Version = meta.Version
	}
	if cfg.Tocall == "" {
		cfg.Tocall = meta.ToCall
	}
	return &Manager{
		cfg:     cfg,
		log:     log,
		aprLog:  aprLog,
		clients: make(map[string]*client.Client),
	}
}

// client returns a connected client for the callsign, creating and connecting
// it on first use. Subsequent calls reuse the cached connection.
func (m *Manager) client(callsign string) (*client.Client, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if c, ok := m.clients[callsign]; ok {
		return c, nil
	}

	if len(m.cfg.Servers) == 0 {
		return nil, fmt.Errorf("aprsis: no servers configured")
	}
	if !aprsutils.ValidateCallsign(callsign) {
		return nil, fmt.Errorf("aprsis: invalid callsign %q", callsign)
	}

	passcode := strconv.Itoa(aprsutils.Passcode(callsign))

	opts := []client.Option{
		client.WithSoftwareAndVersion(m.cfg.Software, m.cfg.Version),
		client.WithRetryTimes(m.cfg.RetryTimes),
	}
	if m.aprLog != nil {
		opts = append(opts, client.WithLogger(m.aprLog))
	}

	// Try each configured server until one connects.
	var lastErr error
	for _, srv := range m.cfg.Servers {
		c := client.NewClient(
			callsign,
			passcode,
			client.Fullfeed,
			client.TCP,
			srv.Host,
			srv.Port,
			opts...,
		)
		if err := c.Connect(); err != nil {
			lastErr = err
			m.log.Warn(context.Background(),
				fmt.Sprintf("aprsis: connect %s via %s:%d failed: %v", callsign, srv.Host, srv.Port, err))
			c.Close()
			continue
		}
		m.log.Info(context.Background(),
			fmt.Sprintf("aprsis: %s connected via %s", callsign, c.RemoteAddr()))
		m.clients[callsign] = c
		return c, nil
	}
	return nil, fmt.Errorf("aprsis: all servers failed for %s: %w", callsign, lastErr)
}

// Send transmits one or more raw packet strings for the given callsign,
// (re)connecting as needed. On send failure the cached client is dropped so the
// next call reconnects fresh.
func (m *Manager) Send(callsign string, packets []string) error {
	c, err := m.client(callsign)
	if err != nil {
		return err
	}
	for _, p := range packets {
		if err := c.SendPacket(p); err != nil {
			m.drop(callsign)
			return fmt.Errorf("aprsis: send for %s: %w", callsign, err)
		}
	}
	return nil
}

// SendPosition builds and sends a position (and optional status) report. The
// configured Tocall is applied unless the position already carries one, so the
// masquerade identity is set in a single place.
func (m *Manager) SendPosition(pos packet.Position) error {
	if pos.Tocall == "" {
		pos.Tocall = m.cfg.Tocall
	}
	return m.Send(pos.Callsign, pos.Build())
}

// drop removes and closes the cached client for a callsign.
func (m *Manager) drop(callsign string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if c, ok := m.clients[callsign]; ok {
		c.Close()
		delete(m.clients, callsign)
	}
}

// Close tears down all cached clients.
func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for callsign, c := range m.clients {
		c.Close()
		delete(m.clients, callsign)
	}
}
