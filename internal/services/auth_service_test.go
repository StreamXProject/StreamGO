package services_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"testing"

	"streamgo/internal/config"
	"streamgo/internal/models"
	"streamgo/internal/services"
)

type mockUserRepo struct {
	users map[int64]*models.User
}

func (m *mockUserRepo) UpsertUser(ctx context.Context, user *models.User) (*models.User, error) {
	m.users[user.ID] = user
	return user, nil
}

func (m *mockUserRepo) GetUserByID(ctx context.Context, id int64) (*models.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, nil
	}
	return u, nil
}

func TestTelegramInitDataAuth(t *testing.T) {
	botToken := "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"
	cfg := &config.Config{
		BotToken:  botToken,
		SecretKey: "supersecretkey123",
	}

	repo := &mockUserRepo{users: make(map[int64]*models.User)}
	authSvc := services.NewAuthService(cfg, repo)

	// Construct valid Telegram initData
	userJSON := `{"id":987654321,"first_name":"Test","last_name":"User","username":"testuser","photo_url":"https://example.com/p.jpg"}`
	authDate := "1726000000"

	// Prepare data_check_string
	dataCheckString := fmt.Sprintf("auth_date=%s\nuser=%s", authDate, userJSON)

	// HMAC-SHA256("WebAppData", botToken)
	h := hmac.New(sha256.New, []byte("WebAppData"))
	h.Write([]byte(botToken))
	secretKey := h.Sum(nil)

	// HMAC-SHA256(secretKey, dataCheckString)
	hData := hmac.New(sha256.New, secretKey)
	hData.Write([]byte(dataCheckString))
	calculatedHash := hex.EncodeToString(hData.Sum(nil))

	initData := fmt.Sprintf("auth_date=%s&user=%s&hash=%s",
		url.QueryEscape(authDate),
		url.QueryEscape(userJSON),
		calculatedHash,
	)

	ctx := context.Background()
	authResp, err := authSvc.AuthenticateTelegram(ctx, initData)
	if err != nil {
		t.Fatalf("AuthenticateTelegram failed: %v", err)
	}

	if !authResp.OK {
		t.Fatalf("expected OK=true, got false")
	}
	if authResp.User.ID != 987654321 {
		t.Fatalf("expected user id 987654321, got %d", authResp.User.ID)
	}
	if authResp.Token == "" {
		t.Fatalf("expected non-empty token")
	}

	// Verify the token
	userID, err := authSvc.VerifyToken(authResp.Token)
	if err != nil {
		t.Fatalf("VerifyToken failed: %v", err)
	}
	if userID != 987654321 {
		t.Fatalf("expected verified user ID 987654321, got %d", userID)
	}
}

func TestVerifyTokenTampered(t *testing.T) {
	cfg := &config.Config{
		SecretKey: "supersecretkey123",
	}
	repo := &mockUserRepo{users: make(map[int64]*models.User)}
	authSvc := services.NewAuthService(cfg, repo)

	// Create valid token
	u := &models.User{ID: 112233, FirstName: "Alice"}
	tok, err := authSvc.CreateToken(u)
	if err != nil {
		t.Fatalf("CreateToken failed: %v", err)
	}

	// Tampered token
	tampered := tok + "tampered"
	if _, err := authSvc.VerifyToken(tampered); err == nil {
		t.Fatalf("expected error for tampered token, got nil")
	}
}
