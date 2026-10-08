package poker

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"strings"
)

const cookieName = "poker_identity"

func (a *App) identify(r *http.Request) (string, string, error) {
	if cookie, err := r.Cookie(cookieName); err == nil {
		parts := strings.Split(cookie.Value, ".")
		if len(parts) == 2 {
			token, e1 := base64.RawURLEncoding.DecodeString(parts[0])
			signature, e2 := base64.RawURLEncoding.DecodeString(parts[1])
			mac := hmac.New(sha256.New, a.secret)
			mac.Write(token)
			if e1 == nil && e2 == nil && len(token) == 32 && hmac.Equal(signature, mac.Sum(nil)) {
				return playerID(token), "", nil
			}
		}
	}
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		return "", "", err
	}
	mac := hmac.New(sha256.New, a.secret)
	mac.Write(token)
	credential := base64.RawURLEncoding.EncodeToString(token) + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return playerID(token), credential, nil
}

func playerID(token []byte) string {
	id := sha256.Sum256(token)
	return "p_" + hex.EncodeToString(id[:16])
}

func setIdentity(w http.ResponseWriter, r *http.Request, credential string) {
	http.SetCookie(w, &http.Cookie{
		Name: cookieName, Value: credential, Path: "/", MaxAge: 365 * 24 * 60 * 60,
		HttpOnly: true, Secure: r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https",
		SameSite: http.SameSiteLaxMode,
	})
}
