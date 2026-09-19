package services_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
	"testing"

	"streamgo/internal/config"
	"streamgo/internal/models"
	"streamgo/internal/services"
)

type mockUserRepo struct {
	users     map[int64]*models.User
	otps      map[int64]*models.RegistrationOTP
	sessions  map[string]*models.BotAuthSession
	fcmTokens map[int64]string
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		users:     make(map[int64]*models.User),
		otps:      make(map[int64]*models.RegistrationOTP),
		sessions:  make(map[string]*models.BotAuthSession),
		fcmTokens: make(map[int64]string),
	}
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

func (m *mockUserRepo) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	for _, u := range m.users {
		if u.Username == username {
			return u, nil
		}
	}
	return nil, nil
}

func (m *mockUserRepo) UpdateUserCredentials(ctx context.Context, userID int64, username string, pwd *models.PasswordHash) error {
	return nil
}

func (m *mockUserRepo) UpdateIntegrations(ctx context.Context, userID int64, req models.IntegrationsUpdateRequest) (*models.UserIntegrations, error) {
	return &models.UserIntegrations{}, nil
}

func (m *mockUserRepo) SaveRegistrationOTP(ctx context.Context, otp *models.RegistrationOTP) error {
	m.otps[otp.ID] = otp
	return nil
}

func (m *mockUserRepo) GetRegistrationOTP(ctx context.Context, userID int64) (*models.RegistrationOTP, error) {
	return m.otps[userID], nil
}

func (m *mockUserRepo) DeleteRegistrationOTP(ctx context.Context, userID int64) error {
	delete(m.otps, userID)
	return nil
}

func (m *mockUserRepo) SaveOIDCSession(ctx context.Context, session *models.OIDCSession) error {
	return nil
}

func (m *mockUserRepo) GetAndDeleteOIDCSession(ctx context.Context, state string) (*models.OIDCSession, error) {
	return nil, nil
}

func (m *mockUserRepo) GetOwnerPassword(ctx context.Context) (*models.PasswordHash, error) {
	return nil, nil
}

func (m *mockUserRepo) SetOwnerPassword(ctx context.Context, pwd *models.PasswordHash, updatedBy int64) error {
	return nil
}

func (m *mockUserRepo) UpdateFCMToken(ctx context.Context, userID int64, fcmToken string) error {
	m.fcmTokens[userID] = fcmToken
	return nil
}

func (m *mockUserRepo) SaveBotAuthSession(ctx context.Context, session *models.BotAuthSession) error {
	m.sessions[session.ID] = session
	return nil
}

func (m *mockUserRepo) GetBotAuthSession(ctx context.Context, sessionID string) (*models.BotAuthSession, error) {
	return m.sessions[sessionID], nil
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

func TestGuestPasswordAndGuestToken(t *testing.T) {
	ctx := context.Background()
	cfg := &config.Config{
		SecretKey:     "supersecretkey123",
		GuestPassword: "testguestpassword",
	}
	repo := &mockUserRepo{users: make(map[int64]*models.User)}
	authSvc := services.NewAuthService(cfg, repo)

	// 1. Check guest password verification
	if !authSvc.VerifyGuestPassword(ctx, "testguestpassword") {
		t.Fatalf("expected VerifyGuestPassword to return true for correct password")
	}
	if authSvc.VerifyGuestPassword(ctx, "wrongpassword") {
		t.Fatalf("expected VerifyGuestPassword to return false for wrong password")
	}

	// 2. Login with server/guest password
	tok, err := authSvc.LoginWithServerPassword(ctx, "testguestpassword")
	if err != nil {
		t.Fatalf("LoginWithServerPassword failed: %v", err)
	}
	if tok == "" {
		t.Fatalf("expected non-empty guest token")
	}

	// 3. Verify guest token claims
	claims, err := authSvc.VerifyTokenClaims(tok)
	if err != nil {
		t.Fatalf("VerifyTokenClaims failed for guest token: %v", err)
	}
	if !claims.IsGuest {
		t.Fatalf("expected claims.IsGuest to be true")
	}
	if claims.UserID != 0 {
		t.Fatalf("expected claims.UserID to be 0 for guest, got %d", claims.UserID)
	}

	// 4. Setup status
	status, err := authSvc.GetSetupStatus(ctx)
	if err != nil {
		t.Fatalf("GetSetupStatus failed: %v", err)
	}
	if !status.Configured {
		t.Fatalf("expected status.Configured to be true when GuestPassword is set")
	}
	if status.NeedsSetup {
		t.Fatalf("expected status.NeedsSetup to be false")
	}
}

func TestDirectManualUserRegistrationAndLogin(t *testing.T) {
	ctx := context.Background()
	cfg := &config.Config{
		SecretKey: "supersecretkey123",
	}
	repo := &mockUserRepo{users: make(map[int64]*models.User)}
	authSvc := services.NewAuthService(cfg, repo)

	// 1. Direct manual registration without Telegram UserID
	regReq := models.RegisterRequest{
		Username: "misfittest",
		Password: "mypassword123",
	}
	resp, err := authSvc.RegisterAccount(ctx, regReq)
	if err != nil {
		t.Fatalf("RegisterAccount failed: %v", err)
	}
	if resp.UserID <= 0 {
		t.Fatalf("expected auto-generated positive UserID, got %d", resp.UserID)
	}
	if resp.Token == "" {
		t.Fatalf("expected token returned on direct registration")
	}

	// 2. Verify token
	claims, err := authSvc.VerifyTokenClaims(resp.Token)
	if err != nil {
		t.Fatalf("VerifyTokenClaims failed: %v", err)
	}
	if claims.UserID != resp.UserID {
		t.Fatalf("expected verified UserID %d, got %d", resp.UserID, claims.UserID)
	}
	if claims.IsGuest {
		t.Fatalf("expected registered user to not be guest")
	}

	// 3. Manual login with password
	loginResp, err := authSvc.LoginWithPassword(ctx, "misfittest", "mypassword123")
	if err != nil {
		t.Fatalf("LoginWithPassword failed: %v", err)
	}
	if loginResp.UserID != resp.UserID {
		t.Fatalf("expected login user ID %d, got %d", resp.UserID, loginResp.UserID)
	}

	// 4. Bad password
	if _, err := authSvc.LoginWithPassword(ctx, "misfittest", "wrongpw"); err == nil {
		t.Fatalf("expected error for wrong password, got nil")
	}
}

type mockAccessRepoForAuth struct {
	policy    *models.AccessPolicy
	invites   map[string]*models.InviteCode
	allowlist map[int64]bool
	bypass    map[int64]bool
}

func newMockAccessRepoForAuth() *mockAccessRepoForAuth {
	return &mockAccessRepoForAuth{
		policy: &models.AccessPolicy{
			RegistrationMode: "open",
		},
		invites:   make(map[string]*models.InviteCode),
		allowlist: make(map[int64]bool),
		bypass:    make(map[int64]bool),
	}
}

func (m *mockAccessRepoForAuth) GetPolicy(ctx context.Context) (*models.AccessPolicy, error) {
	return m.policy, nil
}
func (m *mockAccessRepoForAuth) SavePolicy(ctx context.Context, p *models.AccessPolicy) error {
	m.policy = p
	return nil
}
func (m *mockAccessRepoForAuth) CreateInvite(ctx context.Context, inv *models.InviteCode) error {
	m.invites[inv.Code] = inv
	return nil
}
func (m *mockAccessRepoForAuth) GetInvite(ctx context.Context, code string) (*models.InviteCode, error) {
	return m.invites[code], nil
}
func (m *mockAccessRepoForAuth) ConsumeInvite(ctx context.Context, code string) error {
	if inv, ok := m.invites[code]; ok {
		inv.UsedCount++
	}
	return nil
}
func (m *mockAccessRepoForAuth) RevokeInvite(ctx context.Context, code string) bool { return true }
func (m *mockAccessRepoForAuth) ListInvites(ctx context.Context, d bool) ([]*models.InviteCode, error) {
	return nil, nil
}
func (m *mockAccessRepoForAuth) AddToAllowlist(ctx context.Context, e *models.AllowlistEntry) error {
	m.allowlist[e.UserID] = true
	return nil
}
func (m *mockAccessRepoForAuth) RemoveFromAllowlist(ctx context.Context, u int64) bool { return true }
func (m *mockAccessRepoForAuth) IsInAllowlist(ctx context.Context, u int64) bool {
	return m.allowlist[u]
}
func (m *mockAccessRepoForAuth) ListAllowlist(ctx context.Context) ([]*models.AllowlistEntry, error) {
	return nil, nil
}
func (m *mockAccessRepoForAuth) AddToBypass(ctx context.Context, e *models.AllowlistEntry) error {
	m.bypass[e.UserID] = true
	return nil
}
func (m *mockAccessRepoForAuth) RemoveFromBypass(ctx context.Context, u int64) bool { return true }
func (m *mockAccessRepoForAuth) IsInBypass(ctx context.Context, u int64) bool {
	return m.bypass[u]
}
func (m *mockAccessRepoForAuth) ListBypass(ctx context.Context) ([]*models.AllowlistEntry, error) {
	return nil, nil
}
func (m *mockAccessRepoForAuth) ListUsers(ctx context.Context, q, s string, l, sk int) ([]*models.User, int64, error) {
	return nil, 0, nil
}
func (m *mockAccessRepoForAuth) SetUserStatus(ctx context.Context, u int64, s, r string) error {
	return nil
}
func (m *mockAccessRepoForAuth) RevokeUserSessions(ctx context.Context, u int64) (int, error) {
	return 0, nil
}

func TestOTPRegistrationFlow(t *testing.T) {
	ctx := context.Background()
	cfg := &config.Config{
		SecretKey: "supersecretkey123",
		BotToken:  "test:token",
	}
	userRepo := newMockUserRepo()
	accessRepo := newMockAccessRepoForAuth()
	authSvc := services.NewAuthService(cfg, userRepo)
	authSvc.SetAccessControlRepository(accessRepo)

	// Direct call to validate without pending registration should fail
	_, err := authSvc.ValidateOTP(ctx, models.ValidateOTPRequest{
		UserID: 12345,
		OTP:    "123456",
	})
	if err == nil || err.Error() != "no pending registration found" {
		t.Fatalf("expected 'no pending registration found', got %v", err)
	}

	// Manually save an OTP doc into repo (simulating bot having sent OTP)
	pwdHash, _ := authSvc.HashPassword("securepass")
	userRepo.otps[12345] = &models.RegistrationOTP{
		ID:         12345,
		OTP:        "654321",
		Username:   "telegramuser",
		Password:   pwdHash,
		FirstName:  "Telegram User",
		InviteCode: "INVITE123",
	}
	accessRepo.invites["INVITE123"] = &models.InviteCode{
		Code:    "INVITE123",
		MaxUses: 2,
	}

	// Wrong OTP
	_, err = authSvc.ValidateOTP(ctx, models.ValidateOTPRequest{
		UserID: 12345,
		OTP:    "000000",
	})
	if err == nil || err.Error() != "invalid OTP" {
		t.Fatalf("expected 'invalid OTP', got %v", err)
	}

	// Correct OTP
	resp, err := authSvc.ValidateOTP(ctx, models.ValidateOTPRequest{
		UserID: 12345,
		OTP:    "654321",
	})
	if err != nil {
		t.Fatalf("ValidateOTP failed: %v", err)
	}
	if !resp.OK || resp.Token == "" {
		t.Fatalf("expected successful response with token, got %+v", resp)
	}
	if resp.UserID != 12345 {
		t.Fatalf("expected user ID 12345, got %d", resp.UserID)
	}

	// Check OTP doc was deleted after validation
	if _, exists := userRepo.otps[12345]; exists {
		t.Fatalf("expected registration OTP to be deleted after completion")
	}

	// Check user was created in repository with registered_via
	savedUser, err := userRepo.GetUserByID(ctx, 12345)
	if err != nil || savedUser == nil {
		t.Fatalf("expected saved user in repository")
	}
	if savedUser.Username != "telegramuser" {
		t.Fatalf("expected username telegramuser, got %s", savedUser.Username)
	}
	if savedUser.RegisteredVia == nil {
		t.Fatalf("expected RegisteredVia to be populated")
	}

	// Check invite was consumed
	if accessRepo.invites["INVITE123"].UsedCount != 1 {
		t.Fatalf("expected invite used count 1, got %d", accessRepo.invites["INVITE123"].UsedCount)
	}
}

func TestRegistrationPolicies(t *testing.T) {
	ctx := context.Background()
	cfg := &config.Config{
		SecretKey: "supersecretkey123",
	}
	userRepo := newMockUserRepo()
	accessRepo := newMockAccessRepoForAuth()
	authSvc := services.NewAuthService(cfg, userRepo)
	authSvc.SetAccessControlRepository(accessRepo)

	// 1. Closed mode
	accessRepo.policy.RegistrationMode = "closed"
	_, err := authSvc.RegisterAccount(ctx, models.RegisterRequest{
		Username: "closeduser",
		Password: "password123",
	})
	if err == nil || err.Error() != "registration_closed: New accounts cannot be created at this time" {
		t.Fatalf("expected registration_closed error, got %v", err)
	}

	// 2. Allowlist mode
	accessRepo.policy.RegistrationMode = "allowlist"
	_, err = authSvc.RegisterAccount(ctx, models.RegisterRequest{
		UserID:   99999,
		Username: "allowlistuser",
		Password: "password123",
	})
	if err == nil || err.Error() != "not_allowlisted: Your account is not on the registration allowlist" {
		t.Fatalf("expected not_allowlisted error, got %v", err)
	}

	// Add to allowlist and retry (UserID <= 0 for direct creation)
	accessRepo.allowlist[0] = true
	resp, err := authSvc.RegisterAccount(ctx, models.RegisterRequest{
		UserID:   0,
		Username: "allowlistuser",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("expected success for allowlisted user, got %v", err)
	}
	if resp.UserID <= 0 {
		t.Fatalf("expected positive user ID, got %d", resp.UserID)
	}

	// 3. Invite mode
	accessRepo.policy.RegistrationMode = "invite"
	_, err = authSvc.RegisterAccount(ctx, models.RegisterRequest{
		Username: "inviteuser",
		Password: "password123",
	})
	if err == nil || err.Error() != "invite_required: An invite code is required to register" {
		t.Fatalf("expected invite_required error, got %v", err)
	}

	_, err = authSvc.RegisterAccount(ctx, models.RegisterRequest{
		Username:   "inviteuser",
		Password:   "password123",
		InviteCode: "INVALIDCODE",
	})
	if err == nil || err.Error() != "invite_invalid: That invite code is invalid or has been revoked" {
		t.Fatalf("expected invite_invalid error, got %v", err)
	}

	accessRepo.invites["VALIDCODE"] = &models.InviteCode{
		Code:    "VALIDCODE",
		MaxUses: 1,
	}
	resp, err = authSvc.RegisterAccount(ctx, models.RegisterRequest{
		Username:   "inviteuser",
		Password:   "password123",
		InviteCode: "VALIDCODE",
	})
	if err != nil {
		t.Fatalf("expected success with valid invite code, got %v", err)
	}
	if accessRepo.invites["VALIDCODE"].UsedCount != 1 {
		t.Fatalf("expected invite code used count 1, got %d", accessRepo.invites["VALIDCODE"].UsedCount)
	}
}

func TestBotSessionAndFCMToken(t *testing.T) {
	ctx := context.Background()
	userRepo := newMockUserRepo()
	cfg := &config.Config{
		BotToken:  "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11",
		SecretKey: "supersecretkey123",
	}
	authSvc := services.NewAuthService(cfg, userRepo)

	// 1. Create Bot Session
	res, err := authSvc.CreateBotSession(ctx, "testinvite")
	if err != nil {
		t.Fatalf("CreateBotSession failed: %v", err)
	}
	sessionID, ok := res["session_id"].(string)
	if !ok || sessionID == "" || !strings.HasPrefix(sessionID, "auth_") {
		t.Fatalf("unexpected session_id: %v", res["session_id"])
	}

	// 2. Check pending status
	st, err := authSvc.CheckBotSessionStatus(ctx, sessionID)
	if err != nil {
		t.Fatalf("CheckBotSessionStatus failed: %v", err)
	}
	if st["status"] != "pending" {
		t.Fatalf("expected status pending, got %v", st["status"])
	}

	// 3. Check not found status
	stNotFound, err := authSvc.CheckBotSessionStatus(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("CheckBotSessionStatus failed: %v", err)
	}
	if stNotFound["status"] != "not_found" {
		t.Fatalf("expected status not_found, got %v", stNotFound["status"])
	}

	// 4. Update session to confirmed
	sess := userRepo.sessions[sessionID]
	sess.Status = "confirmed"
	sess.Token = "test_token_123"
	sess.UserID = 999
	sess.Username = "confirmed_user"

	stConfirmed, err := authSvc.CheckBotSessionStatus(ctx, sessionID)
	if err != nil {
		t.Fatalf("CheckBotSessionStatus failed: %v", err)
	}
	if stConfirmed["status"] != "confirmed" || stConfirmed["token"] != "test_token_123" || stConfirmed["user_id"] != int64(999) {
		t.Fatalf("unexpected confirmed status payload: %+v", stConfirmed)
	}

	// 5. Update FCM Token
	err = authSvc.UpdateFCMToken(ctx, 999, "fcm_token_sample_abc")
	if err != nil {
		t.Fatalf("UpdateFCMToken failed: %v", err)
	}
	if userRepo.fcmTokens[999] != "fcm_token_sample_abc" {
		t.Fatalf("expected fcm token stored, got %v", userRepo.fcmTokens[999])
	}
}

