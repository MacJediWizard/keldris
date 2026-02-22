// Package logs provides agent log collection and management functionality.
package logs

import (
	"context"
	"sync"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// Store defines the interface for log persistence operations.
type Store interface {
	CreateAgentLogs(ctx context.Context, logs []*models.AgentLog) error
}

// Collector handles batched log collection from agents.
type Collector struct {
	store      Store
	logger     zerolog.Logger
	batchSize  int
	flushDelay time.Duration
	buffer     map[uuid.UUID][]*models.AgentLog
	mu         sync.Mutex
	stopCh     chan struct{}
	doneCh     chan struct{}
}

// NewCollector creates a new log collector.
func NewCollector(store Store, logger zerolog.Logger) *Collector {
	c := &Collector{
		store:      store,
		logger:     logger.With().Str("component", "log_collector").Logger(),
		batchSize:  100,
		flushDelay: 5 * time.Second,
		buffer:     make(map[uuid.UUID][]*models.AgentLog),
		stopCh:     make(chan struct{}),
		doneCh:     make(chan struct{}),
	}
	go c.flushLoop()
	return c
}

// Collect adds logs to the buffer for batched writing.
func (c *Collector) Collect(ctx context.Context, agentID, orgID uuid.UUID, entries []models.AgentLogEntry) error {
	logs := make([]*models.AgentLog, 0, len(entries))
	for _, entry := range entries {
		log := models.NewAgentLog(agentID, orgID, entry.Level, entry.Message)
		log.Component = entry.Component
		log.Metadata = entry.Metadata
		if !entry.Timestamp.IsZero() {
			log.Timestamp = entry.Timestamp
		}
		logs = append(logs, log)
	}

	c.mu.Lock()
	c.buffer[agentID] = append(c.buffer[agentID], logs...)
	shouldFlush := len(c.buffer[agentID]) >= c.batchSize
	c.mu.Unlock()

	if shouldFlush {
		c.flushAgent(ctx, agentID)
	}

	return nil
}

// CollectSync immediately writes logs without buffering.
func (c *Collector) CollectSync(ctx context.Context, agentID, orgID uuid.UUID, entries []models.AgentLogEntry) error {
	logs := make([]*models.AgentLog, 0, len(entries))
	for _, entry := range entries {
		log := models.NewAgentLog(agentID, orgID, entry.Level, entry.Message)
		log.Component = entry.Component
		log.Metadata = entry.Metadata
		if !entry.Timestamp.IsZero() {
			log.Timestamp = entry.Timestamp
		}
		logs = append(logs, log)
	}

	return c.store.CreateAgentLogs(ctx, logs)
}

// flushLoop periodically flushes buffered logs.
func (c *Collector) flushLoop() {
	ticker := time.NewTicker(c.flushDelay)
	defer ticker.Stop()
	defer close(c.doneCh)

	for {
		select {
		case <-ticker.C:
			c.flushAll(context.Background())
		case <-c.stopCh:
			c.flushAll(context.Background())
			return
		}
	}
}

// flushAgent flushes logs for a specific agent.
func (c *Collector) flushAgent(ctx context.Context, agentID uuid.UUID) {
	c.mu.Lock()
	logs := c.buffer[agentID]
	delete(c.buffer, agentID)
	c.mu.Unlock()

	if len(logs) == 0 {
		return
	}

	if err := c.store.CreateAgentLogs(ctx, logs); err != nil {
		c.logger.Error().Err(err).
			Str("agent_id", agentID.String()).
			Int("log_count", len(logs)).
			Msg("failed to flush agent logs")
	} else {
		c.logger.Debug().
			Str("agent_id", agentID.String()).
			Int("log_count", len(logs)).
			Msg("flushed agent logs")
	}
}

// flushAll flushes all buffered logs.
func (c *Collector) flushAll(ctx context.Context) {
	c.mu.Lock()
	agentIDs := make([]uuid.UUID, 0, len(c.buffer))
	for agentID := range c.buffer {
		agentIDs = append(agentIDs, agentID)
	}
	c.mu.Unlock()

	for _, agentID := range agentIDs {
		c.flushAgent(ctx, agentID)
	}
}

// Stop stops the collector and flushes remaining logs.
func (c *Collector) Stop() {
	close(c.stopCh)
	<-c.doneCh
}
