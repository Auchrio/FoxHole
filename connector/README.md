# Foxhole Connector

A NAT traversal and messaging tool that enables direct peer-to-peer connections between devices behind NAT (Network Address Translation) firewalls using STUN (Session Traversal Utilities for NAT) and UDP hole punching, with integrated Nostr-based messaging.

## Overview

The Connector program has multiple modes:

1. **Listening Mode** (`-l`): Waits for incoming connections and publishes its public address
2. **Connect Mode** (default): Connects to a peer that's listening
3. **Send Message** (`<id> <message>`): Sends an encrypted message via Nostr
4. **Read Message** (`-read <id>`): Reads the latest message for an ID
5. **Listen for Message** (`-listen-msg <id>`): Waits for new messages

The program:
- Uses STUN servers to discover your public IP and port
- Publishes connection info via integrated Nostr messaging
- Establishes direct connections using UDP hole punching for NAT traversal
- Falls back to TCP if UDP hole punching fails
- Provides secure messaging with AES-256-GCM encryption
- **WireGuard Integration**: Automatically exchanges WireGuard configuration for VPN access

## Architecture

### Components

- **main.go**: CLI interface and mode orchestration
- **stun.go**: STUN client for public address discovery and NAT type detection
- **nat.go**: NAT traversal using UDP hole punching with fallback to TCP
- **exchange.go**: Connection info exchange via integrated Nostr messaging
- **messaging.go**: Nostr-based secure messaging with encryption
- **config.go**: Configuration loading from pulse.conf

### Connection Flow

### Connection Flow

#### Listening (Peer A)

```
1. Get public IP:port from STUN server (using consistent local port)
2. Publish address via Nostr with ID "A"
3. Listen on local port for incoming connections
4. Wait for connecting peer to send their address via Nostr
5. Accept NAT-traversed connection
6. Exchange WireGuard configuration
```

#### Connecting (Peer B)

```
1. Get public IP:port from STUN server (using consistent local port)
2. Publish own address via Nostr with ID "B"
3. Send address to listener "A" via Nostr message
4. Retrieve listener "A"'s address via Nostr
5. Perform UDP hole punching + TCP connection to "A"'s address
6. Establish direct P2P connection
7. Receive WireGuard configuration
```

## Usage

### Listen Mode

```bash
connector -l <id>
```

Example:
```bash
connector -l alice
```

This will:
- Start listening on a UDP port
- Query a STUN server to get your public address
- Publish your connection info under ID "alice"
- Wait for an incoming connection

### Connect Mode

```bash
connector <local-id> <remote-id>
```

Example:
```bash
connector bob alice
```

This will:
- Get your public address from STUN
- Retrieve "alice"'s connection info from pulse-compat
- Attempt to connect to alice's peer
- **Receive WireGuard configuration for VPN access**
- Establish direct communication

### Options

- `-l`: Listen mode (wait for incoming connections)
- `-addr string`: Address to listen on (default: "0.0.0.0:0" for auto port)
- `-stun string`: STUN server to use (default: "stun.l.google.com:19302")
- `-timeout int`: Connection timeout in seconds (default: 30)

## How NAT Traversal Works

### UDP Hole Punching

1. Both peers query STUN to get public addresses
2. Connecting peer sends "probe" packets to listening peer's public address
3. These probes create a mapping in connecting peer's NAT device
4. When listening peer responds, packets can traverse back through the NAT
5. Direct connection is established through the "holes" opened in both NATs

### Connection Attempts

The program tries connections in this order:

1. **Direct TCP**: Fastest if both peers have compatible NAT
2. **UDP Hole Punching**: Works for most NAT types
3. **TCP Fallback**: Works as last resort for restrictive firewalls

### Handshake

All successful connections include a handshake to verify connectivity:
- Sender: `FOXHOLE_HANDSHAKE_v1` (21 bytes)
- Receiver echoes back the same handshake
- If received, connection is confirmed

## WireGuard Integration

When a connection is established through NAT traversal, the listener automatically sends WireGuard configuration to the connecting peer. This enables secure VPN connectivity even when both peers are behind hard NATs.

### How NAT Traversal Enables WireGuard

The connector establishes a **direct peer-to-peer connection** between devices behind NATs, then uses this connection to securely exchange WireGuard configuration. The actual WireGuard VPN traffic can then flow directly between the peers.

### Setup Requirements

1. **NAT Traversal**: Both peers must be able to reach Nostr relays for address exchange
2. **STUN Server**: Accessible STUN server for public address discovery
3. **WireGuard Server**: Running on the listener host
4. **Consistent Ports**: The connector uses consistent local ports for NAT mapping

### Scenarios That Now Work

✅ **Both behind hard NATs** - Connector establishes P2P link, then WireGuard works
✅ **Mixed NAT types** - Works with symmetric, restricted, and open NATs
✅ **No port forwarding needed** - NAT traversal eliminates this requirement
✅ **Dynamic IPs** - STUN handles changing public addresses

### Connection Flow

1. Both peers discover their public addresses via STUN
2. Addresses are exchanged securely via Nostr
3. UDP hole punching establishes direct P2P connection
4. WireGuard configuration is sent over the secure P2P link
5. WireGuard client connects directly to WireGuard server (using the P2P connection for initial setup only)

### Example WireGuard Config Exchange

The listener sends configuration like:
```
To connect to WireGuard:
1. Install WireGuard on your system
2. Use this configuration:

[Interface]
PrivateKey = <your-private-key>
Address = 10.0.0.2/24

[Peer]
PublicKey = <server-public-key>
Endpoint = <server-endpoint>
AllowedIPs = 0.0.0.0/0

Replace the placeholders with actual values from your WireGuard server.
```

## Integrated Messaging

The connector includes integrated Nostr-based messaging for peer discovery and communication. Connection details (ID, IP, port) are sent as JSON:

```json
{
  "id": "alice",
  "ip": "203.0.113.42",
  "port": 54321
}
```

Messages are encrypted using AES-256-GCM with keys derived from the peer ID and a shared secret. The system uses decentralized Nostr relays for message delivery.

## Requirements

- Go 1.20+
- pulse-compat binary in PATH or current directory
- Network access to at least one STUN server
- UDP and TCP network access (for connections)

## Dependencies

- `github.com/pion/stun`: STUN protocol implementation

## Examples

### Basic Two-Way Connection

**Terminal 1 (Alice - Listening):**
```bash
connector -l alice
# Output: Listening mode - ID: alice
#         Public Address: 203.0.113.42:54321
#         Connection info published. Waiting for peer...
```

**Terminal 2 (Bob - Connecting):**
```bash
connector bob alice
# Output: Connect mode - Local ID: bob, Remote ID: alice
#         Our Public Address: 198.51.100.77:12345
#         Peer Address: 203.0.113.42:54321
#         ✓ Connection established!
#         Connection active. Press Ctrl+C to close.
```

### Custom STUN Server

```bash
connector -l -stun stun.stunprotocol.org:3478 myid
```

### Extended Timeout

```bash
connector bob alice -timeout 60
```

## NAT Type Detection

The program can detect NAT types to optimize connection strategies:

- **Open**: No NAT or symmetric NAT without port remapping
- **Restricted**: NAT that restricts based on IP
- **Port Restricted**: NAT that also restricts based on port
- **Symmetric**: NAT that creates unique mappings per destination

## Troubleshooting

### "Failed to get public address"
- Check STUN server is accessible
- Try alternate STUN server with `-stun` flag
- Verify network firewall isn't blocking STUN (UDP port 3478/19302)

### "No connection info published by peer"
- Ensure remote peer is running in listen mode
- Check that both peers can reach Nostr relay servers
- Verify network allows outbound connections to relays (wss://relay.damus.io, wss://nos.lol)

### "No response from peer"
- Both peers should appear connected now
- Check firewall rules allow UDP between peers
- Try directly if you have IPs (some firewalls block UDP)
- Fallback to TCP will be attempted

### Connection still fails
- Check both peers have internet connectivity
- Verify DNS resolution works
- Try with explicit STUN server
- Some carrier-grade NAT implementations may not support hole punching

## Building from Source

```bash
cd connector
go mod download
go build -o connector
```

## License

Part of Foxhole Project
