package auth

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/jackc/pgx/v5/pgxpool"

	"thufir/internal/config"
)

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
		SELECT credential_id, public_key, sign_count, transports
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
		if err := rows.Scan(&credIDBase64, &pubKey, &signCount, &transports); err != nil {
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
			webauthn.WithResidentKeyRequirement(protocol.ResidentKeyRequirementRequired),
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
			INSERT INTO credential (user_id, credential_id, public_key, sign_count, transports, device_name)
			VALUES ($1::uuid, $2, $3, $4, $5, $6)
		`, body.UserID, CredentialIDToBase64(cred.ID), cred.PublicKey,
			cred.Authenticator.SignCount, transports, body.DeviceName,
		); err != nil {
			writeErr(w, http.StatusInternalServerError, "save credential: "+err.Error())
			return
		}

		if err := tx.Commit(r.Context()); err != nil {
			writeErr(w, http.StatusInternalServerError, "commit")
			return
		}

		sessionID, err := CreateSession(r.Context(), pool, body.UserID)
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
		assertion, session, err := wa.BeginDiscoverableLogin()
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
			user, err := loadUserWithCredentials(r, pool, uid)
			if err != nil {
				return nil, err
			}
			foundUserID = uid
			return user, nil
		}

		cred, err := wa.ValidateDiscoverableLogin(handler, *session, parsed)
		if err != nil {
			writeErr(w, http.StatusUnauthorized, "authentication failed")
			return
		}

		// Update sign count to prevent replay attacks
		_, _ = pool.Exec(r.Context(), `
			UPDATE credential SET sign_count = $1 WHERE credential_id = $2
		`, cred.Authenticator.SignCount, CredentialIDToBase64(cred.ID))

		sessionID, err := CreateSession(r.Context(), pool, foundUserID)
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

		excludeList := make([]protocol.CredentialDescriptor, len(user.creds))
		for i, c := range user.creds {
			excludeList[i] = protocol.CredentialDescriptor{
				Type:         protocol.PublicKeyCredentialType,
				CredentialID: c.ID,
				Transport:    c.Transport,
			}
		}

		creation, session, err := wa.BeginRegistration(user,
			webauthn.WithResidentKeyRequirement(protocol.ResidentKeyRequirementRequired),
			webauthn.WithExclusions(excludeList),
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
			INSERT INTO credential (user_id, credential_id, public_key, sign_count, transports, device_name)
			VALUES ($1::uuid, $2, $3, $4, $5, $6)
		`, info.UserID, CredentialIDToBase64(cred.ID), cred.PublicKey,
			cred.Authenticator.SignCount, transports, body.DeviceName,
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


