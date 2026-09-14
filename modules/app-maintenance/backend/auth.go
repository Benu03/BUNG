package main

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const sessionCookieName = "bung_session"

// claims is what gets signed into the session JWT. ModuleCodes is baked in
// at login time (a join over roles -> role_modules -> modules), so nginx's
// auth_request check and every module's authorization decision is a pure
// in-memory check against the token - no database round trip on every
// request. The trade-off: a role/module change only takes effect for a
// user the next time they log in (or the token expires).
type claims struct {
	Username    string   `json:"username"`
	ModuleCodes []string `json:"moduleCodes"`
	jwt.RegisteredClaims
}

func hashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(b), err
}

func checkPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

type authAPI struct {
	store     *Store
	jwtSecret []byte
	tokenTTL  time.Duration
}

func newAuthAPI(store *Store, secret string, ttl time.Duration) *authAPI {
	return &authAPI{store: store, jwtSecret: []byte(secret), tokenTTL: ttl}
}

func (a *authAPI) signToken(userID, username string, moduleCodes []string) (string, error) {
	now := time.Now()
	c := claims{
		Username:    username,
		ModuleCodes: moduleCodes,
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

func (a *authAPI) authUser(userID string) (*AuthUser, error) {
	u, err := a.store.GetUser(userID)
	if err != nil {
		return nil, err
	}
	modules, err := a.store.UserModules(userID)
	if err != nil {
		return nil, err
	}
	return &AuthUser{ID: u.ID, Username: u.Username, FullName: u.FullName, Email: u.Email, Modules: modules}, nil
}

// login verifies credentials and, on success, sets the session cookie and
// returns the authenticated user + the modules they can access.
func (a *authAPI) login(w http.ResponseWriter, r *http.Request) {
	var in loginRequest
	if err := decodeJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}

	user, passwordHash, err := a.store.GetUserByUsername(in.Username)
	if errors.Is(err, sql.ErrNoRows) || errors.Is(err, ErrNotFound) {
		writeErr(w, http.StatusUnauthorized, "invalid username or password")
		return
	}
	if err != nil {
		handleErr(w, err)
		return
	}
	if !user.IsActive {
		writeErr(w, http.StatusUnauthorized, "account is disabled")
		return
	}
	if passwordHash == "" || !checkPassword(passwordHash, in.Password) {
		writeErr(w, http.StatusUnauthorized, "invalid username or password")
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

	token, err := a.signToken(user.ID, user.Username, codes)
	if err != nil {
		handleErr(w, err)
		return
	}
	a.setSessionCookie(w, token)
	writeJSON(w, http.StatusOK, &AuthUser{ID: user.ID, Username: user.Username, FullName: user.FullName, Email: user.Email, Modules: modules})
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
// it. No database access, so it stays cheap even at high request volume.
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

	if moduleCode := r.Header.Get("X-Module-Code"); moduleCode != "" {
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
