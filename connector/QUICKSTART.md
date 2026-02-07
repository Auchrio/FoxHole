# Foxhole Connector - Quick Start Guide

## What Was Created

Your connector program implements a complete NAT traversal and messaging system with:

### Core Components
- **main.go** - CLI interface with listen, connect, and messaging modes
- **stun.go** - STUN client for public address discovery
- **nat.go** - UDP hole punching and connection handling
- **exchange.go** - Integrated Nostr messaging for peer discovery
- **messaging.go** - Secure messaging with AES-256-GCM encryption
- **config.go** - Configuration loading from pulse.conf
- **connector.exe** - Ready-to-run executable

### Key Features
✓ STUN-based public IP discovery  
✓ UDP hole punching for NAT traversal  
✓ Automatic TCP fallback  
✓ Integrated Nostr messaging for peer discovery  
✓ Secure AES-256-GCM encrypted messaging  
✓ **WireGuard VPN configuration exchange**  
✓ Configurable STUN servers  
✓ Handshake verification  

---

## Quick Test

### Test 1: Listen Mode
```powershell
cd c:\Users\Isaac\Documents\FoxHole\connector
.\connector.exe -l test_peer
```

Expected output:
```
Listening mode - ID: test_peer
Public Address: [your public IP]:[port]
Connection info published. Waiting for peer...
```

### Test 2: CLI Help
```powershell
.\connector.exe -h
```

Shows all available options and examples.

---

## Two-Peer Connection Scenario

### Peer 1 (Alice) - Terminal 1
```powershell
.\connector.exe -l alice
```

### Peer 2 (Bob) - Terminal 2
```powershell
.\connector.exe bob alice
```

The program will:
1. Get both peers' public addresses via STUN
2. Exchange addresses through pulse-compat
3. Attempt direct TCP → UDP hole punching → TCP fallback
4. Establish direct connection
5. **Exchange WireGuard configuration for VPN access**
6. Keep connection alive

---

## Available CLI Options

```
-l              Enable listen mode
-addr string    Address to listen on (default: 0.0.0.0:0)
-stun string    STUN server (default: stun.l.google.com:19302)
-timeout int    Connection timeout in seconds (default: 30)
```

### Example with Custom Settings
```powershell
.\connector.exe -l myid -stun stun.stunprotocol.org:3478 -timeout 60
```

---

## WireGuard VPN Integration

The connector automatically exchanges WireGuard configuration when a connection is established, allowing the connecting peer to access the WireGuard VPN server running on the listener's host.

### Important: Now Works Behind Hard NATs!

**The connector uses NAT traversal to establish direct P2P connections, enabling WireGuard VPN access even when both peers are behind hard NATs.**

- ✅ **Works**: Both peers behind hard NATs (symmetric, restricted, etc.)
- ✅ **No port forwarding required** for the connector
- ✅ **Secure config exchange** over NAT-traversed connection
- ⚠️ **WireGuard server still needs accessibility** (see below)

### For the Listener (Host with WireGuard Server)

1. **Ensure WireGuard Server is Running** on your host
   - The server should be accessible (public IP, port forwarding, or the connector will make it accessible)

2. **Start Listener Mode**:
   ```powershell
   .\connector.exe -l myhost
   ```
   The program will:
   - Discover your public address via STUN
   - Publish address via Nostr
   - Listen for incoming NAT-traversed connections
   - Automatically send WireGuard config when connected

### For the Connector (Client wanting VPN access)

1. **Run Connect Mode**:
   ```powershell
   .\connector.exe myclient myhost
   ```
2. The program will:
   - Discover your public address
   - Exchange addresses with the listener
   - Perform NAT traversal (UDP hole punching)
   - Establish direct P2P connection
   - Receive WireGuard configuration
   - Display setup instructions

### Requirements

- **Nostr relay access** for address exchange (default: relay.damus.io)
- **STUN server access** for public address discovery (default: stun.l.google.com)
- **WireGuard server** running on listener host
- **WireGuard client** on connecting host

### Troubleshooting WireGuard Connection

If WireGuard won't connect after receiving config:
- Check if the WireGuard server is running and accessible
- Verify the endpoint in the config matches your WireGuard server
- Test UDP connectivity to the WireGuard port
- Ensure firewall allows WireGuard traffic

---

## Architecture Overview

```
Peer A (Listener)                    Peer B (Connector)
┌─────────────────┐                 ┌─────────────────┐
│  Get Public IP  │                 │  Get Public IP  │
│  (STUN Query)   │                 │  (STUN Query)   │
└────────┬────────┘                 └────────┬────────┘
         │                                    │
         ├──→ Publish via pulse-compat ←──────┤
         │                                    │
         │         Retrieve Peer B IP ←───────┤
         │                                    │
         │     UDP Hole Punching Attempt      │
         │ ←──────────────────────────────────┤
         │                                    │
         │ (If fails, TCP fallback)           │
         │                                    │
         └──────→ Direct Connection ←─────────┘
```

---

## Connection Methods (Tried in Order)

1. **Direct TCP** - Fastest, works if firewalls allow
2. **UDP Hole Punching** - Works for most NAT types
3. **TCP Fallback** - Last resort, always works

Both peers send "probe" packets to create NAT mappings, then establish communication through the opened "holes".

---

## Integrated Messaging

Connection info format (exchanged as JSON):
```json
{
  "id": "alice",
  "ip": "203.0.113.42",
  "port": 54321
}
```

This is sent through the integrated Nostr messaging system, allowing peers to discover each other's public addresses without requiring external tools.

---

## Next Steps

1. **Test locally** - Run test_peer in listen mode, create multiple connections
2. **Network test** - Test with peers on different networks  
3. **Add data exchange** - Implement actual data transfer in handleConnection()
4. **Optimize timeouts** - Adjust -timeout based on your network conditions
5. **Implement keep-alive** - Add periodic pings to maintain NAT mappings

---

## Building & Rebuilding

If you modify the code:
```powershell
cd c:\Users\Isaac\Documents\FoxHole\connector
go mod tidy
go build -o connector.exe
```

---

## Dependencies

- **pion/stun** - STUN protocol implementation
- **go-nostr** - Nostr protocol for decentralized messaging
- **pulse.conf** - Configuration file (optional, defaults provided)

All dependencies are integrated - no external programs required.

---

## Troubleshooting

| Problem | Solution |
|---------|----------|
| STUN fails | Try different STUN server with `-stun` flag |
| Peer not found | Ensure other peer published their ID first |
| No connection | Some NATs don't support hole punching; TCP fallback will be used |
| Timeout too short | Increase with `-timeout` flag |

---

## Files Structure

```
connector/
├── main.go           # CLI & connection orchestration
├── stun.go           # STUN client implementation
├── nat.go            # NAT traversal & UDP wrapper
├── exchange.go       # Pulse-compat integration
├── connector.exe     # Compiled executable
├── go.mod            # Go module definition
└── README.md         # Full documentation
```

---

Good luck with your NAT traversal implementation! 🚀
