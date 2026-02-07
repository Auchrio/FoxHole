# Pulse Compat - Microcontroller Edition

```sh
go build -ldflags="-s -w" -trimpath -o pulse-compat.exe main.go
```

```sh
set GOOS=linux&& set GOARCH=arm&& go build -ldflags="-s -w" -trimpath -o pulse-compat main.go && set GOOS=&& set GOARCH=
```

A lightweight, memory-efficient implementation of Pulse designed for microcontrollers and embedded systems with minimal storage and RAM.

## Features

- **Read Messages**: Retrieve the latest encrypted message
- **Listen**: Wait for new incoming messages with timeout
- **Send**: Transmit encrypted messages

## Optimizations

- **Removed**: Cobra CLI framework, verbose logging, chat mode, config files, status tracking
- **Compact**: Single core file with all essential logic (~180 lines)
- **Memory-safe**: Uses pre-allocated buffers, context timeouts, and controlled goroutines
- **Minimal dependencies**: Only `go-nostr` required
- **Short variable names**: Reduces binary size

## Build

```bash
cd compat
go build -ldflags="-s -w" -o pulse-compat main.go core.go
```

The `-s -w` flags strip debug symbols for minimal binary size.

## Usage

### Read Latest Message
```bash
./pulse-compat read <channel_id>
```

### Listen for New Message
```bash
./pulse-compat listen <channel_id> [timeout_seconds]
```

Default timeout is 30 seconds. Use 0 for no timeout (not recommended on microcontrollers).

### Send Message
```bash
./pulse-compat send <channel_id> "your message here"
```

## Configuration

Edit the constants in `core.go`:

```go
const (
    timeout       = 5 * time.Second    // Connection timeout
    listenTimeout = 30 * time.Second   // Default listen timeout
    historyLimit  = 5                   // Max messages to fetch
)

var (
    secret = "super-secret-key"         // Encryption secret
    relays = []string{...}              // Nostr relays
)
```

## Memory Footprint

- Binary size: ~4-5 MB (with `-s -w` flags)
- Runtime: ~10-20 MB depending on relay connections
- Goroutines: Limited to number of relays (typically 2)

## Security Notes

- Uses AES-256-GCM for encryption
- SHA-256 key derivation from ID + secret
- No plaintext logs
