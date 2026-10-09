package main

// The bare minimum of Nostr that zaps (NIP-57) need: events, ids, signatures and publishing to a
// relay. It is built here on what the project already depends on (btcec/v2 and gorilla/websocket)
// instead of pulling a full Nostr library, which would bring dozens of modules into a payment server.

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/gorilla/websocket"
)

type NostrEvent struct {
	ID        string     `json:"id"`
	PubKey    string     `json:"pubkey"`
	CreatedAt int64      `json:"created_at"`
	Kind      int        `json:"kind"`
	Tags      [][]string `json:"tags"`
	Content   string     `json:"content"`
	Sig       string     `json:"sig"`
}

// serialize returns the NIP-01 [0, pubkey, created_at, kind, tags, content] array, whose sha256 is the id.
func (e *NostrEvent) serialize() []byte {
	tags := e.Tags
	if tags == nil {
		tags = [][]string{}
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.Encode([]interface{}{0, e.PubKey, e.CreatedAt, e.Kind, tags, e.Content})
	out := bytes.TrimSuffix(buf.Bytes(), []byte("\n"))
	// encoding/json escapes U+2028 and U+2029; NIP-01 wants them as they are.
	out = bytes.ReplaceAll(out, []byte(`\u2028`), []byte("\u2028"))
	out = bytes.ReplaceAll(out, []byte(`\u2029`), []byte("\u2029"))
	return out
}

func (e *NostrEvent) computeID() string {
	sum := sha256.Sum256(e.serialize())
	return hex.EncodeToString(sum[:])
}

// CheckSignature checks that the id matches the content and that the signature is the pubkey's.
func (e *NostrEvent) CheckSignature() bool {
	if e.ID != e.computeID() {
		return false
	}
	pk, err := hex.DecodeString(e.PubKey)
	if err != nil || len(pk) != 32 {
		return false
	}
	pub, err := schnorr.ParsePubKey(pk)
	if err != nil {
		return false
	}
	sigb, err := hex.DecodeString(e.Sig)
	if err != nil {
		return false
	}
	sig, err := schnorr.ParseSignature(sigb)
	if err != nil {
		return false
	}
	id, _ := hex.DecodeString(e.ID)
	return sig.Verify(id, pub)
}

func (e *NostrEvent) Sign(sk *btcec.PrivateKey) error {
	e.PubKey = xOnly(sk)
	e.ID = e.computeID()
	id, _ := hex.DecodeString(e.ID)
	sig, err := schnorr.Sign(sk, id)
	if err != nil {
		return err
	}
	e.Sig = hex.EncodeToString(sig.Serialize())
	return nil
}

func (e *NostrEvent) firstTag(name string) []string {
	for _, t := range e.Tags {
		if len(t) >= 1 && t[0] == name {
			return t
		}
	}
	return nil
}

func (e *NostrEvent) countTags(name string) int {
	n := 0
	for _, t := range e.Tags {
		if len(t) >= 1 && t[0] == name {
			n++
		}
	}
	return n
}

func xOnly(sk *btcec.PrivateKey) string {
	return hex.EncodeToString(schnorr.SerializePubKey(sk.PubKey()))
}

func parseNostrKey(hexKey string) (*btcec.PrivateKey, error) {
	b, err := hex.DecodeString(strings.TrimSpace(hexKey))
	if err != nil || len(b) != 32 {
		return nil, errors.New("NOSTR_PRIVATE_KEY must be 64 hex characters")
	}
	sk, _ := btcec.PrivKeyFromBytes(b)
	return sk, nil
}

// publishEvent sends the event to a relay and waits for its OK (or the deadline).
func publishEvent(relayURL string, ev *NostrEvent, timeout time.Duration) error {
	dialer := websocket.Dialer{HandshakeTimeout: timeout}
	conn, _, err := dialer.Dial(relayURL, nil)
	if err != nil {
		return err
	}
	defer conn.Close()
	deadline := time.Now().Add(timeout)
	conn.SetWriteDeadline(deadline)
	if err := conn.WriteJSON([]interface{}{"EVENT", ev}); err != nil {
		return err
	}
	conn.SetReadDeadline(deadline)
	for {
		var msg []json.RawMessage
		if err := conn.ReadJSON(&msg); err != nil {
			return err
		}
		if len(msg) < 3 {
			continue
		}
		var label, id string
		json.Unmarshal(msg[0], &label)
		json.Unmarshal(msg[1], &id)
		if label != "OK" || id != ev.ID {
			continue
		}
		var ok bool
		json.Unmarshal(msg[2], &ok)
		if ok {
			return nil
		}
		reason := ""
		if len(msg) > 3 {
			json.Unmarshal(msg[3], &reason)
		}
		return fmt.Errorf("relay refused: %s", reason)
	}
}
