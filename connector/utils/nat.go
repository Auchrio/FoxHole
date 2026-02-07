package utils

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// NATTraversal handles NAT traversal and hole punching
type NATTraversal struct {
	mu sync.Mutex
}

// NewNATTraversal creates a new NAT traversal handler
func NewNATTraversal() *NATTraversal {
	return &NATTraversal{}
}

// EstablishConnection attempts to establish a connection to a remote peer through NAT
// It uses UDP hole punching to allow two peers behind NAT to communicate directly
func (nt *NATTraversal) EstablishConnection(remoteAddr string, timeoutSec int) (net.Conn, error) {
	timeout := time.Duration(timeoutSec) * time.Second

	// First attempt: try direct connection
	conn, err := nt.AttemptDirectConnection(remoteAddr, timeout)
	if err == nil {
		return conn, nil
	}

	fmt.Printf("Direct connection failed: %v\n", err)
	fmt.Printf("Attempting enhanced hole punching with multiple rounds and port variations...\n")

	// Second attempt: Enhanced UDP hole punching
	conn, err = nt.performHolePunching(remoteAddr, timeout)
	if err == nil {
		return conn, nil
	}

	fmt.Printf("Enhanced hole punching failed: %v\n", err)
	fmt.Printf("Attempting TCP fallback...\n")

	// Fall back to TCP
	return nt.attemptTCPConnection(remoteAddr, timeout)
}

// AttemptDirectConnection tries a direct TCP connection to the peer
func (nt *NATTraversal) AttemptDirectConnection(remoteAddr string, timeout time.Duration) (net.Conn, error) {
	dialer := &net.Dialer{
		Timeout: timeout,
	}

	conn, err := dialer.Dial("tcp", remoteAddr)
	if err != nil {
		return nil, fmt.Errorf("direct TCP connection failed: %w", err)
	}

	// Send handshake to verify connection
	if err := nt.sendHandshake(conn); err != nil {
		conn.Close()
		return nil, err
	}

	fmt.Printf("Direct connection established\n")
	return conn, nil
}

// performHolePunching implements UDP hole punching
// This technique allows two peers behind different NATs to communicate directly
func (nt *NATTraversal) performHolePunching(remoteAddr string, timeout time.Duration) (net.Conn, error) {
	// Parse remote address
	raddr, err := net.ResolveUDPAddr("udp", remoteAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve remote address: %w", err)
	}

	// Try multiple port combinations for better hole punching success
	portOffsets := []int{0, 1, -1, 2, -2, 3, -3}

	for _, offset := range portOffsets {
		// Create local UDP connection with port offset
		targetPort := raddr.Port + offset
		if targetPort < 1024 || targetPort > 65535 {
			continue // Skip invalid ports
		}

		raddrWithOffset := &net.UDPAddr{
			IP:   raddr.IP,
			Port: targetPort,
		}

		conn, err := nt.attemptHolePunchingOnPort(raddrWithOffset, timeout)
		if err == nil {
			return conn, nil
		}

		fmt.Printf("Port offset %d failed, trying next...\n", offset)
	}

	return nil, fmt.Errorf("all port combinations failed")
}

// attemptHolePunchingOnPort attempts hole punching on a specific remote port
func (nt *NATTraversal) attemptHolePunchingOnPort(raddr *net.UDPAddr, timeout time.Duration) (net.Conn, error) {
	// Create local UDP connection
	localAddr := net.UDPAddr{
		Port: 0,
		IP:   net.ParseIP("0.0.0.0"),
	}
	conn, err := net.ListenUDP("udp", &localAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to bind UDP socket: %w", err)
	}

	// Get our local port
	localPort := conn.LocalAddr().(*net.UDPAddr).Port
	fmt.Printf("Listening on UDP port %d, targeting remote port %d\n", localPort, raddr.Port)

	// Multiple rounds of probing with increasing intensity
	rounds := 3
	for round := 1; round <= rounds; round++ {
		fmt.Printf("Hole punching round %d/%d...\n", round, rounds)

		// Send probes with varying intervals
		probeCount := 10 + (round * 5) // 10, 15, 20 probes per round
		baseInterval := 200 * time.Millisecond

		for i := 0; i < probeCount; i++ {
			probe := []byte(fmt.Sprintf("HOLE_PUNCH_PROBE_R%d_P%d", round, i))
			_, err := conn.WriteToUDP(probe, raddr)
			if err != nil {
				conn.Close()
				return nil, fmt.Errorf("failed to send probe: %w", err)
			}

			// Vary interval slightly to avoid patterns
			interval := baseInterval + time.Duration(i*10)*time.Millisecond
			if interval > 500*time.Millisecond {
				interval = 500 * time.Millisecond
			}

			fmt.Printf("Probe %d sent (round %d)\n", i+1, round)
			time.Sleep(interval)
		}

		// After each round, listen for responses with a timeout
		roundTimeout := 3 * time.Second
		conn.SetReadDeadline(time.Now().Add(roundTimeout))

		// Try to receive response
		buffer := make([]byte, 1024)
		n, peerAddr, err := conn.ReadFromUDP(buffer)
		if err == nil {
			fmt.Printf("Received hole punch response from %s (%d bytes): %s\n", peerAddr, n, string(buffer[:n]))

			// Wrap UDP connection to satisfy net.Conn interface
			return NewUDPConn(conn, peerAddr), nil
		}

		if round < rounds {
			fmt.Printf("Round %d timeout, starting round %d...\n", round, round+1)
			time.Sleep(1 * time.Second) // Brief pause between rounds
		}
	}

	conn.Close()
	return nil, fmt.Errorf("no response from peer after %d rounds", rounds)
}

// attemptTCPConnection attempts a TCP connection as fallback
func (nt *NATTraversal) attemptTCPConnection(remoteAddr string, timeout time.Duration) (net.Conn, error) {
	dialer := &net.Dialer{
		Timeout: timeout,
	}

	conn, err := dialer.Dial("tcp", remoteAddr)
	if err != nil {
		return nil, fmt.Errorf("TCP connection failed: %w", err)
	}

	if err := nt.sendHandshake(conn); err != nil {
		conn.Close()
		return nil, err
	}

	fmt.Printf("TCP connection established (fallback)\n")
	return conn, nil
}

// sendHandshake sends a handshake to verify the connection
func (nt *NATTraversal) sendHandshake(conn net.Conn) error {
	handshake := []byte("FOXHOLE_HANDSHAKE_v1")
	_, err := conn.Write(handshake)
	if err != nil {
		return fmt.Errorf("failed to send handshake: %w", err)
	}

	// Read handshake response
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	response := make([]byte, 21)
	n, err := conn.Read(response)
	if err != nil {
		return fmt.Errorf("failed to receive handshake response: %w", err)
	}

	if n != 21 || string(response[:n]) != "FOXHOLE_HANDSHAKE_v1" {
		return fmt.Errorf("invalid handshake response")
	}

	return nil
}

// UDPConn wraps UDP connection to implement net.Conn interface
type UDPConn struct {
	conn       *net.UDPConn
	remoteAddr *net.UDPAddr
	mu         sync.Mutex
}

// NewUDPConn creates a new UDP connection wrapper
func NewUDPConn(conn *net.UDPConn, remoteAddr *net.UDPAddr) *UDPConn {
	return &UDPConn{
		conn:       conn,
		remoteAddr: remoteAddr,
	}
}

// Read implements net.Conn interface
func (uc *UDPConn) Read(b []byte) (int, error) {
	uc.mu.Lock()
	defer uc.mu.Unlock()
	n, _, err := uc.conn.ReadFromUDP(b)
	return n, err
}

// Write implements net.Conn interface
func (uc *UDPConn) Write(b []byte) (n int, err error) {
	uc.mu.Lock()
	defer uc.mu.Unlock()
	return uc.conn.WriteToUDP(b, uc.remoteAddr)
}

// Close implements net.Conn interface
func (uc *UDPConn) Close() error {
	uc.mu.Lock()
	defer uc.mu.Unlock()
	return uc.conn.Close()
}

// LocalAddr implements net.Conn interface
func (uc *UDPConn) LocalAddr() net.Addr {
	return uc.conn.LocalAddr()
}

// RemoteAddr implements net.Conn interface
func (uc *UDPConn) RemoteAddr() net.Addr {
	return uc.remoteAddr
}

// SetDeadline implements net.Conn interface
func (uc *UDPConn) SetDeadline(t time.Time) error {
	return uc.conn.SetDeadline(t)
}

// SetReadDeadline implements net.Conn interface
func (uc *UDPConn) SetReadDeadline(t time.Time) error {
	return uc.conn.SetReadDeadline(t)
}

// SetWriteDeadline implements net.Conn interface
func (uc *UDPConn) SetWriteDeadline(t time.Time) error {
	return uc.conn.SetWriteDeadline(t)
}
