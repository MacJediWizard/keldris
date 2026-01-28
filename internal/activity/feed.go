// Package activity provides real-time activity event capture and fan-out.
package activity

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
)

// Store defines the interface for activity event persistence operations.
type Store interface {
	CreateActivityEvent(ctx context.Context, event *models.ActivityEvent) error
	GetActivityEvents(ctx context.Context, orgID uuid.UUID, filter models.ActivityEventFilter) ([]*models.ActivityEvent, error)
	GetActivityEventCount(ctx context.Context, orgID uuid.UUID, filter models.ActivityEventFilter) (int, error)
}

// Client represents a connected WebSocket client.
type Client struct {
	id     uuid.UUID
	orgID  uuid.UUID
	userID uuid.UUID
	conn   *websocket.Conn
	send   chan *models.ActivityEvent
	feed   *Feed
	filter *ClientFilter
	mu     sync.Mutex
}

// ClientFilter holds the filter preferences for a connected client.
type ClientFilter struct {
	Categories []models.ActivityEventCategory `json:"categories,omitempty"`
	Types      []models.ActivityEventType     `json:"types,omitempty"`
	AgentIDs   []uuid.UUID                    `json:"agent_ids,omitempty"`
	UserIDs    []uuid.UUID                    `json:"user_ids,omitempty"`
}

// Matches checks if an event matches the client's filter.
func (f *ClientFilter) Matches(event *models.ActivityEvent) bool {
	if f == nil {
		return true
	}

	// Check categories filter
	if len(f.Categories) > 0 {
		found := false
		for _, cat := range f.Categories {
			if cat == event.Category {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check types filter
	if len(f.Types) > 0 {
		found := false
		for _, t := range f.Types {
			if t == event.Type {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check agent filter
	if len(f.AgentIDs) > 0 && event.AgentID != nil {
		found := false
		for _, id := range f.AgentIDs {
			if id == *event.AgentID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check user filter
	if len(f.UserIDs) > 0 && event.UserID != nil {
		found := false
		for _, id := range f.UserIDs {
			if id == *event.UserID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// Config holds configuration for the Feed.
type Config struct {
	// PingInterval is how often to send ping messages to clients.
	PingInterval time.Duration
	// WriteTimeout is the timeout for writing to a client.
	WriteTimeout time.Duration
	// ReadTimeout is the timeout for reading from a client.
	ReadTimeout time.Duration
	// MaxMessageSize is the maximum size of a message from a client.
	MaxMessageSize int64
	// SendBufferSize is the size of the send buffer per client.
	SendBufferSize int
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		PingInterval:   30 * time.Second,
		WriteTimeout:   10 * time.Second,
		ReadTimeout:    60 * time.Second,
		MaxMessageSize: 512,
		SendBufferSize: 256,
	}
}

// Feed manages activity event broadcasting to connected clients.
type Feed struct {
	config   Config
	store    Store
	logger   zerolog.Logger
	upgrader websocket.Upgrader

	clients    map[uuid.UUID]*Client
	clientsMu  sync.RWMutex
	orgClients map[uuid.UUID]map[uuid.UUID]*Client // orgID -> clientID -> client

	broadcast chan *models.ActivityEvent
	register  chan *Client
	unregister chan *Client

	done chan struct{}
	wg   sync.WaitGroup
}

// NewFeed creates a new Feed with the given configuration.
func NewFeed(store Store, cfg Config, logger zerolog.Logger) *Feed {
	return &Feed{
		config: cfg,
		store:  store,
		logger: logger.With().Str("component", "activity_feed").Logger(),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for now
			},
		},
		clients:    make(map[uuid.UUID]*Client),
		orgClients: make(map[uuid.UUID]map[uuid.UUID]*Client),
		broadcast:  make(chan *models.ActivityEvent, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		done:       make(chan struct{}),
	}
}

// Start begins processing events and client management.
func (f *Feed) Start() {
	f.wg.Add(1)
	go f.run()
	f.logger.Info().Msg("activity feed started")
}

// Stop stops the feed and closes all client connections.
func (f *Feed) Stop() {
	close(f.done)
	f.wg.Wait()
	f.logger.Info().Msg("activity feed stopped")
}

// run is the main event loop.
func (f *Feed) run() {
	defer f.wg.Done()

	for {
		select {
		case <-f.done:
			f.closeAllClients()
			return

		case client := <-f.register:
			f.addClient(client)

		case client := <-f.unregister:
			f.removeClient(client)

		case event := <-f.broadcast:
			f.broadcastEvent(event)
		}
	}
}

// addClient adds a client to the feed.
func (f *Feed) addClient(client *Client) {
	f.clientsMu.Lock()
	defer f.clientsMu.Unlock()

	f.clients[client.id] = client

	if _, ok := f.orgClients[client.orgID]; !ok {
		f.orgClients[client.orgID] = make(map[uuid.UUID]*Client)
	}
	f.orgClients[client.orgID][client.id] = client

	f.logger.Debug().
		Str("client_id", client.id.String()).
		Str("org_id", client.orgID.String()).
		Str("user_id", client.userID.String()).
		Msg("client connected")
}

// removeClient removes a client from the feed.
func (f *Feed) removeClient(client *Client) {
	f.clientsMu.Lock()
	defer f.clientsMu.Unlock()

	if _, ok := f.clients[client.id]; !ok {
		return
	}

	delete(f.clients, client.id)

	if orgClients, ok := f.orgClients[client.orgID]; ok {
		delete(orgClients, client.id)
		if len(orgClients) == 0 {
			delete(f.orgClients, client.orgID)
		}
	}

	close(client.send)

	f.logger.Debug().
		Str("client_id", client.id.String()).
		Str("org_id", client.orgID.String()).
		Msg("client disconnected")
}

// closeAllClients closes all client connections.
func (f *Feed) closeAllClients() {
	f.clientsMu.Lock()
	defer f.clientsMu.Unlock()

	for _, client := range f.clients {
		close(client.send)
	}
	f.clients = make(map[uuid.UUID]*Client)
	f.orgClients = make(map[uuid.UUID]map[uuid.UUID]*Client)
}

// broadcastEvent sends an event to all clients in the same organization.
func (f *Feed) broadcastEvent(event *models.ActivityEvent) {
	f.clientsMu.RLock()
	orgClients := f.orgClients[event.OrgID]
	f.clientsMu.RUnlock()

	for _, client := range orgClients {
		if client.filter.Matches(event) {
			select {
			case client.send <- event:
			default:
				// Client's send buffer is full, skip
				f.logger.Warn().
					Str("client_id", client.id.String()).
					Msg("client send buffer full, dropping event")
			}
		}
	}
}

// Publish publishes an event to the feed and optionally persists it.
func (f *Feed) Publish(ctx context.Context, event *models.ActivityEvent) error {
	// Persist to database
	if f.store != nil {
		if err := f.store.CreateActivityEvent(ctx, event); err != nil {
			f.logger.Error().Err(err).
				Str("event_type", string(event.Type)).
				Msg("failed to persist activity event")
			return err
		}
	}

	// Broadcast to connected clients
	select {
	case f.broadcast <- event:
	default:
		f.logger.Warn().Msg("broadcast buffer full, dropping event")
	}

	return nil
}

// PublishBackupStarted publishes a backup started event.
func (f *Feed) PublishBackupStarted(ctx context.Context, orgID, agentID uuid.UUID, agentName, scheduleName string) error {
	event := models.NewActivityEvent(orgID, models.ActivityEventBackupStarted, "Backup Started", "Backup job started for "+scheduleName)
	event.SetAgent(agentID, agentName)
	event.SetResource("schedule", uuid.Nil, scheduleName)
	return f.Publish(ctx, event)
}

// PublishBackupCompleted publishes a backup completed event.
func (f *Feed) PublishBackupCompleted(ctx context.Context, orgID, agentID uuid.UUID, agentName, scheduleName string, sizeBytes int64, duration time.Duration) error {
	event := models.NewActivityEvent(orgID, models.ActivityEventBackupCompleted, "Backup Completed", "Backup job completed successfully")
	event.SetAgent(agentID, agentName)
	event.SetResource("schedule", uuid.Nil, scheduleName)
	event.SetMetadata(map[string]any{
		"size_bytes": sizeBytes,
		"duration":   duration.String(),
	})
	return f.Publish(ctx, event)
}

// PublishBackupFailed publishes a backup failed event.
func (f *Feed) PublishBackupFailed(ctx context.Context, orgID, agentID uuid.UUID, agentName, scheduleName, errorMsg string) error {
	event := models.NewActivityEvent(orgID, models.ActivityEventBackupFailed, "Backup Failed", "Backup job failed: "+errorMsg)
	event.SetAgent(agentID, agentName)
	event.SetResource("schedule", uuid.Nil, scheduleName)
	event.SetMetadata(map[string]any{
		"error": errorMsg,
	})
	return f.Publish(ctx, event)
}

// PublishAgentConnected publishes an agent connected event.
func (f *Feed) PublishAgentConnected(ctx context.Context, orgID, agentID uuid.UUID, agentName string) error {
	event := models.NewActivityEvent(orgID, models.ActivityEventAgentConnected, "Agent Connected", agentName+" is now online")
	event.SetAgent(agentID, agentName)
	return f.Publish(ctx, event)
}

// PublishAgentDisconnected publishes an agent disconnected event.
func (f *Feed) PublishAgentDisconnected(ctx context.Context, orgID, agentID uuid.UUID, agentName string) error {
	event := models.NewActivityEvent(orgID, models.ActivityEventAgentDisconnected, "Agent Disconnected", agentName+" went offline")
	event.SetAgent(agentID, agentName)
	return f.Publish(ctx, event)
}

// PublishUserLogin publishes a user login event.
func (f *Feed) PublishUserLogin(ctx context.Context, orgID, userID uuid.UUID, userName string) error {
	event := models.NewActivityEvent(orgID, models.ActivityEventUserLogin, "User Login", userName+" logged in")
	event.SetUser(userID, userName)
	return f.Publish(ctx, event)
}

// PublishAlertTriggered publishes an alert triggered event.
func (f *Feed) PublishAlertTriggered(ctx context.Context, orgID uuid.UUID, alertTitle, alertSeverity string) error {
	event := models.NewActivityEvent(orgID, models.ActivityEventAlertTriggered, "Alert Triggered", alertTitle)
	event.SetMetadata(map[string]any{
		"severity": alertSeverity,
	})
	return f.Publish(ctx, event)
}

// HandleWebSocket handles a WebSocket connection upgrade and client management.
func (f *Feed) HandleWebSocket(w http.ResponseWriter, r *http.Request, orgID, userID uuid.UUID) {
	conn, err := f.upgrader.Upgrade(w, r, nil)
	if err != nil {
		f.logger.Error().Err(err).Msg("failed to upgrade websocket connection")
		return
	}

	client := &Client{
		id:     uuid.New(),
		orgID:  orgID,
		userID: userID,
		conn:   conn,
		send:   make(chan *models.ActivityEvent, f.config.SendBufferSize),
		feed:   f,
		filter: &ClientFilter{},
	}

	f.register <- client

	// Start read and write pumps
	go client.writePump()
	go client.readPump()
}

// GetClientCount returns the number of connected clients for an organization.
func (f *Feed) GetClientCount(orgID uuid.UUID) int {
	f.clientsMu.RLock()
	defer f.clientsMu.RUnlock()

	if orgClients, ok := f.orgClients[orgID]; ok {
		return len(orgClients)
	}
	return 0
}

// GetTotalClientCount returns the total number of connected clients.
func (f *Feed) GetTotalClientCount() int {
	f.clientsMu.RLock()
	defer f.clientsMu.RUnlock()
	return len(f.clients)
}

// readPump reads messages from the client.
func (c *Client) readPump() {
	defer func() {
		c.feed.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(c.feed.config.MaxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(c.feed.config.ReadTimeout))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(c.feed.config.ReadTimeout))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.feed.logger.Debug().Err(err).Msg("websocket read error")
			}
			break
		}

		// Parse filter update message
		var filterUpdate struct {
			Type   string       `json:"type"`
			Filter ClientFilter `json:"filter"`
		}
		if err := json.Unmarshal(message, &filterUpdate); err == nil && filterUpdate.Type == "filter" {
			c.mu.Lock()
			c.filter = &filterUpdate.Filter
			c.mu.Unlock()
		}
	}
}

// writePump writes messages to the client.
func (c *Client) writePump() {
	ticker := time.NewTicker(c.feed.config.PingInterval)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case event, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(c.feed.config.WriteTimeout))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}

			data, _ := json.Marshal(event)
			w.Write(data)

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(c.feed.config.WriteTimeout))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
