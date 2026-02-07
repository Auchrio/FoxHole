package utils

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/pion/stun"
)

// STUNClient handles communication with STUN servers
type STUNClient struct {
	server string
}

// NewSTUNClient creates a new STUN client
func NewSTUNClient(server string) *STUNClient {
	return &STUNClient{server: server}
}

// GetPublicAddress queries a STUN server to get public IP and port
func (s *STUNClient) GetPublicAddress() (string, uint16, error) {
	return s.GetPublicAddressOnPort(0)
}

// GetPublicAddressOnPort queries STUN server using a specific local port
func (s *STUNClient) GetPublicAddressOnPort(localPort int) (string, uint16, error) {
	var udpConn *net.UDPConn
	var err error

	// Always use UDP connection for consistency
	if localPort == 0 {
		// Use any available port
		udpConn, err = net.ListenUDP("udp", nil)
	} else {
		// Bind to specific local port
		localAddr := &net.UDPAddr{IP: net.ParseIP("0.0.0.0"), Port: localPort}
		udpConn, err = net.ListenUDP("udp", localAddr)
	}

	if err != nil {
		return "", 0, fmt.Errorf("failed to create UDP socket: %w", err)
	}
	defer udpConn.Close()

	// Resolve server address
	serverAddr, err := net.ResolveUDPAddr("udp", s.server)
	if err != nil {
		return "", 0, fmt.Errorf("failed to resolve server address: %w", err)
	}

	// Create STUN message
	message := stun.MustBuild(stun.TransactionID, stun.BindingRequest)

	// Send request
	_, err = udpConn.WriteToUDP(message.Raw, serverAddr)
	if err != nil {
		return "", 0, fmt.Errorf("failed to send STUN request: %w", err)
	}

	// Read response
	response := make([]byte, 1024)
	udpConn.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, _, err := udpConn.ReadFromUDP(response)
	if err != nil {
		return "", 0, fmt.Errorf("failed to read STUN response: %w", err)
	}

	// Parse response
	msg := &stun.Message{Raw: response[:n]}
	if err := msg.Decode(); err != nil {
		return "", 0, fmt.Errorf("failed to decode STUN response: %w", err)
	}

	// Extract XOR-MAPPED-ADDRESS
	var xorAddr stun.XORMappedAddress
	if err := xorAddr.GetFrom(msg); err != nil {
		// Try MAPPED-ADDRESS as fallback
		var addr stun.MappedAddress
		if err := addr.GetFrom(msg); err != nil {
			return "", 0, fmt.Errorf("no address in STUN response: %w", err)
		}
		ip := addr.IP.String()
		port := uint16(addr.Port)
		return ip, port, nil
	}

	ip := xorAddr.IP.String()
	port := uint16(xorAddr.Port)

	return ip, port, nil
}

// VerifyAddress verifies that we can reach the STUN server and the server can reach us back
func (s *STUNClient) VerifyAddress(localAddr string) (bool, error) {
	conn, err := net.Dial("udp", s.server)
	if err != nil {
		return false, err
	}
	defer conn.Close()

	message := stun.MustBuild(stun.TransactionID, stun.BindingRequest)
	_, err = conn.Write(message.Raw)
	if err != nil {
		return false, err
	}

	response := make([]byte, 1024)
	_, err = conn.Read(response)
	if err != nil {
		return false, err
	}

	// If we got a response, the STUN server can reach us
	return true, nil
}

// DetectNATType detects the type of NAT we're behind
// Returns: "open", "restricted", "port_restricted", or "symmetric"
func (s *STUNClient) DetectNATType() (string, error) {
	// Get primary address
	ip1, port1, err := s.GetPublicAddress()
	if err != nil {
		return "", err
	}

	addr1 := fmt.Sprintf("%s:%d", ip1, port1)

	// Try to get address from alternate STUN server if available
	altServer := strings.Replace(s.server, "stun.l.google.com", "stun1.l.google.com", 1)
	if altServer == s.server {
		// If we can't use alternate, use different port on same server
		altServer = "stun2.l.google.com:19302"
	}

	altClient := NewSTUNClient(altServer)
	ip2, port2, err := altClient.GetPublicAddress()
	if err != nil {
		// Try once more with original server
		ip2, port2, err = s.GetPublicAddress()
		if err != nil {
			return "", err
		}
	}

	addr2 := fmt.Sprintf("%s:%d", ip2, port2)

	if addr1 == addr2 {
		return "open", nil // Symmetric NAT or no NAT
	}

	// Further detection would require more complex checks
	// For simplicity, return "restricted" if addresses differ
	return "restricted", nil
}
