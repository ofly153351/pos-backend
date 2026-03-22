package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

func hashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := deriveKey(password, salt)
	return hex.EncodeToString(salt) + ":" + hex.EncodeToString(hash), nil
}

func verifyPassword(password, encoded string) bool {
	parts := strings.Split(encoded, ":")
	if len(parts) != 2 {
		return false
	}

	salt, err := hex.DecodeString(parts[0])
	if err != nil {
		return false
	}

	expectedHash, err := hex.DecodeString(parts[1])
	if err != nil {
		return false
	}

	actualHash := deriveKey(password, salt)
	return subtle.ConstantTimeCompare(expectedHash, actualHash) == 1
}

func deriveKey(password string, salt []byte) []byte {
	seed := append([]byte(password), salt...)
	sum := sha256.Sum256(seed)
	key := sum[:]

	for range 120000 {
		next := sha256.Sum256(append(key, salt...))
		key = next[:]
	}

	out := make([]byte, len(key))
	copy(out, key)
	return out
}

type TokenManager struct {
	secret []byte
	ttl    time.Duration
}

type Claims struct {
	UserID    string
	Email     string
	Name      string
	Role      string
	ExpiresAt int64
}

func NewTokenManager(secret string, ttl time.Duration) TokenManager {
	return TokenManager{
		secret: []byte(secret),
		ttl:    ttl,
	}
}

func (m TokenManager) Issue(user User) (string, error) {
	headerPart := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))

	payload := map[string]any{
		"sub":   user.ID,
		"email": user.Email,
		"name":  user.Name,
		"role":  user.Role,
		"exp":   time.Now().Add(m.ttl).Unix(),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	payloadPart := base64.RawURLEncoding.EncodeToString(payloadBytes)
	unsigned := headerPart + "." + payloadPart

	mac := hmac.New(sha256.New, m.secret)
	if _, err := mac.Write([]byte(unsigned)); err != nil {
		return "", err
	}

	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return unsigned + "." + signature, nil
}

func (m TokenManager) Parse(token string) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, errors.New("invalid token format")
	}

	unsigned := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, m.secret)
	if _, err := mac.Write([]byte(unsigned)); err != nil {
		return Claims{}, err
	}

	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if subtle.ConstantTimeCompare([]byte(expected), []byte(parts[2])) != 1 {
		return Claims{}, errors.New("invalid token signature")
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Claims{}, err
	}

	claims := make(map[string]any)
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return Claims{}, err
	}

	exp, ok := claims["exp"].(float64)
	if !ok || int64(exp) < time.Now().Unix() {
		return Claims{}, fmt.Errorf("token expired")
	}

	return Claims{
		UserID:    asString(claims["sub"]),
		Email:     asString(claims["email"]),
		Name:      asString(claims["name"]),
		Role:      asString(claims["role"]),
		ExpiresAt: int64(exp),
	}, nil
}

func asString(value any) string {
	text, _ := value.(string)
	return text
}
