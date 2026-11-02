package server

import (
	"github.com/lynxland/lynx/contrib/schedule"
)

func NewScheduler() (*schedule.Scheduler, error) {
	return schedule.NewScheduler([]schedule.Task{}, schedule.WithDebugEnabled())
}
