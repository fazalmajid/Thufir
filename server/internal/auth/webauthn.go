package auth

import (
	"bytes"
	"encoding/base64"
	"encoding/json"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"thufir/internal/config"
)

// NewWebAuthn creates the relying-party WebAuthn instance.
func NewWebAuthn(cfg config.Config) *webauthn.WebAuthn {
	wa, err := webauthn.New(&webauthn.Config{
		RPDisplayName: cfg.RPName,
		RPID:          cfg.RPID,
		RPOrigins:     []string{cfg.RPOrigin},
	})
	if err != nil {
		panic("webauthn init: " + err.Error())
	}
	return wa
}

// waUser implements webauthn.User.
type waUser struct {
	id          string
	displayName string
	creds       []webauthn.Credential
}

func (u *waUser) WebAuthnID() []byte                         { return []byte(u.id) }
func (u *waUser) WebAuthnName() string                       { return u.displayName }
func (u *waUser) WebAuthnDisplayName() string                { return u.displayName }
func (u *waUser) WebAuthnCredentials() []webauthn.Credential { return u.creds }
func (u *waUser) WebAuthnIcon() string                       { return "" }

// CredentialIDToBase64 encodes raw credential ID bytes to base64url (as stored in DB).
func CredentialIDToBase64(id []byte) string {
	return base64.RawURLEncoding.EncodeToString(id)
}

// Base64ToCredentialID decodes the stored base64url credential ID to raw bytes.
func Base64ToCredentialID(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}

// ParseCredentialCreationResponse extracts the WebAuthn registration response from
// the raw JSON value the client sent as the "response" field.
func ParseCredentialCreationResponse(raw json.RawMessage) (*protocol.ParsedCredentialCreationData, error) {
	return protocol.ParseCredentialCreationResponseBody(bytes.NewReader(raw))
}

// ParseCredentialRequestResponse extracts the WebAuthn assertion response from
// the raw JSON value the client sent as the "response" field.
func ParseCredentialRequestResponse(raw json.RawMessage) (*protocol.ParsedCredentialAssertionData, error) {
	return protocol.ParseCredentialRequestResponseBody(bytes.NewReader(raw))
}
