package metrics

import (
	"context"
	"time"

	"github.com/rs/zerolog"
)

// Scheduler runs nightly metrics aggregation.
type Scheduler struct {
	aggregator *Aggregator
	logger     zerolog.Logger
	stop       chan struct{}
	done       chan struct{}
}

// NewScheduler creates a new metrics aggregation scheduler.
func NewScheduler(aggregator *Aggregator, logger zerolog.Logger) *Scheduler {
	return &Scheduler{
		aggregator: aggregator,
		logger:     logger.With().Str("component", "metrics_scheduler").Logger(),
		stop:       make(chan struct{}),
		done:       make(chan struct{}),
	}
}

// Start begins the scheduler. It aggregates the previous day on startup
// to catch any missed runs, then schedules nightly aggregation at midnight UTC.
func (s *Scheduler) Start(ctx context.Context) {
	go s.run(ctx)
}

func (s *Scheduler) run(ctx context.Context) {
	defer close(s.done)

	// Aggregate previous day on startup to catch missed runs
	yesterday := time.Now().UTC().AddDate(0, 0, -1)
	s.logger.Info().Str("date", yesterday.Format("2006-01-02")).Msg("aggregating previous day metrics on startup")
	if err := s.aggregator.AggregateAllOrgs(ctx, yesterday); err != nil {
		s.logger.Error().Err(err).Msg("failed to aggregate previous day metrics on startup")
	}

	for {
		next := nextMidnightUTC()
		timer := time.NewTimer(time.Until(next))

		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-s.stop:
			timer.Stop()
			return
		case <-timer.C:
			// Aggregate the day that just ended
			completedDay := next.AddDate(0, 0, -1)
			s.logger.Info().Str("date", completedDay.Format("2006-01-02")).Msg("running nightly metrics aggregation")
			if err := s.aggregator.AggregateAllOrgs(ctx, completedDay); err != nil {
				s.logger.Error().Err(err).Msg("nightly metrics aggregation failed")
			}
		}
	}
}

// Stop signals the scheduler to stop and waits for it to finish.
func (s *Scheduler) Stop() {
	close(s.stop)
	<-s.done
}

// nextMidnightUTC returns the next midnight UTC time.
func nextMidnightUTC() time.Time {
	now := time.Now().UTC()
	next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
	return next
}
