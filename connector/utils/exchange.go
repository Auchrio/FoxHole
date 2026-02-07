package utils

import (
	"encoding/json"
	"fmt"
	"time"
)

// ConnectionInfo holds the connection details of a peer
type ConnectionInfo struct {
	ID        string `json:"id"`
	IP        string `json:"ip"`         // Public IP
	Port      uint16 `json:"port"`       // Public port
	LocalIP   string `json:"local_ip"`   // Local IP
	LocalPort uint16 `json:"local_port"` // Local port
}

// Exchanger handles connection info exchange between peers via Nostr
type Exchanger struct{}

// NewExchanger creates a new connection exchanger
func NewExchanger() *Exchanger {
	return &Exchanger{}
}

// PublishConnectionInfo publishes this peer's connection info via Nostr
// The info is sent to a special channel that the remote peer can listen to
func (e *Exchanger) PublishConnectionInfo(info *ConnectionInfo) error {
	// Encode connection info as JSON
	data, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("failed to marshal connection info: %w", err)
	}

	// Send via Nostr messaging
	return SendMessage(info.ID, string(data))
}

// RetrieveConnectionInfo retrieves connection info from a peer via Nostr
// It polls the Nostr relays for messages from the peer
func (e *Exchanger) RetrieveConnectionInfo(peerID string) (*ConnectionInfo, error) {
	// Try to retrieve message from peer using Nostr messaging
	message, err := ReadMessages(peerID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve from Nostr: %w", err)
	}

	// Parse JSON response
	var info ConnectionInfo
	if err := json.Unmarshal([]byte(message), &info); err != nil {
		return nil, fmt.Errorf("failed to parse connection info: %w", err)
	}

	return &info, nil
}

// RetrieveConnectionInfoWithRetry retrieves connection info with retry logic
// It waits for the peer to publish their connection info
func (e *Exchanger) RetrieveConnectionInfoWithRetry(peerID string, timeout time.Duration, maxRetries int) (*ConnectionInfo, error) {
	deadline := time.Now().Add(timeout)

	for attempt := 0; attempt < maxRetries; attempt++ {
		info, err := e.RetrieveConnectionInfo(peerID)
		if err == nil {
			return info, nil
		}

		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timeout waiting for connection info from %s", peerID)
		}

		waitTime := time.Duration((attempt + 1)) * time.Second
		if waitTime > 5*time.Second {
			waitTime = 5 * time.Second
		}

		fmt.Printf("Waiting for peer to publish connection info (attempt %d/%d)...\n", attempt+1, maxRetries)
		time.Sleep(waitTime)
	}

	return nil, fmt.Errorf("failed to retrieve connection info from %s after %d attempts", peerID, maxRetries)
}

// ListenForConnectionInfo listens for connection info from peers via Nostr
// This is used in listener mode to wait for the connecting peer's info
func (e *Exchanger) ListenForConnectionInfo(listenerID string, timeout time.Duration) (*ConnectionInfo, error) {
	// Use Nostr messaging in listen mode
	timeoutSec := int(timeout.Seconds())

	message, err := ListenMessages(listenerID, timeoutSec)
	if err != nil {
		return nil, fmt.Errorf("failed to listen via Nostr: %w", err)
	}

	// Parse JSON response
	var info ConnectionInfo
	if err := json.Unmarshal([]byte(message), &info); err != nil {
		return nil, fmt.Errorf("failed to parse incoming connection info: %w", err)
	}

	return &info, nil
}
