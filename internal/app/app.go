// Package app wires the infrastructure (config, logger, scheduler) together
// with the domain services (beacon, traccar) and registers their scheduled
// jobs. It is the composition root of the application.
package app

import (
	"context"
	"fmt"
	"time"

	"aprs-beacon/internal/aprsis"
	"aprs-beacon/internal/beacon"
	"aprs-beacon/internal/cron"
	"aprs-beacon/internal/infra/config"
	"aprs-beacon/internal/infra/logger"
	"aprs-beacon/internal/traccar"

	"github.com/APRSCN/aprsutils"
)

// App holds the constructed services and owns their lifecycle.
type App struct {
	log     *logger.ZapLogger
	manager *aprsis.Manager
}

// Init builds all services from the loaded config and registers their cron
// jobs against the (already-created) global scheduler. It does not start the
// scheduler; call cron.Start afterwards.
func Init() *App {
	cfg := config.Get()

	// The scaffold ZapLogger satisfies both our service Logger interfaces and
	// aprsutils.Logger, so a single adapter serves everywhere.
	log := &logger.ZapLogger{Logger: logger.L}

	manager := aprsis.NewManager(cfg.APRS, log, aprsutils.Logger(log))

	app := &App{log: log, manager: manager}

	app.registerBeacon(cfg.Beacon)
	app.registerTraccar(cfg.Traccar)

	return app
}

func (a *App) registerBeacon(cfg config.BeaconConfig) {
	if !cfg.Enabled {
		a.log.Info(context.Background(), "beacon: disabled")
		return
	}
	if cfg.Interval <= 0 {
		a.log.Warn(context.Background(), "beacon: non-positive interval, skipping registration")
		return
	}

	svc := beacon.NewService(cfg, a.manager, a.log)

	_, err := cron.C.Every(cfg.Interval).Seconds().
		SingletonMode().
		Do(svc.Run)
	if err != nil {
		a.log.Error(context.Background(), fmt.Sprintf("beacon: register job: %v", err))
		return
	}
	a.log.Info(context.Background(),
		fmt.Sprintf("beacon: registered %d station(s) every %ds", len(cfg.Stations), cfg.Interval))
}

func (a *App) registerTraccar(cfg config.TraccarConfig) {
	if !cfg.Enabled {
		a.log.Info(context.Background(), "traccar: disabled")
		return
	}
	if cfg.Interval <= 0 {
		a.log.Warn(context.Background(), "traccar: non-positive interval, skipping registration")
		return
	}

	api := traccar.NewClient(requestTimeout(cfg.RequestTimeout))
	svc := traccar.NewService(cfg, api, a.manager, a.log)

	_, err := cron.C.Every(cfg.Interval).Seconds().
		SingletonMode().
		Do(svc.Run)
	if err != nil {
		a.log.Error(context.Background(), fmt.Sprintf("traccar: register job: %v", err))
		return
	}
	a.log.Info(context.Background(),
		fmt.Sprintf("traccar: registered %d device(s) every %ds", len(cfg.Devices), cfg.Interval))
}

// Close releases resources (APRS-IS connections). The scheduler is stopped by
// the caller via cron.Stop.
func (a *App) Close() {
	if a.manager != nil {
		a.manager.Close()
	}
}

// requestTimeout converts a seconds count into a time.Duration, falling back to
// 5s when the configured value is non-positive.
func requestTimeout(seconds int) time.Duration {
	if seconds <= 0 {
		return 5 * time.Second
	}
	return time.Duration(seconds) * time.Second
}
