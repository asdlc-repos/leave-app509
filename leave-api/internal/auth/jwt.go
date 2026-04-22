package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// Minimal HS256 JWT implementation.

var ErrInvalidToken = errors.New("invalid token")

type Claims struct {
	Sub  string `json:"sub"`
	Role string `json:"role"`
	Name string `json:"name"`
	Iat  int64  `json:"iat"`
	Exp  int64  `json:"exp"`
}

type Signer struct {
	secret []byte
	ttl    time.Duration
}

func NewSigner(secret string, ttl time.Duration) *Signer {
	return &Signer{secret: []byte(secret), ttl: ttl}
}

func (s *Signer) Sign(c Claims) string {
	headerJSON := `{"alg":"HS256","typ":"JWT"}`
	header := base64URLEncode([]byte(headerJSON))
	body, _ := json.Marshal(c)
	payload := base64URLEncode(body)
	unsigned := header + "." + payload
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(unsigned))
	sig := base64URLEncode(mac.Sum(nil))
	return unsigned + "." + sig
}

func (s *Signer) Verify(token string) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}
	unsigned := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(unsigned))
	expected := base64URLEncode(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(parts[2])) {
		return nil, ErrInvalidToken
	}
	payload, err := base64URLDecode(parts[1])
	if err != nil {
		return nil, ErrInvalidToken
	}
	var c Claims
	if err := json.Unmarshal(payload, &c); err != nil {
		return nil, ErrInvalidToken
	}
	if time.Now().Unix() >= c.Exp {
		return nil, errors.New("token expired")
	}
	return &c, nil
}

// TTL returns the configured token lifetime (used for sliding renewal).
func (s *Signer) TTL() time.Duration { return s.ttl }

// Issue produces a fresh token for the given subject with sliding expiry.
func (s *Signer) Issue(userID, role, name string) string {
	now := time.Now()
	return s.Sign(Claims{
		Sub:  userID,
		Role: role,
		Name: name,
		Iat:  now.Unix(),
		Exp:  now.Add(s.ttl).Unix(),
	})
}

func base64URLEncode(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

func base64URLDecode(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}
