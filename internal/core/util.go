package core

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// generateID creates a unique ID for memories and relations
func generateID() string {
	bytes := make([]byte, 12)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// timeNow returns the current time (useful for testing)
var timeNow = time.Now
