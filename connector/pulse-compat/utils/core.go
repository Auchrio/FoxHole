package utils

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"sort"
	"sync"
	"time"

	"github.com/nbd-wtf/go-nostr"
)

const (
	timeout       = 5 * time.Second
	listenTimeout = 30 * time.Second
	historyLimit  = 5
)

var (
	Secret = "super-secret-key"
	relays = []string{"wss://relay.damus.io", "wss://nos.lol"}
	sk     string
	pk     string
)

// deriveKey creates AES key from ID and secret
func deriveKey(id string) []byte {
	h := sha256.Sum256([]byte(id + Secret))
	return h[:]
}

// encrypt using AES-256-GCM
func encrypt(text string, key []byte) (string, error) {
	b, _ := aes.NewCipher(key)
	g, _ := cipher.NewGCM(b)
	n := make([]byte, g.NonceSize())
	io.ReadFull(rand.Reader, n)
	return hex.EncodeToString(g.Seal(n, n, []byte(text), nil)), nil
}

// decrypt using AES-256-GCM
func decrypt(hex_data string, key []byte) (string, error) {
	d, err := hex.DecodeString(hex_data)
	if err != nil {
		return "", err
	}
	b, _ := aes.NewCipher(key)
	g, _ := cipher.NewGCM(b)
	ns := g.NonceSize()
	if len(d) < ns {
		return "", fmt.Errorf("short")
	}
	n, c := d[:ns], d[ns:]
	p, err := g.Open(nil, n, c, nil)
	return string(p), err
}

// ReadMessages retrieves latest message
func ReadMessages(id string) error {
	key := deriveKey(id)
	tag := hex.EncodeToString(key)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	msgs := make([]*nostr.Event, 0, historyLimit)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, url := range relays {
		wg.Add(1)
		go func(u string) {
			defer wg.Done()
			r, err := nostr.RelayConnect(ctx, u)
			if err != nil {
				return
			}
			defer r.Close()

			sub, _ := r.Subscribe(ctx, []nostr.Filter{{
				Tags:  nostr.TagMap{"t": []string{tag}},
				Kinds: []int{nostr.KindTextNote},
				Limit: historyLimit,
			}})

			tm := time.After(300 * time.Millisecond)
			for {
				select {
				case ev := <-sub.Events:
					mu.Lock()
					msgs = append(msgs, ev)
					mu.Unlock()
					return
				case <-tm:
					return
				case <-ctx.Done():
					return
				}
			}
		}(url)
	}
	wg.Wait()

	if len(msgs) == 0 {
		return fmt.Errorf("no messages")
	}

	sort.Slice(msgs, func(i, j int) bool {
		return msgs[i].CreatedAt < msgs[j].CreatedAt
	})

	m, err := decrypt(msgs[len(msgs)-1].Content, key)
	if err != nil {
		return err
	}
	fmt.Print(m)
	return nil
}

// ListenMessages waits for new message with timeout
func ListenMessages(id string, timeoutSec int) error {
	key := deriveKey(id)
	tag := hex.EncodeToString(key)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sk := nostr.GeneratePrivateKey()
	pk, _ := nostr.GetPublicKey(sk)

	found := false
	done := make(chan struct{})
	var wg sync.WaitGroup

	for _, url := range relays {
		wg.Add(1)
		go func(u string) {
			defer wg.Done()
			if found {
				return
			}

			r, err := nostr.RelayConnect(ctx, u)
			if err != nil {
				return
			}
			defer r.Close()

			now := nostr.Now()
			sub, _ := r.Subscribe(ctx, []nostr.Filter{{
				Tags:  nostr.TagMap{"t": []string{tag}},
				Kinds: []int{nostr.KindTextNote},
				Since: &now,
			}})

			for ev := range sub.Events {
				if ev.PubKey != pk {
					m, err := decrypt(ev.Content, key)
					if err == nil {
						fmt.Print(m)
						found = true
						close(done)
						cancel()
						return
					}
				}
			}
		}(url)
	}

	go func() {
		wg.Wait()
		if !found {
			close(done)
		}
	}()

	if timeoutSec == 0 {
		// Wait indefinitely
		<-done
		return nil
	}

	select {
	case <-done:
		return nil
	case <-time.After(time.Duration(timeoutSec) * time.Second):
		return fmt.Errorf("timeout")
	}
}

// SendMessage sends encrypted message
func SendMessage(id, text string) error {
	key := deriveKey(id)
	tag := hex.EncodeToString(key)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	enc, err := encrypt(text, key)
	if err != nil {
		return err
	}

	sk := nostr.GeneratePrivateKey()
	pk, _ := nostr.GetPublicKey(sk)

	ev := nostr.Event{
		PubKey:    pk,
		CreatedAt: nostr.Now(),
		Kind:      nostr.KindTextNote,
		Tags:      nostr.Tags{{"t", tag}},
		Content:   enc,
	}
	ev.Sign(sk)

	var wg sync.WaitGroup
	errFlag := false

	for _, url := range relays {
		wg.Add(1)
		go func(u string) {
			defer wg.Done()
			r, err := nostr.RelayConnect(ctx, u)
			if err != nil {
				errFlag = true
				return
			}
			defer r.Close()

			if err := r.Publish(ctx, ev); err != nil {
				errFlag = true
			}
		}(url)
	}
	wg.Wait()

	if errFlag {
		return fmt.Errorf("publish failed")
	}

	fmt.Print("OK")
	return nil
}
