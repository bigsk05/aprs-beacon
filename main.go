package main

import (
	"os"
	"os/signal"
	"syscall"

	"aprs-beacon/internal/app"
	"aprs-beacon/internal/cron"
	"aprs-beacon/internal/infra/config"
	"aprs-beacon/internal/infra/logger"
)

func main() {
	// Load static config (also installs the SIGHUP reload handler).
	config.Init()
	defer config.Cleanup()

	// Init logger.
	logger.Init()
	defer logger.Cleanup()

	// Create the scheduler (jobs are registered before it starts).
	cron.Init()

	// Compose the application: APRS-IS manager, beacon and traccar services,
	// and their scheduled jobs.
	application := app.Init()
	defer application.Close()

	// Start running scheduled jobs.
	cron.Start()
	defer cron.Stop()

	logger.L.Info("aprs-beacon started")

	// Block until an interrupt/termination signal arrives, then shut down via
	// the deferred Close/Stop/Cleanup calls.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	logger.L.Info("aprs-beacon shutting down")
}
