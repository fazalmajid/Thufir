package auth

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/protocol/webauthncose"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/jackc/pgx/v5/pgxpool"

	"thufir/internal/config"
)

// androidCompatParams restricts pubKeyCredParams to ES256 + RS256 only.
// Android's Credential Manager has compatibility issues with some of the
// less common algorithms (ES384, EdDSA, etc.) that go-webauthn includes by default.
var androidCompatParams = webauthn.WithCredentialParameters([]protocol.CredentialParameter{
	{Type: protocol.PublicKeyCredentialType, Algorithm: webauthncose.AlgES256},
	{Type: protocol.PublicKeyCredentialType, Algorithm: webauthncose.AlgRS256},
})

// clientIP extracts the real client IP, respecting X-Forwarded-For from proxies.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first (client) IP in the chain.
		if idx := len(xff); idx > 0 {
			for i := 0; i < len(xff); i++ {
				if xff[i] == ',' {
					return xff[:i]
				}
			}
		}
		return xff
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// Strip port from RemoteAddr.
	addr := r.RemoteAddr
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			return addr[:i]
		}
	}
	return addr
}

// ── helpers ────────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func setSessionCookie(w http.ResponseWriter, sessionID string, isProd bool) {
	sameSite := http.SameSiteLaxMode
	if isProd {
		// SameSite=None is required for cross-origin requests (e.g. the bookmarklet)
		// to include the cookie. Requires Secure=true (HTTPS).
		sameSite = http.SameSiteNoneMode
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    sessionID,
		Path:     "/",
		MaxAge:   90 * 24 * 60 * 60,
		HttpOnly: true,
		Secure:   isProd,
		SameSite: sameSite,
	})
}

func setChallengeCookie(w http.ResponseWriter, token string, isProd bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     "challenge",
		Value:    token,
		Path:     "/",
		MaxAge:   300,
		HttpOnly: true,
		Secure:   isProd,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:    name,
		Value:   "",
		Path:    "/",
		MaxAge:  -1,
		Expires: time.Unix(0, 0),
	})
}

func challengeToken(r *http.Request) string {
	c, err := r.Cookie("challenge")
	if err != nil {
		return ""
	}
	return c.Value
}

// loadUserWithCredentials fetches the user and all their stored passkeys.
func loadUserWithCredentials(r *http.Request, pool *pgxpool.Pool, userID string) (*waUser, error) {
	var displayName string
	err := pool.QueryRow(r.Context(), `
		SELECT display_name FROM name WHERE id = $1::uuid
`, userID).Scan(&displayName)
	if err != nil {
		return nil, err
	}

	rows, err := pool.Query(r.Context(), `
		SELECT credential_id, public_key, sign_count, transports, backup_eligible, backup_state
		FROM credential WHERE user_id = $1::uuid
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var creds []webauthn.Credential
	for rows.Next() {
		var credIDBase64 string
		var pubKey []byte
		var signCount int64
		var transports []string
		var backupEligible, backupState bool
		if err := rows.Scan(&credIDBase64, &pubKey, &signCount, &transports, &backupEligible, &backupState); err != nil {
			return nil, err
		}
		credID, err := Base64ToCredentialID(credIDBase64)
		if err != nil {
			return nil, err
		}
		var t []protocol.AuthenticatorTransport
		for _, s := range transports {
			t = append(t, protocol.AuthenticatorTransport(s))
		}
		creds = append(creds, webauthn.Credential{
			ID:        credID,
			PublicKey: pubKey,
			Transport: t,
			Flags: webauthn.CredentialFlags{
				BackupEligible: backupEligible,
				BackupState:    backupState,
			},
			Authenticator: webauthn.Authenticator{
				SignCount: uint32(signCount),
			},
		})
	}
	return &waUser{id: userID, displayName: displayName, creds: creds}, nil
}

// ── handler constructors ──────────────────────────────────────────────────────

func HandleStatus(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var count int
		if err := pool.QueryRow(r.Context(), `SELECT COUNT(*)::int FROM name`).Scan(&count); err != nil {
			writeErr(w, http.StatusInternalServerError, "db error")
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"hasUsers": count > 0})
	}
}

func HandleMe(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"user": nil})
			return
		}
		info, err := ValidateSession(r.Context(), pool, cookie.Value)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"user": nil})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"user": map[string]string{"id": info.UserID, "displayName": info.DisplayName},
		})
	}
}

func HandleLogout(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if cookie, err := r.Cookie("session"); err == nil {
			_ = DeleteSession(r.Context(), pool, cookie.Value)
		}
		clearCookie(w, "session")
		writeJSON(w, http.StatusOK, map[string]bool{"success": true})
	}
}

// ── first-time setup ──────────────────────────────────────────────────────────

func HandleSetupOptions(pool *pgxpool.Pool, wa *webauthn.WebAuthn, cs *ChallengeStore, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			DisplayName string `json:"displayName"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.DisplayName == "" {
			writeErr(w, http.StatusBadRequest, "displayName required")
			return
		}

		var count int
		pool.QueryRow(r.Context(), `SELECT COUNT(*)::int FROM name`).Scan(&count) //nolint:errcheck
		if count > 0 {
			writeErr(w, http.StatusForbidden, "Setup already complete")
			return
		}

		userUUID := NewUUID()

		user := &waUser{id: userUUID, displayName: body.DisplayName}
		creation, session, err := wa.BeginRegistration(user,
			webauthn.WithAuthenticatorSelection(protocol.AuthenticatorSelection{
				RequireResidentKey: protocol.ResidentKeyNotRequired(),
				ResidentKey:        protocol.ResidentKeyRequirementPreferred,
				UserVerification:   protocol.VerificationPreferred,
			}),
			webauthn.WithConveyancePreference(protocol.PreferNoAttestation),
			androidCompatParams,
		)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "begin registration: "+err.Error())
			return
		}

		token := NewToken()
		if err := cs.Set(token, session, userUUID); err != nil {
			writeErr(w, http.StatusInternalServerError, "store challenge")
			return
		}
		setChallengeCookie(w, token, cfg.IsProd)
		writeJSON(w, http.StatusOK, map[string]any{"options": creation.Response, "userId": userUUID})
	}
}

func HandleSetupVerify(pool *pgxpool.Pool, wa *webauthn.WebAuthn, cs *ChallengeStore, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var count int
		pool.QueryRow(r.Context(), `SELECT COUNT(*)::int FROM name`).Scan(&count) //nolint:errcheck
		if count > 0 {
			writeErr(w, http.StatusForbidden, "Setup already complete")
			return
		}

		var body struct {
			UserID      string          `json:"userId"`
			DisplayName string          `json:"displayName"`
			DeviceName  *string         `json:"deviceName"`
			Response    json.RawMessage `json:"response"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid body")
			return
		}

		token := challengeToken(r)
		session, userID, ok := cs.Get(token)
		if !ok || userID != body.UserID {
			writeErr(w, http.StatusBadRequest, "challenge expired or invalid")
			return
		}
		clearCookie(w, "challenge")

		parsed, err := ParseCredentialCreationResponse(body.Response)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "parse response: "+err.Error())
			return
		}
		user := &waUser{id: body.UserID, displayName: body.DisplayName}
		cred, err := wa.CreateCredential(user, *session, parsed)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "verify registration: "+err.Error())
			return
		}

		// Persist user + credential in a transaction
		tx, err := pool.Begin(r.Context())
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "db error")
			return
		}
		defer tx.Rollback(r.Context()) //nolint:errcheck

		if _, err := tx.Exec(r.Context(),
			`INSERT INTO name (id, display_name) VALUES ($1::uuid, $2)`,
			body.UserID, body.DisplayName,
		); err != nil {
			writeErr(w, http.StatusInternalServerError, "create user: "+err.Error())
			return
		}

		transports := make([]string, len(cred.Transport))
		for i, t := range cred.Transport {
			transports[i] = string(t)
		}
		if _, err := tx.Exec(r.Context(), `
			INSERT INTO credential (user_id, credential_id, public_key, sign_count, transports, device_name, backup_eligible, backup_state)
			VALUES ($1::uuid, $2, $3, $4, $5, $6, $7, $8)
		`, body.UserID, CredentialIDToBase64(cred.ID), cred.PublicKey,
			cred.Authenticator.SignCount, transports, body.DeviceName,
			cred.Flags.BackupEligible, cred.Flags.BackupState,
		); err != nil {
			writeErr(w, http.StatusInternalServerError, "save credential: "+err.Error())
			return
		}

		if err := tx.Commit(r.Context()); err != nil {
			writeErr(w, http.StatusInternalServerError, "commit")
			return
		}

		sessionID, err := CreateSession(r.Context(), pool, body.UserID, r.UserAgent(), clientIP(r))
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "create session")
			return
		}
		setSessionCookie(w, sessionID, cfg.IsProd)
		writeJSON(w, http.StatusOK, map[string]bool{"success": true})
	}
}

// ── login ─────────────────────────────────────────────────────────────────────

func HandleLoginOptions(wa *webauthn.WebAuthn, cs *ChallengeStore, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		assertion, session, err := wa.BeginDiscoverableLogin(
			webauthn.WithUserVerification(protocol.VerificationPreferred),
		)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "begin login: "+err.Error())
			return
		}
		token := NewToken()
		if err := cs.Set(token, session, ""); err != nil {
			writeErr(w, http.StatusInternalServerError, "store challenge")
			return
		}
		setChallengeCookie(w, token, cfg.IsProd)
		writeJSON(w, http.StatusOK, assertion.Response)
	}
}

func HandleLoginVerify(pool *pgxpool.Pool, wa *webauthn.WebAuthn, cs *ChallengeStore, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Response json.RawMessage `json:"response"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid body")
			return
		}

		token := challengeToken(r)
		session, _, ok := cs.Get(token)
		if !ok {
			writeErr(w, http.StatusBadRequest, "challenge expired")
			return
		}
		clearCookie(w, "challenge")

		parsed, err := ParseCredentialRequestResponse(body.Response)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "parse response: "+err.Error())
			return
		}

		var foundUserID string
		handler := func(rawID, userHandle []byte) (webauthn.User, error) {
			uid := string(userHandle)

			// Apple Passkeys may omit the user handle; fall back to credential ID lookup.
			if uid == "" {
				credIDBase64 := CredentialIDToBase64(rawID)
				if err := pool.QueryRow(r.Context(),
					`SELECT user_id::text FROM credential WHERE credential_id = $1`,
					credIDBase64,
				).Scan(&uid); err != nil {
					return nil, fmt.Errorf("credential not found for rawID: %w", err)
				}
			}

			user, err := loadUserWithCredentials(r, pool, uid)
			if err != nil {
				return nil, err
			}
			foundUserID = uid
			return user, nil
		}

		cred, err := wa.ValidateDiscoverableLogin(handler, *session, parsed)
		if err != nil {
			log.Printf("ValidateDiscoverableLogin error: %v", err)
			writeErr(w, http.StatusUnauthorized, "authentication failed")
			return
		}

		// Update mutable fields after successful authentication.
		_, _ = pool.Exec(r.Context(), `
			UPDATE credential SET sign_count = $1, backup_state = $2 WHERE credential_id = $3
		`, cred.Authenticator.SignCount, cred.Flags.BackupState, CredentialIDToBase64(cred.ID))

		sessionID, err := CreateSession(r.Context(), pool, foundUserID, r.UserAgent(), clientIP(r))
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "create session")
			return
		}
		setSessionCookie(w, sessionID, cfg.IsProd)

		var displayName string
		pool.QueryRow(r.Context(), `SELECT display_name FROM name WHERE id = $1::uuid`, foundUserID).Scan(&displayName) //nolint:errcheck

		writeJSON(w, http.StatusOK, map[string]any{
			"success": true,
			"user":    map[string]string{"id": foundUserID, "displayName": displayName},
		})
	}
}

// ── add device ────────────────────────────────────────────────────────────────

func HandleDeviceOptions(pool *pgxpool.Pool, wa *webauthn.WebAuthn, cs *ChallengeStore, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err != nil {
			writeErr(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		info, err := ValidateSession(r.Context(), pool, cookie.Value)
		if err != nil {
			writeErr(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		user, err := loadUserWithCredentials(r, pool, info.UserID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "load user")
			return
		}

		// Exclude already-enrolled credentials (no transports — Android Credential Manager
		// can mishandle transport hints from credentials enrolled on other platforms).
		excludeList := make([]protocol.CredentialDescriptor, len(user.creds))
		for i, c := range user.creds {
			excludeList[i] = protocol.CredentialDescriptor{
				Type:         protocol.PublicKeyCredentialType,
				CredentialID: c.ID,
			}
		}

		creation, session, err := wa.BeginRegistration(user,
			webauthn.WithAuthenticatorSelection(protocol.AuthenticatorSelection{
				RequireResidentKey: protocol.ResidentKeyNotRequired(),
				ResidentKey:        protocol.ResidentKeyRequirementPreferred,
				UserVerification:   protocol.VerificationPreferred,
			}),
			webauthn.WithConveyancePreference(protocol.PreferNoAttestation),
			webauthn.WithExclusions(excludeList),
			androidCompatParams,
		)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "begin registration: "+err.Error())
			return
		}

		token := NewToken()
		if err := cs.Set(token, session, info.UserID); err != nil {
			writeErr(w, http.StatusInternalServerError, "store challenge")
			return
		}
		setChallengeCookie(w, token, cfg.IsProd)
		writeJSON(w, http.StatusOK, creation.Response)
	}
}

func HandleDeviceVerify(pool *pgxpool.Pool, wa *webauthn.WebAuthn, cs *ChallengeStore, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err != nil {
			writeErr(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		info, err := ValidateSession(r.Context(), pool, cookie.Value)
		if err != nil {
			writeErr(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		var body struct {
			DeviceName *string         `json:"deviceName"`
			Response   json.RawMessage `json:"response"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid body")
			return
		}

		token := challengeToken(r)
		session, tokenUserID, ok := cs.Get(token)
		if !ok || tokenUserID != info.UserID {
			writeErr(w, http.StatusBadRequest, "challenge expired or invalid")
			return
		}
		clearCookie(w, "challenge")

		user, err := loadUserWithCredentials(r, pool, info.UserID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "load user")
			return
		}

		parsed, err := ParseCredentialCreationResponse(body.Response)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "parse response: "+err.Error())
			return
		}
		cred, err := wa.CreateCredential(user, *session, parsed)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "verify registration: "+err.Error())
			return
		}

		transports := make([]string, len(cred.Transport))
		for i, t := range cred.Transport {
			transports[i] = string(t)
		}
		if _, err := pool.Exec(r.Context(), `
			INSERT INTO credential (user_id, credential_id, public_key, sign_count, transports, device_name, backup_eligible, backup_state)
			VALUES ($1::uuid, $2, $3, $4, $5, $6, $7, $8)
		`, info.UserID, CredentialIDToBase64(cred.ID), cred.PublicKey,
			cred.Authenticator.SignCount, transports, body.DeviceName,
			cred.Flags.BackupEligible, cred.Flags.BackupState,
		); err != nil {
			writeErr(w, http.StatusInternalServerError, "save credential: "+err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"success": true})
	}
}

func HandleListDevices(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err != nil {
			writeErr(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		info, err := ValidateSession(r.Context(), pool, cookie.Value)
		if err != nil {
			writeErr(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		rows, err := pool.Query(r.Context(), `
			SELECT id::text, device_name, transports, created_at
			FROM credential WHERE user_id = $1::uuid ORDER BY created_at ASC
		`, info.UserID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "db error")
			return
		}
		defer rows.Close()

		type device struct {
			ID         string   `json:"id"`
			DeviceName *string  `json:"device_name"`
			Transports []string `json:"transports"`
			CreatedAt  string   `json:"created_at"`
		}
		var devices []device
		for rows.Next() {
			var d device
			var createdAt time.Time
			if err := rows.Scan(&d.ID, &d.DeviceName, &d.Transports, &createdAt); err != nil {
				writeErr(w, http.StatusInternalServerError, "scan")
				return
			}
			d.CreatedAt = createdAt.UTC().Format(time.RFC3339)
			devices = append(devices, d)
		}
		if devices == nil {
			devices = []device{}
		}
		writeJSON(w, http.StatusOK, devices)
	}
}

func HandleListSessions(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err != nil {
			writeErr(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		info, err := ValidateSession(r.Context(), pool, cookie.Value)
		if err != nil {
			writeErr(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		rows, err := pool.Query(r.Context(), `
			SELECT id::text, ua_display, ip_address, created_at, expires_at
			FROM session
			WHERE user_id = $1::uuid AND expires_at > NOW()
			ORDER BY created_at DESC
		`, info.UserID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "db error")
			return
		}
		defer rows.Close()

		type sessionInfo struct {
			ID         string  `json:"id"`
			UADisplay  *string `json:"ua_display"`
			IPAddress  *string `json:"ip_address"`
			CreatedAt  string  `json:"created_at"`
			ExpiresAt  string  `json:"expires_at"`
			IsCurrent  bool    `json:"is_current"`
		}
		var sessions []sessionInfo
		for rows.Next() {
			var s sessionInfo
			var createdAt, expiresAt time.Time
			if err := rows.Scan(&s.ID, &s.UADisplay, &s.IPAddress, &createdAt, &expiresAt); err != nil {
				writeErr(w, http.StatusInternalServerError, "scan")
				return
			}
			s.CreatedAt = createdAt.UTC().Format(time.RFC3339)
			s.ExpiresAt = expiresAt.UTC().Format(time.RFC3339)
			s.IsCurrent = s.ID == cookie.Value
			sessions = append(sessions, s)
		}
		if sessions == nil {
			sessions = []sessionInfo{}
		}
		writeJSON(w, http.StatusOK, sessions)
	}
}

func HandleDeleteSession(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err != nil {
			writeErr(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		info, err := ValidateSession(r.Context(), pool, cookie.Value)
		if err != nil {
			writeErr(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		sessionID := chi.URLParam(r, "id")

		// Verify the session belongs to this user before deleting.
		var ownerID string
		if err := pool.QueryRow(r.Context(),
			`SELECT user_id::text FROM session WHERE id = $1::uuid`,
			sessionID,
		).Scan(&ownerID); err != nil || ownerID != info.UserID {
			writeErr(w, http.StatusNotFound, "session not found")
			return
		}

		if err := DeleteSession(r.Context(), pool, sessionID); err != nil {
			writeErr(w, http.StatusInternalServerError, "db error")
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"success": true})
	}
}

func HandleDeleteDevice(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err != nil {
			writeErr(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		info, err := ValidateSession(r.Context(), pool, cookie.Value)
		if err != nil {
			writeErr(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		credID := chi.URLParam(r, "id")

		var count int
		if err := pool.QueryRow(r.Context(),
			`SELECT COUNT(*)::int FROM credential WHERE user_id = $1::uuid`,
			info.UserID,
		).Scan(&count); err != nil || count <= 1 {
			writeErr(w, http.StatusBadRequest, "Cannot remove the last passkey")
			return
		}

		if _, err := pool.Exec(r.Context(),
			`DELETE FROM credential WHERE id = $1::uuid AND user_id = $2::uuid`,
			credID, info.UserID,
		); err != nil {
			writeErr(w, http.StatusInternalServerError, "db error")
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"success": true})
	}
}


