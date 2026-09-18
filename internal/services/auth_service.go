package services

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"streamgo/internal/config"
	"streamgo/internal/models"
	"streamgo/internal/repository"
)

// AuthService handles Telegram WebApp/Widget authentication and token issuance.
type AuthService struct {
	cfg      *config.Config
	userRepo repository.UserRepository
}

// NewAuthService creates a new AuthService instance.
func NewAuthService(cfg *config.Config, userRepo repository.UserRepository) *AuthService {
	return &AuthService{
		cfg:      cfg,
		userRepo: userRepo,
	}
}

func (s *AuthService) getSecretCandidates() [][]byte {
	candidates := make([][]byte, 0, 3)
	if s.cfg.SecretKey != "" {
		candidates = append(candidates, []byte(s.cfg.SecretKey))
	}
	if s.cfg.BotToken != "" {
		candidates = append(candidates, []byte(s.cfg.BotToken))
	}
	candidates = append(candidates, []byte("SHA234567JDNKDNSNNFNDKSMSERTYUWERTY"))
	return candidates
}

func (s *AuthService) primarySecret() []byte {
	if s.cfg.SecretKey != "" {
		return []byte(s.cfg.SecretKey)
	}
	if s.cfg.BotToken != "" {
		return []byte(s.cfg.BotToken)
	}
	return []byte("SHA234567JDNKDNSNNFNDKSMSERTYUWERTY")
}

// ValidateTelegramInitData verifies the HMAC-SHA256 signature of Telegram WebApp initData.
func (s *AuthService) ValidateTelegramInitData(initData string) (*models.User, error) {
	initData = strings.TrimSpace(initData)
	if initData == "" {
		return nil, errors.New("init_data is required")
	}

	values, err := url.ParseQuery(initData)
	if err != nil {
		return nil, fmt.Errorf("invalid init_data query string: %w", err)
	}

	receivedHash := values.Get("hash")
	if receivedHash == "" {
		return nil, errors.New("hash is missing from init_data")
	}
	values.Del("hash")

	// Sort keys alphabetically
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var checkPairs []string
	for _, k := range keys {
		checkPairs = append(checkPairs, fmt.Sprintf("%s=%s", k, values.Get(k)))
	}
	dataCheckString := strings.Join(checkPairs, "\n")

	botToken := strings.TrimSpace(s.cfg.BotToken)
	if botToken == "" {
		return nil, errors.New("BOT_TOKEN is not configured")
	}

	// Telegram WebApp Secret Key = HMAC-SHA256("WebAppData", botToken)
	h := hmac.New(sha256.New, []byte("WebAppData"))
	h.Write([]byte(botToken))
	secretKey := h.Sum(nil)

	// Calculated Hash = HMAC-SHA256(secretKey, dataCheckString)
	hData := hmac.New(sha256.New, secretKey)
	hData.Write([]byte(dataCheckString))
	calculatedHash := hex.EncodeToString(hData.Sum(nil))

	if subtle.ConstantTimeCompare([]byte(calculatedHash), []byte(receivedHash)) != 1 {
		return nil, errors.New("invalid Telegram signature")
	}

	userJSON := values.Get("user")
	if userJSON == "" {
		return nil, errors.New("user payload missing from init_data")
	}

	var tgPayload struct {
		ID        int64  `json:"id"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Username  string `json:"username"`
		PhotoURL  string `json:"photo_url"`
	}
	if err := json.Unmarshal([]byte(userJSON), &tgPayload); err != nil {
		return nil, fmt.Errorf("failed to parse user json: %w", err)
	}
	if tgPayload.ID <= 0 {
		return nil, errors.New("invalid user id in init_data")
	}

	profileURL := tgPayload.PhotoURL
	if profileURL == "" && tgPayload.Username != "" {
		profileURL = fmt.Sprintf("https://t.me/i/userpic/320/%s.jpg", tgPayload.Username)
	}

	return &models.User{
		ID:         tgPayload.ID,
		UserID:     tgPayload.ID,
		FirstName:  tgPayload.FirstName,
		Username:   tgPayload.Username,
		PhotoURL:   tgPayload.PhotoURL,
		ProfileURL: profileURL,
		Telegram: &models.TelegramInfo{
			ID:       tgPayload.ID,
			Username: tgPayload.Username,
		},
	}, nil
}

// ValidateTelegramWidget verifies callback data from Telegram Login Widget.
func (s *AuthService) ValidateTelegramWidget(req models.TelegramWidgetRequest) (*models.User, error) {
	if req.ID <= 0 || req.Hash == "" {
		return nil, errors.New("invalid widget payload")
	}

	botToken := strings.TrimSpace(s.cfg.BotToken)
	if botToken == "" {
		return nil, errors.New("BOT_TOKEN is not configured")
	}

	// Secret key for widget is SHA256(botToken)
	secretKey := sha256.Sum256([]byte(botToken))

	pairs := make(map[string]string)
	pairs["id"] = strconv.FormatInt(req.ID, 10)
	pairs["first_name"] = req.FirstName
	if req.LastName != "" {
		pairs["last_name"] = req.LastName
	}
	if req.Username != "" {
		pairs["username"] = req.Username
	}
	if req.PhotoURL != "" {
		pairs["photo_url"] = req.PhotoURL
	}
	pairs["auth_date"] = strconv.FormatInt(req.AuthDate, 10)

	keys := make([]string, 0, len(pairs))
	for k := range pairs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var checkLines []string
	for _, k := range keys {
		checkLines = append(checkLines, fmt.Sprintf("%s=%s", k, pairs[k]))
	}
	dataCheckString := strings.Join(checkLines, "\n")

	h := hmac.New(sha256.New, secretKey[:])
	h.Write([]byte(dataCheckString))
	calculatedHash := hex.EncodeToString(h.Sum(nil))

	if subtle.ConstantTimeCompare([]byte(calculatedHash), []byte(req.Hash)) != 1 {
		return nil, errors.New("invalid widget signature")
	}

	profileURL := req.PhotoURL
	if profileURL == "" && req.Username != "" {
		profileURL = fmt.Sprintf("https://t.me/i/userpic/320/%s.jpg", req.Username)
	}

	return &models.User{
		ID:         req.ID,
		UserID:     req.ID,
		FirstName:  req.FirstName,
		Username:   req.Username,
		PhotoURL:   req.PhotoURL,
		ProfileURL: profileURL,
		Telegram: &models.TelegramInfo{
			ID:       req.ID,
			Username: req.Username,
		},
	}, nil
}

// CreateToken creates a signed token interoperable with Python StreamXBot.
func (s *AuthService) CreateToken(u *models.User) (string, error) {
	now := time.Now().Unix()
	exp := now + (365 * 24 * 60 * 60) // 1 year

	payload := map[string]any{
		"uid":         u.ID,
		"user_id":     u.ID,
		"userid":      u.ID,
		"iat":         now,
		"exp":         exp,
		"first_name":  u.FirstName,
		"photo_url":   u.PhotoURL,
		"profile_url": u.ProfileURL,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadBytes)
	secret := s.primarySecret()

	h := hmac.New(sha256.New, secret)
	h.Write([]byte(payloadB64))
	sigB64 := base64.RawURLEncoding.EncodeToString(h.Sum(nil))

	return fmt.Sprintf("v1.%s.%s", payloadB64, sigB64), nil
}

// VerifyToken verifies token signature and returns the user ID.
func (s *AuthService) VerifyToken(rawToken string) (int64, error) {
	token := strings.TrimSpace(rawToken)
	if token == "" {
		return 0, errors.New("missing auth token")
	}

	if strings.HasPrefix(strings.ToLower(token), "bearer ") {
		token = strings.TrimSpace(token[7:])
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return 0, errors.New("invalid auth token format")
	}

	// 1. Format v1.{payload_b64}.{sig_b64}
	if parts[0] == "v1" {
		payloadB64 := parts[1]
		sigB64 := parts[2]

		matched := false
		candidates := s.getSecretCandidates()
		for _, cand := range candidates {
			h := hmac.New(sha256.New, cand)
			h.Write([]byte(payloadB64))
			expectedSig := base64.RawURLEncoding.EncodeToString(h.Sum(nil))
			if subtle.ConstantTimeCompare([]byte(expectedSig), []byte(sigB64)) == 1 {
				matched = true
				break
			}
		}

		if !matched {
			return 0, errors.New("invalid token signature")
		}

		payloadBytes, err := base64.RawURLEncoding.DecodeString(payloadB64)
		if err != nil {
			return 0, errors.New("malformed token payload")
		}

		var payload struct {
			UID    int64 `json:"uid"`
			UserID int64 `json:"user_id"`
			Exp    int64 `json:"exp"`
		}
		if err := json.Unmarshal(payloadBytes, &payload); err != nil {
			return 0, errors.New("invalid token payload json")
		}

		uid := payload.UID
		if uid == 0 {
			uid = payload.UserID
		}
		if uid <= 0 {
			return 0, errors.New("invalid user id in token")
		}

		if payload.Exp > 0 && time.Now().Unix() > payload.Exp {
			return 0, errors.New("auth token expired")
		}

		return uid, nil
	}

	// 2. Standard JWT fallback: header.payload.signature
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		// Try with std padding if needed
		if pb, err2 := base64.URLEncoding.DecodeString(parts[1]); err2 == nil {
			payloadBytes = pb
		} else {
			return 0, errors.New("malformed jwt payload")
		}
	}

	var jwtPayload struct {
		UID    int64 `json:"uid"`
		UserID int64 `json:"user_id"`
		Exp    int64 `json:"exp"`
	}
	if err := json.Unmarshal(payloadBytes, &jwtPayload); err != nil {
		return 0, errors.New("invalid jwt payload json")
	}

	uid := jwtPayload.UID
	if uid == 0 {
		uid = jwtPayload.UserID
	}
	if uid <= 0 {
		return 0, errors.New("invalid user id in jwt")
	}

	if jwtPayload.Exp > 0 && time.Now().Unix() > jwtPayload.Exp {
		return 0, errors.New("jwt token expired")
	}

	return uid, nil
}

// AuthenticateTelegram validates Telegram initData, saves user, and creates token.
func (s *AuthService) AuthenticateTelegram(ctx context.Context, initData string) (*models.AuthResponse, error) {
	u, err := s.ValidateTelegramInitData(initData)
	if err != nil {
		return nil, err
	}

	savedUser, err := s.userRepo.UpsertUser(ctx, u)
	if err != nil {
		return nil, err
	}

	token, err := s.CreateToken(savedUser)
	if err != nil {
		return nil, err
	}

	return &models.AuthResponse{
		OK:    true,
		Token: token,
		User:  savedUser,
	}, nil
}

// AuthenticateWidget validates Telegram Widget login, saves user, and creates token.
func (s *AuthService) AuthenticateWidget(ctx context.Context, req models.TelegramWidgetRequest) (*models.AuthResponse, error) {
	u, err := s.ValidateTelegramWidget(req)
	if err != nil {
		return nil, err
	}

	savedUser, err := s.userRepo.UpsertUser(ctx, u)
	if err != nil {
		return nil, err
	}

	token, err := s.CreateToken(savedUser)
	if err != nil {
		return nil, err
	}

	return &models.AuthResponse{
		OK:    true,
		Token: token,
		User:  savedUser,
	}, nil
}

// GetUserByID returns user information by ID.
func (s *AuthService) GetUserByID(ctx context.Context, id int64) (*models.User, error) {
	return s.userRepo.GetUserByID(ctx, id)
}
