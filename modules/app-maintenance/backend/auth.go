package main

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const sessionCookieName = "bung_session"

// claims is what gets signed into the session JWT. ModuleCodes and
// MustChangePassword are both baked in at login time, so nginx's
// auth_request check and every module's authorization decision are a pure
// in-memory check against the token - no database round trip on every
// request. The trade-off: a role/module change, or a password aging past
// the expiry policy mid-session, only takes effect the next time the user
// logs in (or the token expires).
type claims struct {
	Username           string   `json:"username"`
	ModuleCodes        []string `json:"moduleCodes"`
	MustChangePassword bool     `json:"mustChangePassword"`
	jwt.RegisteredClaims
}

func hashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(b), err
}

func checkPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// passwordExpired reports whether a password last changed at changedAt has
// aged past maxDays (site_settings.password_expiry_days). maxDays <= 0
// means the policy is disabled.
func passwordExpired(changedAt time.Time, maxDays int) bool {
	if maxDays <= 0 {
		return false
	}
	return time.Since(changedAt) > time.Duration(maxDays)*24*time.Hour
}

// newResetToken returns a high-entropy random token (for the reset link)
// and its SHA-256 hash (what actually gets stored - see the comment on
// password_resets in migrate.go). SHA-256 is fine here, unlike for
// passwords: the input is 256 bits of random data, not a guessable human
// password, so there's no brute-force/rainbow-table concern to design
// around with a slow hash like bcrypt.
func newResetToken() (token, tokenHash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	token = hex.EncodeToString(b)
	sum := sha256.Sum256([]byte(token))
	tokenHash = hex.EncodeToString(sum[:])
	return token, tokenHash, nil
}

type authAPI struct {
	store         *Store
	jwtSecret     []byte
	tokenTTL      time.Duration
	limiter       *loginLimiter
	emailSender   EmailSender
	publicBaseURL string
}

func newAuthAPI(store *Store, secret string, ttl time.Duration, emailSender EmailSender, publicBaseURL string) *authAPI {
	return &authAPI{
		store:         store,
		jwtSecret:     []byte(secret),
		tokenTTL:      ttl,
		limiter:       newLoginLimiter(),
		emailSender:   emailSender,
		publicBaseURL: publicBaseURL,
	}
}

func (a *authAPI) signToken(userID, username string, moduleCodes []string, mustChangePassword bool) (string, error) {
	now := time.Now()
	c := claims{
		Username:           username,
		ModuleCodes:        moduleCodes,
		MustChangePassword: mustChangePassword,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(a.tokenTTL)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	return token.SignedString(a.jwtSecret)
}

func (a *authAPI) parseToken(raw string) (*claims, error) {
	var c claims
	token, err := jwt.ParseWithClaims(raw, &c, func(t *jwt.Token) (any, error) {
		return a.jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return &c, nil
}

func (a *authAPI) setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		// Secure is left off so this works over plain http in local/dev
		// docker compose. Turn it on once this sits behind TLS (Cloudflare).
		MaxAge: int(a.tokenTTL.Seconds()),
	})
}

func (a *authAPI) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

// passwordExpiryDays reads the platform's current policy (Settings tab);
// defaults to 60 (matches the DB column default) if it can't be read for
// some reason, rather than failing login/me entirely over it.
func (a *authAPI) passwordExpiryDays() int {
	st, err := a.store.GetSiteSettings()
	if err != nil {
		log.Printf("read password_expiry_days: %v", err)
		return 60
	}
	return st.PasswordExpiryDays
}

func (a *authAPI) authUser(userID string) (*AuthUser, error) {
	u, err := a.store.GetUser(userID)
	if err != nil {
		return nil, err
	}
	modules, err := a.store.UserModules(userID)
	if err != nil {
		return nil, err
	}
	mustChange := passwordExpired(u.PasswordChangedAt, a.passwordExpiryDays())
	return &AuthUser{ID: u.ID, Username: u.Username, FullName: u.FullName, Email: u.Email, Modules: modules, MustChangePassword: mustChange}, nil
}

// login verifies credentials and, on success, sets the session cookie and
// returns the authenticated user + the modules they can access.
//
// Failed attempts are rate-limited per username+IP (see ratelimit.go) and
// every attempt - success or failure - is written to the audit log.
func (a *authAPI) login(w http.ResponseWriter, r *http.Request) {
	var in loginRequest
	if err := decodeJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}

	ip := clientIP(r)
	limitKey := in.Username + "|" + ip

	if blocked, retryAfter := a.limiter.blocked(limitKey); blocked {
		minutes := int(retryAfter.Minutes()) + 1
		a.audit("auth.login_blocked", "user", "", in.Username, ip)
		writeErr(w, http.StatusTooManyRequests, fmt.Sprintf("too many failed attempts, try again in %d minute(s)", minutes))
		return
	}

	fail := func(msg string) {
		a.limiter.recordFailure(limitKey)
		a.audit("auth.login_failed", "user", "", in.Username, ip)
		writeErr(w, http.StatusUnauthorized, msg)
	}

	user, passwordHash, err := a.store.GetUserByUsername(in.Username)
	if errors.Is(err, sql.ErrNoRows) || errors.Is(err, ErrNotFound) {
		fail("invalid username or password")
		return
	}
	if err != nil {
		handleErr(w, err)
		return
	}
	if !user.IsActive {
		fail("account is disabled")
		return
	}
	if passwordHash == "" || !checkPassword(passwordHash, in.Password) {
		fail("invalid username or password")
		return
	}

	modules, err := a.store.UserModules(user.ID)
	if err != nil {
		handleErr(w, err)
		return
	}
	codes := make([]string, len(modules))
	for i, m := range modules {
		codes[i] = m.Code
	}
	mustChange := passwordExpired(user.PasswordChangedAt, a.passwordExpiryDays())

	token, err := a.signToken(user.ID, user.Username, codes, mustChange)
	if err != nil {
		handleErr(w, err)
		return
	}
	a.limiter.recordSuccess(limitKey)
	a.setSessionCookie(w, token)
	a.audit("auth.login", "user", user.ID, user.Username, ip)
	writeJSON(w, http.StatusOK, &AuthUser{
		ID: user.ID, Username: user.Username, FullName: user.FullName, Email: user.Email,
		Modules: modules, MustChangePassword: mustChange,
	})
}

// audit is a small helper for the auth endpoints, which run outside
// nginx's auth_request gate (see nginx.conf) and so don't have
// X-User-Id/X-Username headers to read like writeAudit (handlers.go) does.
func (a *authAPI) audit(action, entityType, entityID, username, ip string) {
	e := &AuditEntry{
		ActorUserID:   entityID,
		ActorUsername: username,
		ModuleCode:    "app-maintenance",
		Action:        action,
		EntityType:    entityType,
		EntityID:      entityID,
		IPAddress:     ip,
	}
	if err := a.store.InsertAuditLog(e); err != nil {
		log.Printf("audit log write failed: %v", err)
	}
}

func (a *authAPI) logout(w http.ResponseWriter, r *http.Request) {
	a.clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

// me returns the currently logged-in user (from the session cookie), for
// the portal (and any module) to check "am I logged in, and what can I see".
func (a *authAPI) me(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "not logged in")
		return
	}
	c, err := a.parseToken(cookie.Value)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "session expired")
		return
	}
	authUser, err := a.authUser(c.Subject)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, authUser)
}

// verify backs nginx's `auth_request` directive (see /nginx/auth-common.conf).
// It's called as an internal subrequest on every gated request across every
// module, so it only checks the token's signature/expiry and - if the
// caller set X-Module-Code - that the token's baked-in module list includes
// it, and that a password-change-required user is only let through to
// App Maintenance (where they can actually change it). No database access,
// so it stays cheap even at high request volume.
func (a *authAPI) verify(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "not logged in")
		return
	}
	c, err := a.parseToken(cookie.Value)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "session expired")
		return
	}

	moduleCode := r.Header.Get("X-Module-Code")

	if c.MustChangePassword && moduleCode != "" && moduleCode != "app-maintenance" {
		writeErr(w, http.StatusForbidden, "password change required")
		return
	}

	if moduleCode != "" {
		allowed := false
		for _, m := range c.ModuleCodes {
			if m == moduleCode {
				allowed = true
				break
			}
		}
		if !allowed {
			writeErr(w, http.StatusForbidden, "no access to this module")
			return
		}
	}

	w.Header().Set("X-User-Id", c.Subject)
	w.Header().Set("X-Username", c.Username)
	w.WriteHeader(http.StatusOK)
}

// changePassword is self-service (unlike the admin-only "Set Password"
// action in the Users tab): the caller must know their current password.
// Used both voluntarily and when MustChangePassword forces it.
func (a *authAPI) changePassword(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "not logged in")
		return
	}
	c, err := a.parseToken(cookie.Value)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "session expired")
		return
	}

	var in changePasswordRequest
	if err := decodeJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	if len(in.NewPassword) < 6 {
		writeErr(w, http.StatusBadRequest, "password must be at least 6 characters")
		return
	}

	user, passwordHash, err := a.store.GetUserByUsername(c.Username)
	if err != nil {
		handleErr(w, err)
		return
	}
	if !checkPassword(passwordHash, in.CurrentPassword) {
		writeErr(w, http.StatusUnauthorized, "current password is incorrect")
		return
	}

	hash, err := hashPassword(in.NewPassword)
	if err != nil {
		handleErr(w, err)
		return
	}
	if err := a.store.SetUserPassword(user.ID, hash); err != nil {
		handleErr(w, err)
		return
	}
	a.audit("auth.change_password", "user", user.ID, user.Username, clientIP(r))

	// Re-issue the session so MustChangePassword clears immediately,
	// instead of waiting for the old token to expire.
	modules, err := a.store.UserModules(user.ID)
	if err != nil {
		handleErr(w, err)
		return
	}
	codes := make([]string, len(modules))
	for i, m := range modules {
		codes[i] = m.Code
	}
	token, err := a.signToken(user.ID, user.Username, codes, false)
	if err != nil {
		handleErr(w, err)
		return
	}
	a.setSessionCookie(w, token)
	writeJSON(w, http.StatusOK, &AuthUser{ID: user.ID, Username: user.Username, FullName: user.FullName, Email: user.Email, Modules: modules})
}

// forgotPassword always responds with a generic message, whether or not
// the identifier matched a user - so this endpoint can't be used to probe
// which usernames/emails exist. It emails a one-time, short-lived reset
// link (see EmailSender - logged instead of sent until SMTP_* is set).
func (a *authAPI) forgotPassword(w http.ResponseWriter, r *http.Request) {
	var in forgotPasswordRequest
	if err := decodeJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}

	const genericMsg = "If that account exists, a password reset link has been sent to its email."

	user, _, err := a.store.GetUserByIdentifier(in.Identifier)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeJSON(w, http.StatusOK, map[string]string{"message": genericMsg})
			return
		}
		handleErr(w, err)
		return
	}

	token, tokenHash, err := newResetToken()
	if err != nil {
		handleErr(w, err)
		return
	}
	if err := a.store.CreatePasswordReset(user.ID, tokenHash, time.Now().Add(time.Hour)); err != nil {
		handleErr(w, err)
		return
	}

	link := fmt.Sprintf("%s/?reset_token=%s", a.publicBaseURL, token)
	body := fmt.Sprintf(
		"Hi %s,\n\nSomeone requested a password reset for your account. If this was you, set a new password here (valid for 1 hour):\n\n%s\n\nIf you didn't request this, you can ignore this email.",
		user.Username, link,
	)
	if err := a.emailSender.Send(user.Email, "Reset your password", body); err != nil {
		log.Printf("send reset email: %v", err)
	}
	a.audit("auth.forgot_password", "user", user.ID, user.Username, clientIP(r))

	writeJSON(w, http.StatusOK, map[string]string{"message": genericMsg})
}

// resetPassword consumes a one-time token (see ConsumePasswordReset) and
// sets a new password.
func (a *authAPI) resetPassword(w http.ResponseWriter, r *http.Request) {
	var in resetPasswordRequest
	if err := decodeJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	if len(in.NewPassword) < 6 {
		writeErr(w, http.StatusBadRequest, "password must be at least 6 characters")
		return
	}

	sum := sha256.Sum256([]byte(in.Token))
	tokenHash := hex.EncodeToString(sum[:])

	userID, err := a.store.ConsumePasswordReset(tokenHash)
	if errors.Is(err, ErrNotFound) {
		writeErr(w, http.StatusBadRequest, "this reset link is invalid or has expired")
		return
	}
	if err != nil {
		handleErr(w, err)
		return
	}

	hash, err := hashPassword(in.NewPassword)
	if err != nil {
		handleErr(w, err)
		return
	}
	if err := a.store.SetUserPassword(userID, hash); err != nil {
		handleErr(w, err)
		return
	}
	a.audit("auth.password_reset", "user", userID, "", clientIP(r))

	writeJSON(w, http.StatusOK, map[string]string{"message": "Password updated - you can now sign in."})
}
