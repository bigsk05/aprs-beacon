package cron

import (
	"time"

	"github.com/go-co-op/gocron"
)

// C is the global scheduler. It is created by Init but not started until Start
// is called, so that application modules can register their jobs first.
var C *gocron.Scheduler

// Init creates the global scheduler (without starting it).
func Init() {
	C = gocron.NewScheduler(time.Local)
}

// Start begins executing scheduled jobs asynchronously.
func Start() {
	C.StartAsync()
}

// Stop halts the scheduler.
func Stop() {
	if C != nil {
		C.Stop()
	}
}
