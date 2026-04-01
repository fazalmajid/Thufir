package auth

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
)

// challengeEntry holds the serialised SessionData and an optional userID
// (non-empty only during registration and device-add flows).
type challengeEntry struct {
	SessionDataJSON []byte
	UserID          string
	Expires         time.Time
}

// ChallengeStore is an in-memory, goroutine-safe store for pending WebAuthn
// challenges.  Entries expire after 5 minutes.  A background goroutine
// removes stale entries every minute.
type ChallengeStore struct {
	mu sync.Mutex
	m  map[string]challengeEntry
}

func NewChallengeStore() *ChallengeStore {
	cs := &ChallengeStore{m: make(map[string]challengeEntry)}
	go func() {
		t := time.NewTicker(60 * time.Second)
		for range t.C {
			cs.mu.Lock()
			now := time.Now()
			for k, v := range cs.m {
				if v.Expires.Before(now) {
					delete(cs.m, k)
				}
			}
			cs.mu.Unlock()
		}
	}()
	return cs
}

// NewToken generates a fresh random 32-char hex token for use as a challenge key.
func NewToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("challenge: rand read: " + err.Error())
	}
	return hex.EncodeToString(b)
}

// NewUUID generates a random UUID v4 string.
func NewUUID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("challenge: rand read: " + err.Error())
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// Set stores session data (and an optional userID) under the given token.
func (cs *ChallengeStore) Set(token string, session *webauthn.SessionData, userID string) error {
	raw, err := json.Marshal(session)
	if err != nil {
		return err
	}
	cs.mu.Lock()
	cs.m[token] = challengeEntry{
		SessionDataJSON: raw,
		UserID:          userID,
		Expires:         time.Now().Add(5 * time.Minute),
	}
	cs.mu.Unlock()
	return nil
}

// Get retrieves and removes the entry for the given token.
// Returns (nil, "", false) if the token is missing or expired.
func (cs *ChallengeStore) Get(token string) (*webauthn.SessionData, string, bool) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	e, ok := cs.m[token]
	if !ok || e.Expires.Before(time.Now()) {
		delete(cs.m, token)
		return nil, "", false
	}
	delete(cs.m, token)
	var sd webauthn.SessionData
	if err := json.Unmarshal(e.SessionDataJSON, &sd); err != nil {
		return nil, "", false
	}
	return &sd, e.UserID, true
}
