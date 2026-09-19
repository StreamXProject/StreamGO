package services

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"testing"
	"time"

	"streamgo/internal/models"
)

func TestDiscordSessionLifecycle(t *testing.T) {
	svc := NewDiscordService()
	ctx := context.Background()

	sess, err := svc.CreateWSSession(ctx)
	if err != nil {
		t.Fatalf("failed to create WS session: %v", err)
	}

	if sess.SessionID == "" {
		t.Fatalf("expected non-empty session ID")
	}
	if sess.PrivKey == nil {
		t.Fatalf("expected non-nil RSA private key")
	}
	if sess.PubKeyB64 == "" {
		t.Fatalf("expected non-empty public key string")
	}

	// Verify session is discoverable in service
	status, err := svc.GetStatus(sess.SessionID)
	if err != nil {
		t.Fatalf("failed to get session status: %v", err)
	}
	if status.SessionID != sess.SessionID {
		t.Errorf("expected session ID %s, got %s", sess.SessionID, status.SessionID)
	}

	// Test OAEP decryption with session's private key
	testMsg := "test-secret-nonce-or-token"
	encrypted, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, &sess.PrivKey.PublicKey, []byte(testMsg), nil)
	if err != nil {
		t.Fatalf("failed to encrypt OAEP: %v", err)
	}
	encB64 := base64.StdEncoding.EncodeToString(encrypted)

	decrypted, err := sess.decryptOAEP(encB64)
	if err != nil {
		t.Fatalf("failed to decrypt OAEP: %v", err)
	}
	if string(decrypted) != testMsg {
		t.Errorf("expected decrypted %s, got %s", testMsg, string(decrypted))
	}

	// Test cancellation
	err = svc.CancelSession(sess.SessionID)
	if err != nil {
		t.Fatalf("failed to cancel session: %v", err)
	}

	time.Sleep(10 * time.Millisecond)
	if !sess.Cancelled {
		t.Errorf("expected session to be marked as cancelled")
	}
}

func TestResolveExternalAssetsEmpty(t *testing.T) {
	svc := NewDiscordService()
	ctx := context.Background()

	// Empty URLs or token should return empty slice without error
	res, err := svc.ResolveExternalAssets(ctx, models.ExternalAssetsRequest{
		URLs:  []string{},
		Token: "",
	})
	if err != nil {
		t.Fatalf("unexpected error on empty request: %v", err)
	}
	if len(res) != 0 {
		t.Errorf("expected 0 results, got %d", len(res))
	}
}
