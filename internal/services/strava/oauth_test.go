package strava

import (
	"testing"
	"time"

	"github.com/johanbrandhorst/trail-tools/internal/store"
)

func TestOAuthState(t *testing.T) {
	o := &OAuth{stateKey: []byte("secret")}
	userID, err := store.StringToUUID("8f6c1f54-4b1e-4f4e-9a0e-2f0b0a8f6c1f")
	if err != nil {
		t.Fatal(err)
	}
	state := o.signState(userID, "nonce", time.Now().Add(time.Minute))
	got, err := o.verifyState(state, "nonce")
	if err != nil || got != userID {
		t.Fatalf("verifyState: got %v, %v", got, err)
	}
	if _, err := o.verifyState(state, "other-nonce"); err == nil {
		t.Error("state from another browser should be rejected")
	}
	if _, err := (&OAuth{stateKey: []byte("other")}).verifyState(state, "nonce"); err == nil {
		t.Error("state with an invalid signature should be rejected")
	}
	expired := o.signState(userID, "nonce", time.Now().Add(-time.Minute))
	if _, err := o.verifyState(expired, "nonce"); err == nil {
		t.Error("expired state should be rejected")
	}
	if _, err := o.verifyState("garbage", "nonce"); err == nil {
		t.Error("malformed state should be rejected")
	}
}
