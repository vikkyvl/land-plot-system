package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"math/big"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	testIssuer   = "http://localhost:8180/realms/land-plot"
	testAudience = "land-plot-api"
	testSubject  = "164d085d-f431-498a-819a-e5a1ec00e1c1"
	testKeyID    = "key-1"
	rsaKeyBits   = 2048
)

var testNow = time.Date(2026, time.November, 17, 12, 0, 0, 0, time.UTC)

type publishedKey struct {
	id  string
	use string
	key *rsa.PublicKey
}

type fakeJWKS struct {
	mutex    sync.Mutex
	keys     []publishedKey
	requests atomic.Int32
}

func (server *fakeJWKS) publish(keys ...publishedKey) {
	server.mutex.Lock()
	defer server.mutex.Unlock()

	server.keys = keys
}

func (server *fakeJWKS) ServeHTTP(response http.ResponseWriter, _ *http.Request) {
	server.requests.Add(1)
	server.mutex.Lock()
	defer server.mutex.Unlock()

	keys := make([]map[string]string, 0, len(server.keys))
	for _, published := range server.keys {
		keys = append(keys, map[string]string{
			"kty": "RSA",
			"kid": published.id,
			"use": published.use,
			"alg": "RS256",
			"n":   base64.RawURLEncoding.EncodeToString(published.key.N.Bytes()),
			"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(published.key.E)).Bytes()),
		})
	}

	response.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(response).Encode(map[string]any{"keys": keys})
}

func newRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, rsaKeyBits)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}

	return key
}

func startVerifier(t *testing.T, keys ...publishedKey) (*OIDCVerifier, *fakeJWKS) {
	t.Helper()

	jwks := &fakeJWKS{}
	jwks.publish(keys...)

	server := httptest.NewServer(jwks)
	t.Cleanup(server.Close)

	verifier, err := NewOIDCVerifier(t.Context(), OIDCSettings{
		JWKSURL:  server.URL,
		Issuer:   testIssuer,
		Audience: testAudience,
		Timeout:  time.Second,
	}, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("NewOIDCVerifier() error = %v", err)
	}

	verifier.now = func() time.Time { return testNow }

	return verifier, jwks
}

func validClaims() jwt.MapClaims {
	return jwt.MapClaims{
		"iss":   testIssuer,
		"aud":   testAudience,
		"sub":   testSubject,
		"exp":   testNow.Add(15 * time.Minute).Unix(),
		"iat":   testNow.Unix(),
		"email": "taras.melnyk@example.com",
		"name":  "Тарас Мельник",
		"scope": "openid email plots:read profile",
	}
}

func sign(t *testing.T, key *rsa.PrivateKey, keyID string, claims jwt.MapClaims) string {
	t.Helper()

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = keyID

	signed, err := token.SignedString(key)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	return signed
}

func TestVerifierReturnsThePrincipalOfAValidToken(t *testing.T) {
	key := newRSAKey(t)
	verifier, _ := startVerifier(t, publishedKey{id: testKeyID, use: "sig", key: &key.PublicKey})

	principal, err := verifier.Verify(sign(t, key, testKeyID, validClaims()))
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}

	if principal.Subject != testSubject || principal.Email != "taras.melnyk@example.com" ||
		principal.Name != "Тарас Мельник" ||
		!slices.Equal(principal.Scopes, []string{"openid", "email", "plots:read", "profile"}) {
		t.Fatalf("Verify() = %#v", principal)
	}

	if !principal.HasScope("plots:read") || principal.HasScope("plots:write") {
		t.Fatalf("HasScope() is wrong for %v", principal.Scopes)
	}
}

func TestVerifierAcceptsAServiceAccountWithoutEmail(t *testing.T) {
	key := newRSAKey(t)
	verifier, _ := startVerifier(t, publishedKey{id: testKeyID, use: "sig", key: &key.PublicKey})

	claims := validClaims()
	delete(claims, "email")
	delete(claims, "name")
	claims["scope"] = "plots:read plots:write"

	principal, err := verifier.Verify(sign(t, key, testKeyID, claims))
	if err != nil || principal.Email != "" || len(principal.Scopes) != 2 {
		t.Fatalf("Verify() = %#v, %v", principal, err)
	}
}

func TestVerifierToleratesClockSkewOfThirtySeconds(t *testing.T) {
	key := newRSAKey(t)
	verifier, _ := startVerifier(t, publishedKey{id: testKeyID, use: "sig", key: &key.PublicKey})

	claims := validClaims()
	claims["exp"] = testNow.Add(-20 * time.Second).Unix()
	claims["nbf"] = testNow.Add(20 * time.Second).Unix()

	if _, err := verifier.Verify(sign(t, key, testKeyID, claims)); err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
}

func TestVerifierRejectsAnExpiredToken(t *testing.T) {
	key := newRSAKey(t)
	verifier, _ := startVerifier(t, publishedKey{id: testKeyID, use: "sig", key: &key.PublicKey})

	claims := validClaims()
	claims["exp"] = testNow.Add(-time.Minute).Unix()

	if _, err := verifier.Verify(sign(t, key, testKeyID, claims)); !errors.Is(err, ErrExpiredToken) {
		t.Fatalf("Verify() error = %v, want ErrExpiredToken", err)
	}
}

func TestVerifierRejectsInvalidClaims(t *testing.T) {
	key := newRSAKey(t)
	verifier, _ := startVerifier(t, publishedKey{id: testKeyID, use: "sig", key: &key.PublicKey})

	tests := map[string]func(jwt.MapClaims){
		"foreign issuer":        func(claims jwt.MapClaims) { claims["iss"] = "http://evil.example/realms/land-plot" },
		"missing audience":      func(claims jwt.MapClaims) { delete(claims, "aud") },
		"foreign audience":      func(claims jwt.MapClaims) { claims["aud"] = "account" },
		"missing expiration":    func(claims jwt.MapClaims) { delete(claims, "exp") },
		"missing subject":       func(claims jwt.MapClaims) { delete(claims, "sub") },
		"not valid before long": func(claims jwt.MapClaims) { claims["nbf"] = testNow.Add(time.Minute).Unix() },
	}
	for name, change := range tests {
		t.Run(name, func(t *testing.T) {
			claims := validClaims()
			change(claims)

			if _, err := verifier.Verify(sign(t, key, testKeyID, claims)); !errors.Is(err, ErrInvalidToken) {
				t.Fatalf("Verify() error = %v, want ErrInvalidToken", err)
			}
		})
	}
}

func TestVerifierAcceptsAnAudienceList(t *testing.T) {
	key := newRSAKey(t)
	verifier, _ := startVerifier(t, publishedKey{id: testKeyID, use: "sig", key: &key.PublicKey})

	claims := validClaims()
	claims["aud"] = []string{"account", testAudience}

	if _, err := verifier.Verify(sign(t, key, testKeyID, claims)); err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
}

func TestVerifierRejectsTokensNotSignedWithRS256(t *testing.T) {
	key := newRSAKey(t)
	verifier, _ := startVerifier(t, publishedKey{id: testKeyID, use: "sig", key: &key.PublicKey})

	unsigned := jwt.NewWithClaims(jwt.SigningMethodNone, validClaims())
	unsigned.Header["kid"] = testKeyID

	noneToken, err := unsigned.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("sign alg none: %v", err)
	}

	symmetric := jwt.NewWithClaims(jwt.SigningMethodHS256, validClaims())
	symmetric.Header["kid"] = testKeyID

	hmacToken, err := symmetric.SignedString([]byte("local_development_jwt_secret_change_me"))
	if err != nil {
		t.Fatalf("sign HS256: %v", err)
	}

	for name, token := range map[string]string{"alg none": noneToken, "HS256": hmacToken} {
		if _, err := verifier.Verify(token); !errors.Is(err, ErrInvalidToken) {
			t.Fatalf("%s: Verify() error = %v, want ErrInvalidToken", name, err)
		}
	}
}

func TestVerifierRejectsATokenSignedByAnotherKey(t *testing.T) {
	key := newRSAKey(t)
	verifier, _ := startVerifier(t, publishedKey{id: testKeyID, use: "sig", key: &key.PublicKey})

	if _, err := verifier.Verify(sign(t, newRSAKey(t), testKeyID, validClaims())); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("Verify() error = %v, want ErrInvalidToken", err)
	}
}

func TestVerifierIgnoresEncryptionKeys(t *testing.T) {
	key := newRSAKey(t)
	verifier, _ := startVerifier(t, publishedKey{id: testKeyID, use: "enc", key: &key.PublicKey})

	if _, err := verifier.Verify(sign(t, key, testKeyID, validClaims())); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("Verify() error = %v, want ErrInvalidToken", err)
	}
}

func TestVerifierRefreshesKeysForAnUnknownKeyID(t *testing.T) {
	oldKey, newKey := newRSAKey(t), newRSAKey(t)
	verifier, jwks := startVerifier(t, publishedKey{id: "old", use: "sig", key: &oldKey.PublicKey})

	if _, err := verifier.Verify(sign(t, oldKey, "old", validClaims())); err != nil {
		t.Fatalf("Verify() with the old key error = %v", err)
	}

	jwks.publish(
		publishedKey{id: "old", use: "sig", key: &oldKey.PublicKey},
		publishedKey{id: "new", use: "sig", key: &newKey.PublicKey},
	)
	requestsBefore := jwks.requests.Load()

	if _, err := verifier.Verify(sign(t, newKey, "new", validClaims())); err != nil {
		t.Fatalf("Verify() with a rotated key error = %v", err)
	}

	if jwks.requests.Load() != requestsBefore+1 {
		t.Fatalf("JWKS requests = %d, want one refresh after %d", jwks.requests.Load(), requestsBefore)
	}
}

func TestVerifierLimitsRefreshesForUnknownKeyIDs(t *testing.T) {
	key := newRSAKey(t)
	verifier, jwks := startVerifier(t, publishedKey{id: testKeyID, use: "sig", key: &key.PublicKey})

	if _, err := verifier.Verify(sign(t, key, "unknown-1", validClaims())); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("first unknown key: Verify() error = %v", err)
	}

	requestsAfterFirst := jwks.requests.Load()
	started := time.Now()

	if _, err := verifier.Verify(sign(t, key, "unknown-2", validClaims())); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("second unknown key: Verify() error = %v", err)
	}

	if jwks.requests.Load() != requestsAfterFirst || time.Since(started) > 2*time.Second {
		t.Fatalf("second unknown key caused %d more JWKS requests after %v",
			jwks.requests.Load()-requestsAfterFirst, time.Since(started))
	}
}
