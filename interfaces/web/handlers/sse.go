package handlers

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"spaudit/domain/jobs"
	"spaudit/interfaces/web/presenters"
	"spaudit/logging"
)

// SSEClient represents a connected Server-Sent Events client.
type SSEClient struct {
	id       string
	writer   http.ResponseWriter
	flusher  http.Flusher
	done     chan struct{}
	lastSent time.Time
}

// SSEManager manages Server-Sent Events connections and real-time broadcasting.
// Handles job status updates and live UI refreshes.
type SSEManager struct {
	clients        map[string]*SSEClient
	mu             sync.RWMutex
	logger         *logging.Logger
	toastPresenter *presenters.ToastPresenter
}

// NewSSEManager creates a new SSE connection manager with cleanup routines.
func NewSSEManager() *SSEManager {
	manager := &SSEManager{
		clients:        make(map[string]*SSEClient),
		logger:         logging.Default().WithComponent("sse_manager"),
		toastPresenter: presenters.NewToastPresenter(),
	}

	// Start cleanup routine for stale connections
	go manager.cleanupRoutine()

	return manager
}

// AddClient adds a new SSE client connection
func (s *SSEManager) AddClient(clientID string, w http.ResponseWriter) *SSEClient {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Cache-Control")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering

	flusher, ok := w.(http.Flusher)
	if !ok {
		s.logger.Error("Response writer does not support flushing")
		return nil
	}

	// Immediate flush to establish connection
	flusher.Flush()

	client := &SSEClient{
		id:       clientID,
		writer:   w,
		flusher:  flusher,
		done:     make(chan struct{}),
		lastSent: time.Now(),
	}

	s.mu.Lock()
	s.clients[clientID] = client
	s.mu.Unlock()

	s.logger.Info("SSE client connected", "client_id", clientID, "total_clients", len(s.clients))

	// Send initial connection message as comment (won't trigger HTMX)
	s.sendToClient(client, "connected", fmt.Sprintf("Connected client %s", clientID))

	return client
}

// RemoveClient removes an SSE client connection
func (s *SSEManager) RemoveClient(clientID string) {
	s.mu.Lock()
	client, exists := s.clients[clientID]
	if exists {
		delete(s.clients, clientID)
	}
	s.mu.Unlock()

	if exists {
		// Close channel outside of lock to prevent double-close panic
		select {
		case <-client.done:
			// Already closed
		default:
			close(client.done)
		}
		s.logger.Info("SSE client disconnected", "client_id", clientID)
	}
}

// BroadcastJobUpdate broadcasts a job status update to all connected clients
func (s *SSEManager) BroadcastJobUpdate(jobID string, data string) {
	// Copy clients list to avoid holding lock during I/O
	s.mu.RLock()
	clientList := make(map[string]*SSEClient, len(s.clients))
	for id, client := range s.clients {
		clientList[id] = client
	}
	s.mu.RUnlock()

	event := fmt.Sprintf("job:%s:updated", jobID)
	failedClients := []string{}

	for clientID, client := range clientList {
		if err := s.sendToClient(client, event, data); err != nil {
			s.logger.Warn("Failed to send job update to client",
				"client_id", clientID,
				"job_id", jobID,
				"error", err)
			failedClients = append(failedClients, clientID)
		}
	}

	// Remove failed clients after broadcasting
	for _, clientID := range failedClients {
		s.RemoveClient(clientID)
	}

	s.logger.Info("Broadcasted job update", "job_id", jobID, "clients", len(clientList))
}

// BroadcastJobListUpdate broadcasts that the job list has changed
func (s *SSEManager) BroadcastJobListUpdate() {
	// Copy clients list to avoid holding lock during I/O
	s.mu.RLock()
	if len(s.clients) == 0 {
		s.mu.RUnlock()
		s.logger.Debug("No SSE clients connected, skipping job list update broadcast")
		return
	}

	clientList := make(map[string]*SSEClient, len(s.clients))
	for id, client := range s.clients {
		clientList[id] = client
	}
	s.mu.RUnlock()

	successCount := 0
	failedClients := []string{}
	message := `{"action": "refresh", "timestamp": "` + time.Now().Format(time.RFC3339) + `"}`

	for clientID, client := range clientList {
		if err := s.sendToClient(client, "jobs-updated", message); err != nil {
			s.logger.Warn("Failed to send job list update to client",
				"client_id", clientID,
				"error", err)
			failedClients = append(failedClients, clientID)
		} else {
			successCount++
		}
	}

	// Remove failed clients after broadcasting
	for _, clientID := range failedClients {
		s.RemoveClient(clientID)
	}

	s.logger.Info("Broadcasted job list update",
		"total_clients", len(clientList),
		"successful", successCount,
		"failed", len(failedClients))
}

// NotifyUpdate implements UpdateNotifier interface
func (s *SSEManager) NotifyUpdate() {
	s.BroadcastJobListUpdate()
}

// NotifyJobUpdate implements UpdateNotifier interface for job-specific updates
func (s *SSEManager) NotifyJobUpdate(jobID string, job *jobs.Job) {
	// For now, just broadcast the general table update since ListAllJobs now includes live progress
	// In the future, this could send job-specific events for more granular updates
	s.BroadcastJobListUpdate()
}

// BroadcastSitesUpdate broadcasts that the sites table has changed
func (s *SSEManager) BroadcastSitesUpdate() {
	// Copy clients list to avoid holding lock during I/O
	s.mu.RLock()
	if len(s.clients) == 0 {
		s.mu.RUnlock()
		s.logger.Debug("No SSE clients connected, skipping sites update broadcast")
		return
	}

	clientList := make(map[string]*SSEClient, len(s.clients))
	for id, client := range s.clients {
		clientList[id] = client
	}
	s.mu.RUnlock()

	successCount := 0
	failedClients := []string{}
	message := `{"action": "refresh", "timestamp": "` + time.Now().Format(time.RFC3339) + `"}`

	for clientID, client := range clientList {
		if err := s.sendToClient(client, "sites-updated", message); err != nil {
			s.logger.Warn("Failed to send sites update to client",
				"client_id", clientID,
				"error", err)
			failedClients = append(failedClients, clientID)
		} else {
			successCount++
		}
	}

	// Remove failed clients after broadcasting
	for _, clientID := range failedClients {
		s.RemoveClient(clientID)
	}

	s.logger.Info("Broadcasted sites update",
		"total_clients", len(clientList),
		"successful", successCount,
		"failed", len(failedClients))
}

// BroadcastToast broadcasts a simple toast notification to all connected clients
func (s *SSEManager) BroadcastToast(message, toastType string) {
	// Copy clients list to avoid holding lock during I/O
	s.mu.RLock()
	if len(s.clients) == 0 {
		s.mu.RUnlock()
		s.logger.Debug("No SSE clients connected, skipping toast broadcast")
		return
	}

	clientList := make(map[string]*SSEClient, len(s.clients))
	for id, client := range s.clients {
		clientList[id] = client
	}
	s.mu.RUnlock()

	successCount := 0
	failedClients := []string{}

	// Use presenter to format toast HTML (proper clean architecture)
	toastHTML, err := s.toastPresenter.FormatToastNotification(message, toastType)
	if err != nil {
		s.logger.Error("Failed to format toast notification", "error", err, "message", message)
		return
	}

	for clientID, client := range clientList {
		if err := s.sendToClient(client, "toast", toastHTML); err != nil {
			s.logger.Warn("Failed to send toast to client",
				"client_id", clientID,
				"message", message,
				"error", err)
			failedClients = append(failedClients, clientID)
		} else {
			successCount++
		}
	}

	// Remove failed clients after broadcasting
	for _, clientID := range failedClients {
		s.RemoveClient(clientID)
	}

	s.logger.Info("Broadcasted toast notification",
		"message", message,
		"type", toastType,
		"total_clients", len(clientList),
		"successful", successCount,
		"failed", len(failedClients))
}

// BroadcastRichJobToast broadcasts a rich toast notification with job details
func (s *SSEManager) BroadcastRichJobToast(job *jobs.Job) {
	// Copy clients list to avoid holding lock during I/O
	s.mu.RLock()
	if len(s.clients) == 0 {
		s.mu.RUnlock()
		s.logger.Debug("No SSE clients connected, skipping rich job toast broadcast")
		return
	}

	clientList := make(map[string]*SSEClient, len(s.clients))
	for id, client := range s.clients {
		clientList[id] = client
	}
	s.mu.RUnlock()

	successCount := 0
	failedClients := []string{}

	// Use presenter to format rich toast HTML (proper clean architecture)
	toastHTML, err := s.toastPresenter.FormatRichJobToastNotification(job)
	if err != nil {
		s.logger.Error("Failed to format rich job toast notification", "error", err, "job_id", job.ID)
		return
	}

	for clientID, client := range clientList {
		if err := s.sendToClient(client, "toast", toastHTML); err != nil {
			s.logger.Warn("Failed to send rich job toast to client",
				"client_id", clientID,
				"job_id", job.ID,
				"error", err)
			failedClients = append(failedClients, clientID)
		} else {
			successCount++
		}
	}

	// Remove failed clients after broadcasting
	for _, clientID := range failedClients {
		s.RemoveClient(clientID)
	}

	s.logger.Info("Broadcasted rich job toast notification",
		"job_id", job.ID,
		"job_type", job.Type,
		"job_status", job.Status,
		"total_clients", len(clientList),
		"successful", successCount,
		"failed", len(failedClients))
}

// sendToClient sends an SSE message to a specific client
func (s *SSEManager) sendToClient(client *SSEClient, event, data string) error {
	select {
	case <-client.done:
		return fmt.Errorf("client connection closed")
	default:
	}

	// Send the SSE formatted message with proper format
	var message string
	if event == "keepalive" || event == "connected" {
		// Special events - send as comments to avoid triggering HTMX
		message = fmt.Sprintf(": %s\n\n", data)
	} else {
		// Regular events - use proper SSE format
		message = fmt.Sprintf("event: %s\ndata: %s\n\n", event, data)
	}

	_, err := client.writer.Write([]byte(message))
	if err != nil {
		return fmt.Errorf("write error: %w", err)
	}

	client.flusher.Flush()
	client.lastSent = time.Now()

	return nil
}

// SendKeepAlive sends keep-alive messages to all clients
func (s *SSEManager) SendKeepAlive() {
	// Copy clients list to avoid holding lock during I/O
	s.mu.RLock()
	clientList := make(map[string]*SSEClient, len(s.clients))
	for id, client := range s.clients {
		clientList[id] = client
	}
	s.mu.RUnlock()

	failedClients := []string{}
	for clientID, client := range clientList {
		if err := s.sendToClient(client, "keepalive", `{"timestamp": "`+time.Now().Format(time.RFC3339)+`"}`); err != nil {
			s.logger.Debug("Keep-alive failed, removing client", "client_id", clientID)
			failedClients = append(failedClients, clientID)
		}
	}

	// Remove failed clients after keep-alive
	for _, clientID := range failedClients {
		s.RemoveClient(clientID)
	}
}

// cleanupRoutine periodically cleans up stale connections
func (s *SSEManager) cleanupRoutine() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		s.SendKeepAlive()

		// Remove clients that haven't received messages in a while
		s.mu.Lock()
		staleThreshold := time.Now().Add(-2 * time.Minute)
		staleClients := []string{}
		for clientID, client := range s.clients {
			if client.lastSent.Before(staleThreshold) {
				s.logger.Info("Removing stale SSE client", "client_id", clientID)
				staleClients = append(staleClients, clientID)
			}
		}
		s.mu.Unlock()

		// Remove stale clients outside of lock
		for _, clientID := range staleClients {
			s.RemoveClient(clientID)
		}
	}
}

// HandleSSEConnection handles the SSE endpoint
func (s *SSEManager) HandleSSEConnection(w http.ResponseWriter, r *http.Request) {
	s.logger.Info("SSE connection attempt",
		"method", r.Method,
		"path", r.URL.Path,
		"query", r.URL.RawQuery)

	clientID := r.URL.Query().Get("client_id")
	if clientID == "" {
		clientID = fmt.Sprintf("client_%d", time.Now().UnixNano())
	}

	client := s.AddClient(clientID, w)
	if client == nil {
		s.logger.Error("Failed to establish SSE connection", "client_id", clientID)
		http.Error(w, "Failed to establish SSE connection", http.StatusInternalServerError)
		return
	}

	// Send initial keep-alive immediately
	if err := s.sendToClient(client, "keepalive", fmt.Sprintf("Connection established at %s", time.Now().Format(time.RFC3339))); err != nil {
		s.logger.Error("Failed to send initial keep-alive", "client_id", clientID, "error", err)
		s.RemoveClient(clientID)
		return
	}

	// Keep connection alive until client disconnects
	ctx := r.Context()

	// Wait for client disconnect - global cleanup routine handles keep-alives
	select {
	case <-ctx.Done():
		s.logger.Info("SSE client context cancelled", "client_id", clientID)
		s.RemoveClient(clientID)
		return
	case <-client.done:
		s.logger.Info("SSE client connection closed", "client_id", clientID)
		s.RemoveClient(clientID)
		return
	}
}
