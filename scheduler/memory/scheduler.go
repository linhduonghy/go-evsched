package memory

import (
	"context"
	"fmt"
	"time"

	"github.com/go-co-op/gocron/v2"
)

type Scheduler interface {
	Start()
	Shutdown(ctx context.Context) error
	AddOneTimeJob(scheduledAt time.Time, handler any, params ...any) error
	AddIntervalJob(duration time.Duration, handler any, params ...any) error
	addCronJob(cron string, handler any, params ...any) error
}
type scheduler struct {
	sched gocron.Scheduler
}

func NewScheduler(timezone string) (Scheduler, error) {
	tz, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, fmt.Errorf("failed to load timezone %w", err)
	}

	s, err := gocron.NewScheduler(gocron.WithLocation(tz))
	if err != nil {
		return nil, err
	}

	return &scheduler{
		sched: s,
	}, nil
}

func (s *scheduler) AddOneTimeJob(scheduledAt time.Time, handler any, params ...any) error {
	_, err := s.sched.NewJob(
		gocron.OneTimeJob(gocron.OneTimeJobStartDateTime(scheduledAt)),
		gocron.NewTask(handler, params...),
	)
	return err
}

func (s *scheduler) AddIntervalJob(interval time.Duration, handler any, params ...any) error {
	_, err := s.sched.NewJob(
		gocron.DurationJob(interval),
		gocron.NewTask(handler, params...),
	)
	return err
}

func (s *scheduler) addCronJob(cron string, handler any, params ...any) error {
	_, err := s.sched.NewJob(
		gocron.CronJob(cron, true),
		gocron.NewTask(handler, params...),
	)
	return err
}

func (s *scheduler) Start() {
	s.sched.Start()
}

func (s *scheduler) Shutdown(ctx context.Context) error {
	return s.sched.ShutdownWithContext(ctx)
}
