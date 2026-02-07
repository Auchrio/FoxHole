package main

import (
	"connector/utils"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Mode       string // "listen", "connect", "send", "read", "listen-msg"
	ID         string
	Message    string // Message to send
	ListenAddr string // Address to listen on (e.g., "0.0.0.0:0" for auto port)
	RemoteID   string // Remote peer ID to connect to
	STUNServer string
	Timeout    int
}

func main() {
	config := parseArgs()

	switch config.Mode {
	case "listen":
		if err := runListener(config); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "connect":
		if err := runConnector(config); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "send":
		if err := runSendMessage(config); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "read":
		if err := runReadMessage(config); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "listen-msg":
		if err := runListenMessage(config); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}
}

func parseArgs() *Config {
	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	listenMode := fs.Bool("l", false, "Listen mode - wait for incoming connection")
	listenAddr := fs.String("addr", "0.0.0.0:8080", "Address to listen on")
	stunServer := fs.String("stun", "stun.l.google.com:19302", "STUN server")
	timeout := fs.Int("timeout", 30, "Timeout in seconds")

	// Messaging flags
	sendMode := fs.Bool("send", false, "Send message mode")
	readMode := fs.Bool("read", false, "Read latest message mode")
	listenMsgMode := fs.Bool("listen-msg", false, "Listen for new message mode")
	listenMsgTimeout := fs.Int("listen-timeout", -1, "Listen timeout in seconds (0 = indefinite, -1 = config default)")

	// Handle short flags
	args := os.Args[1:]

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Foxhole Connector - NAT Traversal and Messaging Tool\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  connector -l <id>                    # Listen for incoming connection (returns connection info)\n")
		fmt.Fprintf(os.Stderr, "  connector <local-id> <remote-id>      # Connect to a listening peer\n")
		fmt.Fprintf(os.Stderr, "  connector <id> <message>              # Send message\n")
		fmt.Fprintf(os.Stderr, "  connector -read <id>                  # Read latest message\n")
		fmt.Fprintf(os.Stderr, "  connector -listen-msg <id>            # Listen for new message\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  connector -l myid                     # Listen with ID 'myid'\n")
		fmt.Fprintf(os.Stderr, "  connector myid peerId                 # Connect from 'myid' to peer 'peerId'\n")
		fmt.Fprintf(os.Stderr, "  connector myid \"hello world\"          # Send message\n")
		fmt.Fprintf(os.Stderr, "  connector -read myid                  # Read latest message\n")
		fmt.Fprintf(os.Stderr, "  connector -listen-msg myid            # Listen for new message (30s default)\n")
		fmt.Fprintf(os.Stderr, "  connector -listen-msg -listen-timeout 0 myid  # Listen indefinitely\n")
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	parsedArgs := fs.Args()

	if *listenMode {
		if len(parsedArgs) == 0 {
			fmt.Fprintf(os.Stderr, "Error: ID required in listen mode\n")
			fs.Usage()
			os.Exit(1)
		}
		return &Config{
			Mode:       "listen",
			ID:         parsedArgs[0],
			ListenAddr: *listenAddr,
			STUNServer: *stunServer,
			Timeout:    *timeout,
		}
	}

	// Messaging modes
	if *sendMode || *readMode || *listenMsgMode {
		if len(parsedArgs) == 0 {
			fmt.Fprintf(os.Stderr, "Error: ID required for messaging modes\n")
			fs.Usage()
			os.Exit(1)
		}
		id := parsedArgs[0]

		if *sendMode {
			if len(parsedArgs) < 2 {
				fmt.Fprintf(os.Stderr, "Error: message required in send mode\n")
				fs.Usage()
				os.Exit(1)
			}
			return &Config{
				Mode:    "send",
				ID:      id,
				Message: parsedArgs[1],
			}
		}

		if *readMode {
			return &Config{
				Mode: "read",
				ID:   id,
			}
		}

		if *listenMsgMode {
			timeoutSec := *timeout // use config default
			if *listenMsgTimeout >= 0 {
				timeoutSec = *listenMsgTimeout
			}
			return &Config{
				Mode:    "listen-msg",
				ID:      id,
				Timeout: timeoutSec,
			}
		}
	}

	// Send mode (message provided without flag) or Connect mode
	if len(parsedArgs) == 2 {
		secondArg := parsedArgs[1]
		// If second arg contains spaces, quotes, or looks like a message, treat as send mode
		if strings.Contains(secondArg, " ") || strings.Contains(secondArg, "\"") || strings.Contains(secondArg, "'") || len(secondArg) > 50 {
			return &Config{
				Mode:    "send",
				ID:      parsedArgs[0],
				Message: secondArg,
			}
		} else {
			// Otherwise, treat as connect mode (peer ID)
			return &Config{
				Mode:       "connect",
				ID:         parsedArgs[0],
				RemoteID:   parsedArgs[1],
				STUNServer: *stunServer,
				Timeout:    *timeout,
			}
		}
	}

	// Connect mode with more args (fallback)
	if len(parsedArgs) >= 2 {
		return &Config{
			Mode:       "connect",
			ID:         parsedArgs[0],
			RemoteID:   parsedArgs[1],
			STUNServer: *stunServer,
			Timeout:    *timeout,
		}
	}

	// Invalid arguments
	fmt.Fprintf(os.Stderr, "Error: Invalid arguments\n")
	fs.Usage()
	os.Exit(1)
	return nil // This won't be reached, but needed for compilation
}

func runListener(config *Config) error {
	fmt.Printf("Listening mode - ID: %s\n", config.ID)

	// Parse the listen address to get the port
	_, portStr, err := net.SplitHostPort(config.ListenAddr)
	if err != nil {
		return fmt.Errorf("invalid listen address: %w", err)
	}

	localPort, err := net.LookupPort("tcp", portStr)
	if err != nil {
		// If it's not a service name, try parsing as number
		if p, err2 := strconv.Atoi(portStr); err2 == nil {
			localPort = p
		} else {
			return fmt.Errorf("invalid port: %w", err)
		}
	}

	// Get our public IP and port via STUN using the same local port
	stunClient := utils.NewSTUNClient(config.STUNServer)
	ip, publicPort, err := stunClient.GetPublicAddressOnPort(localPort)
	if err != nil {
		return fmt.Errorf("failed to get public address: %w", err)
	}

	fmt.Printf("Public Address: %s:%d (local port: %d)\n", ip, publicPort, localPort)

	// Get local IP address
	localIP, err := getLocalIP()
	if err != nil {
		fmt.Printf("Warning: failed to get local IP: %v, using 127.0.0.1\n", err)
		localIP = "127.0.0.1"
	}

	// Create connection info
	connInfo := &utils.ConnectionInfo{
		ID:        config.ID,
		IP:        ip,
		Port:      publicPort,
		LocalIP:   localIP,
		LocalPort: uint16(localPort),
	}

	// Exchange connection info via pulse-compat
	exchanger := utils.NewExchanger()
	if err := exchanger.PublishConnectionInfo(connInfo); err != nil {
		return fmt.Errorf("failed to publish connection info: %w", err)
	}

	fmt.Printf("Connection info published. Waiting for peer...\n")

	// Wait for incoming connections using NAT traversal
	if err := handleIncomingConnection(config, connInfo, localPort); err != nil {
		return err
	}

	return nil
}

func runConnector(config *Config) error {
	fmt.Printf("Connect mode - Local ID: %s, Remote ID: %s\n", config.ID, config.RemoteID)

	// Use a consistent local port for STUN and application traffic
	localPort := 8080 // Use same port as listener for consistency

	// Get our public IP and port via STUN
	stunClient := utils.NewSTUNClient(config.STUNServer)
	localIP, localPublicPort, err := stunClient.GetPublicAddressOnPort(localPort)
	if err != nil {
		return fmt.Errorf("failed to get local public address: %w", err)
	}

	fmt.Printf("Local Public Address: %s:%d\n", localIP, localPublicPort)

	// Get local IP address
	localIPAddress, err := getLocalIP()
	if err != nil {
		fmt.Printf("Warning: failed to get local IP: %v, using 127.0.0.1\n", err)
		localIPAddress = "127.0.0.1"
	}

	// First, retrieve the listener's published connection info
	fmt.Printf("Retrieving peer connection info...\n")
	exchanger := utils.NewExchanger()
	remoteInfo, err := exchanger.RetrieveConnectionInfo(config.RemoteID)
	if err != nil {
		return fmt.Errorf("failed to retrieve peer connection info: %w", err)
	}

	fmt.Printf("Remote Address: %s:%d\n", remoteInfo.IP, remoteInfo.Port)

	// Publish our connection info so the listener can retrieve it
	localConnInfo := &utils.ConnectionInfo{
		ID:        config.ID,
		IP:        localIP,
		Port:      localPublicPort,
		LocalIP:   localIPAddress,
		LocalPort: uint16(localPort),
	}
	if err := exchanger.PublishConnectionInfo(localConnInfo); err != nil {
		return fmt.Errorf("failed to publish local connection info: %w", err)
	}

	// Send our connection info to the remote peer so they can do hole punching
	localInfoJSON, _ := json.Marshal(localConnInfo)
	if err := utils.SendMessage(config.RemoteID, string(localInfoJSON)); err != nil {
		fmt.Printf("Warning: failed to send local info to peer: %v\n", err)
		// Continue anyway, as the peer might still connect
	}

	// Agree on a synchronized start time (current time + 5 seconds)
	startTime := time.Now().Add(5 * time.Second).Unix()
	startTimeMsg := fmt.Sprintf("HOLE_PUNCH_START_TIME_%d", startTime)

	fmt.Printf("Proposing synchronized start time: %d (%s)\n", startTime, time.Unix(startTime, 0).Format("15:04:05"))

	if err := utils.SendMessage(config.RemoteID, startTimeMsg); err != nil {
		fmt.Printf("Warning: failed to send start time: %v\n", err)
	}

	// Wait for peer's proposed time or use our own
	timeTimeout := 8 * time.Second
	timeReceived := make(chan int64, 1)

	go func() {
		// Listen for start time proposal
		for {
			msg, err := utils.ListenMessages(config.ID, 1) // Short timeout
			if err != nil {
				continue
			}
			if strings.Contains(msg, "HOLE_PUNCH_START_TIME_") {
				parts := strings.Split(msg, "_")
				if len(parts) >= 4 {
					if t, err := strconv.ParseInt(parts[3], 10, 64); err == nil {
						timeReceived <- t
						return
					}
				}
			}
		}
	}()

	// Wait for peer's time or timeout
	select {
	case peerTime := <-timeReceived:
		// Use the later of the two times to ensure both are ready
		if peerTime > startTime {
			startTime = peerTime
		}
		fmt.Printf("Synchronized on start time: %d (%s)\n", startTime, time.Unix(startTime, 0).Format("15:04:05"))
	case <-time.After(timeTimeout):
		fmt.Printf("Using proposed start time: %d (%s)\n", startTime, time.Unix(startTime, 0).Format("15:04:05"))
	}

	// Wait until the exact start time
	now := time.Now().Unix()
	if startTime > now {
		waitDuration := time.Duration(startTime-now) * time.Second
		fmt.Printf("Waiting %v until synchronized start...\n", waitDuration)
		time.Sleep(waitDuration)
	}

	fmt.Printf("🎯 Starting synchronized hole punching at %s!\n", time.Now().Format("15:04:05.000"))

	// Perform NAT traversal and establish connection
	tracer := utils.NewNATTraversal()
	conn, err := tracer.EstablishConnection(
		fmt.Sprintf("%s:%d", remoteInfo.IP, remoteInfo.Port),
		config.Timeout,
	)
	if err != nil {
		return fmt.Errorf("failed to establish connection: %w", err)
	}

	fmt.Printf("✓ Connection established!\n")
	defer conn.Close()

	// Keep connection alive and handle data
	handleConnection(conn)

	return nil
}

func handleIncomingConnection(config *Config, connInfo *utils.ConnectionInfo, localPort int) error {
	// First, try to retrieve the connector's connection info
	fmt.Printf("Waiting for peer connection info...\n")

	exchanger := utils.NewExchanger()
	peerInfo, err := exchanger.ListenForConnectionInfo(config.ID, 30) // Wait up to 30 seconds
	if err != nil {
		fmt.Printf("Warning: Could not retrieve peer info via Nostr: %v\n", err)
		fmt.Printf("Proceeding with direct listening...\n")
	}

	if peerInfo != nil {
		fmt.Printf("Received peer info: %s:%d\n", peerInfo.IP, peerInfo.Port)

		// Agree on a synchronized start time (current time + 5 seconds)
		startTime := time.Now().Add(5 * time.Second).Unix()
		startTimeMsg := fmt.Sprintf("HOLE_PUNCH_START_TIME_%d", startTime)

		fmt.Printf("Proposing synchronized start time: %d (%s)\n", startTime, time.Unix(startTime, 0).Format("15:04:05"))

		if err := utils.SendMessage(peerInfo.ID, startTimeMsg); err != nil {
			fmt.Printf("Warning: failed to send start time: %v\n", err)
		}

		// Wait for peer's proposed time or use our own
		timeTimeout := 8 * time.Second
		timeReceived := make(chan int64, 1)

		go func() {
			// Listen for start time proposal
			for {
				msg, err := utils.ListenMessages(config.ID, 1) // Short timeout
				if err != nil {
					continue
				}
				if strings.Contains(msg, "HOLE_PUNCH_START_TIME_") {
					parts := strings.Split(msg, "_")
					if len(parts) >= 4 {
						if t, err := strconv.ParseInt(parts[3], 10, 64); err == nil {
							timeReceived <- t
							return
						}
					}
				}
			}
		}()

		// Wait for peer's time or timeout
		select {
		case peerTime := <-timeReceived:
			// Use the later of the two times to ensure both are ready
			if peerTime > startTime {
				startTime = peerTime
			}
			fmt.Printf("Synchronized on start time: %d (%s)\n", startTime, time.Unix(startTime, 0).Format("15:04:05"))
		case <-time.After(timeTimeout):
			fmt.Printf("Using proposed start time: %d (%s)\n", startTime, time.Unix(startTime, 0).Format("15:04:05"))
		}

		// Wait until the exact start time
		now := time.Now().Unix()
		if startTime > now {
			waitDuration := time.Duration(startTime-now) * time.Second
			fmt.Printf("Waiting %v until synchronized start...\n", waitDuration)
			time.Sleep(waitDuration)
		}

		fmt.Printf("🎯 Starting synchronized bidirectional hole punching at %s!\n", time.Now().Format("15:04:05.000"))

		// Perform bidirectional hole punching
		go func() {
			tracer := utils.NewNATTraversal()
			peerConn, err := tracer.EstablishConnection(
				fmt.Sprintf("%s:%d", peerInfo.IP, peerInfo.Port),
				30, // 30 second timeout
			)
			if err != nil {
				fmt.Printf("Listener hole punching failed: %v\n", err)
				return
			}

			fmt.Printf("Listener established connection via hole punching!\n")
			// Handle the WireGuard connection
			go handleWireGuardConnection(peerConn, connInfo)
		}()
	}

	// Listen on the local port for incoming connections
	// The NAT should forward connections from our public port to this local port
	listenAddr := fmt.Sprintf("0.0.0.0:%d", localPort)

	fmt.Printf("Listening for incoming connections on %s\n", listenAddr)

	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return fmt.Errorf("failed to start listener: %w", err)
	}
	defer listener.Close()

	fmt.Printf("Waiting for peer to establish connection...\n")

	// Accept incoming connection
	conn, err := listener.Accept()
	if err != nil {
		return fmt.Errorf("failed to accept connection: %w", err)
	}

	fmt.Printf("Connection established from %s\n", conn.RemoteAddr())

	// Handle the WireGuard connection
	go handleWireGuardConnection(conn, connInfo)

	// Wait for the connection to close
	select {}
}

func handleWireGuardConnection(conn net.Conn, connInfo *utils.ConnectionInfo) {
	defer conn.Close()

	fmt.Printf("WireGuard connection established with %s\n", conn.RemoteAddr())

	// Send WireGuard configuration with actual server endpoint
	wgConfig := fmt.Sprintf(`To connect to WireGuard:
1. Install WireGuard on your system
2. Use this configuration:

[Interface]
PrivateKey = <your-private-key>
Address = 10.0.0.2/24

[Peer]
PublicKey = <server-public-key>
Endpoint = %s:51820  # Standard WireGuard port - adjust if different
AllowedIPs = 0.0.0.0/0

Replace the placeholders with actual values from your WireGuard server.
Note: The WireGuard server must be accessible at the endpoint above.
If both you and the server are behind NATs, port forwarding is required.`, connInfo.IP)

	_, err := conn.Write([]byte(wgConfig))
	if err != nil {
		fmt.Printf("Error sending WireGuard config: %v\n", err)
		return
	}

	fmt.Printf("WireGuard configuration sent to %s\n", conn.RemoteAddr())

	// Keep connection alive for any further communication
	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			if err != io.EOF {
				fmt.Printf("Connection closed: %v\n", err)
			}
			return
		}
		fmt.Printf("Received %d bytes from %s: %s\n", n, conn.RemoteAddr(), string(buf[:n]))
	}
}

func handleConnection(conn net.Conn) {
	fmt.Printf("Connection active. Receiving WireGuard configuration...\n")

	// Read WireGuard configuration from the server
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		fmt.Printf("Error reading WireGuard config: %v\n", err)
		return
	}

	wgConfig := string(buf[:n])
	fmt.Printf("Received WireGuard configuration:\n%s\n", wgConfig)

	// Keep connection alive for any further communication
	fmt.Printf("Connection established. You can now configure WireGuard with the above settings.\n")

	// Keep the connection alive
	for {
		n, err := conn.Read(buf)
		if err != nil {
			if err != io.EOF {
				fmt.Printf("Connection closed: %v\n", err)
			}
			return
		}
		fmt.Printf("Received %d bytes: %s\n", n, string(buf[:n]))
	}
}

func runSendMessage(config *Config) error {
	if err := utils.SendMessage(config.ID, config.Message); err != nil {
		return err
	}
	fmt.Print("OK")
	return nil
}

func runReadMessage(config *Config) error {
	msg, err := utils.ReadMessages(config.ID)
	if err != nil {
		return err
	}
	fmt.Print(msg)
	return nil
}

func runListenMessage(config *Config) error {
	msg, err := utils.ListenMessages(config.ID, config.Timeout)
	if err != nil {
		return err
	}
	fmt.Print(msg)
	return nil
}

// getLocalIP returns the local IP address of this machine
func getLocalIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String(), nil
}
