package auth

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestJWTAuthenticator_GenerateToken_Success(t *testing.T) {
	authenticator := NewJWTAuthenticator("test-secret", "test-issuer", 1*time.Hour)

	token, err := authenticator.GenerateToken("user-123", []string{"admin", "user"})
	if err != nil {
		t.Errorf("GenerateToken() error = %v, want nil", err)
	}

	if token == "" {
		t.Error("GenerateToken() returned empty token")
	}
}

func TestJWTAuthenticator_GenerateToken_EmptyUserID(t *testing.T) {
	authenticator := NewJWTAuthenticator("test-secret", "test-issuer", 1*time.Hour)

	_, err := authenticator.GenerateToken("", []string{"admin"})
	if err == nil {
		t.Error("GenerateToken() with empty user ID expected error, got nil")
	}
}

func TestJWTAuthenticator_GenerateToken_NoRoles(t *testing.T) {
	authenticator := NewJWTAuthenticator("test-secret", "test-issuer", 1*time.Hour)

	token, err := authenticator.GenerateToken("user-123", []string{})
	if err != nil {
		t.Errorf("GenerateToken() error = %v, want nil", err)
	}

	if token == "" {
		t.Error("GenerateToken() returned empty token")
	}
}

func TestJWTAuthenticator_GenerateToken_NilRoles(t *testing.T) {
	authenticator := NewJWTAuthenticator("test-secret", "test-issuer", 1*time.Hour)

	token, err := authenticator.GenerateToken("user-123", nil)
	if err != nil {
		t.Errorf("GenerateToken() error = %v, want nil", err)
	}

	if token == "" {
		t.Error("GenerateToken() returned empty token")
	}
}

func TestJWTAuthenticator_ValidateToken_Valid(t *testing.T) {
	authenticator := NewJWTAuthenticator("test-secret", "test-issuer", 1*time.Hour)

	tokenString, _ := authenticator.GenerateToken("user-123", []string{"admin"})

	claims, err := authenticator.ValidateToken(tokenString)
	if err != nil {
		t.Errorf("ValidateToken() error = %v, want nil", err)
	}

	if claims == nil {
		t.Error("ValidateToken() returned nil claims")
	}

	if claims.UserID != "user-123" {
		t.Errorf("ValidateToken() UserID = %q, want %q", claims.UserID, "user-123")
	}

	if len(claims.Roles) != 1 || claims.Roles[0] != "admin" {
		t.Errorf("ValidateToken() Roles = %v, want [admin]", claims.Roles)
	}
}

func TestJWTAuthenticator_ValidateToken_EmptyToken(t *testing.T) {
	authenticator := NewJWTAuthenticator("test-secret", "test-issuer", 1*time.Hour)

	_, err := authenticator.ValidateToken("")
	if err == nil {
		t.Error("ValidateToken() with empty token expected error, got nil")
	}
}

func TestJWTAuthenticator_ValidateToken_InvalidSignature(t *testing.T) {
	authenticator := NewJWTAuthenticator("test-secret", "test-issuer", 1*time.Hour)

	tokenString, _ := authenticator.GenerateToken("user-123", []string{"admin"})

	// Use different secret to validate
	authenticator2 := NewJWTAuthenticator("different-secret", "test-issuer", 1*time.Hour)

	_, err := authenticator2.ValidateToken(tokenString)
	if err == nil {
		t.Error("ValidateToken() with invalid signature expected error, got nil")
	}
}

func TestJWTAuthenticator_ValidateToken_Expired(t *testing.T) {
	authenticator := NewJWTAuthenticator("test-secret", "test-issuer", 1*time.Millisecond)

	tokenString, _ := authenticator.GenerateToken("user-123", []string{"admin"})

	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)

	_, err := authenticator.ValidateToken(tokenString)
	if err == nil {
		t.Error("ValidateToken() with expired token expected error, got nil")
	}
}

func TestJWTAuthenticator_ValidateToken_MalformedToken(t *testing.T) {
	authenticator := NewJWTAuthenticator("test-secret", "test-issuer", 1*time.Hour)

	_, err := authenticator.ValidateToken("malformed.token")
	if err == nil {
		t.Error("ValidateToken() with malformed token expected error, got nil")
	}
}

func TestJWTAuthenticator_Claims_Issuer(t *testing.T) {
	authenticator := NewJWTAuthenticator("test-secret", "my-issuer", 1*time.Hour)

	tokenString, _ := authenticator.GenerateToken("user-123", []string{"admin"})

	claims, _ := authenticator.ValidateToken(tokenString)

	if claims.Issuer != "my-issuer" {
		t.Errorf("Claims Issuer = %q, want %q", claims.Issuer, "my-issuer")
	}
}

func TestJWTAuthenticator_Claims_IssuedAt(t *testing.T) {
	authenticator := NewJWTAuthenticator("test-secret", "test-issuer", 1*time.Hour)

	before := time.Now().Add(-1 * time.Second)
	tokenString, _ := authenticator.GenerateToken("user-123", []string{})
	after := time.Now().Add(1 * time.Second)

	claims, _ := authenticator.ValidateToken(tokenString)

	issuedAt := claims.IssuedAt.Time
	if issuedAt.Before(before) || issuedAt.After(after) {
		t.Errorf("Claims IssuedAt = %v, want between %v and %v", issuedAt, before, after)
	}
}

func TestJWTAuthenticator_Claims_ExpiresAt(t *testing.T) {
	expiry := 2 * time.Hour
	authenticator := NewJWTAuthenticator("test-secret", "test-issuer", expiry)

	before := time.Now().Add(expiry - 1*time.Second)
	tokenString, _ := authenticator.GenerateToken("user-123", []string{})
	after := time.Now().Add(expiry + 1*time.Second)

	claims, _ := authenticator.ValidateToken(tokenString)

	expiresAt := claims.ExpiresAt.Time
	if expiresAt.Before(before) || expiresAt.After(after) {
		t.Errorf("Claims ExpiresAt = %v, want between %v and %v", expiresAt, before, after)
	}
}

func TestJWTAuthenticator_MultipleRoles(t *testing.T) {
	authenticator := NewJWTAuthenticator("test-secret", "test-issuer", 1*time.Hour)

	roles := []string{"admin", "user", "moderator"}
	tokenString, _ := authenticator.GenerateToken("user-123", roles)

	claims, _ := authenticator.ValidateToken(tokenString)

	if len(claims.Roles) != len(roles) {
		t.Errorf("Claims Roles count = %d, want %d", len(claims.Roles), len(roles))
	}

	for i, role := range roles {
		if i < len(claims.Roles) && claims.Roles[i] != role {
			t.Errorf("Claims Roles[%d] = %q, want %q", i, claims.Roles[i], role)
		}
	}
}

func TestWithClaims(t *testing.T) {
	claims := &Claims{
		UserID: "user-123",
		Roles:  []string{"admin"},
	}

	ctx := WithClaims(context.Background(), claims)

	extracted, _ := ExtractClaims(ctx)
	if extracted == nil {
		t.Error("ExtractClaims() returned nil")
	}

	if extracted.UserID != "user-123" {
		t.Errorf("ExtractClaims() UserID = %q, want %q", extracted.UserID, "user-123")
	}
}

func TestExtractClaims_NotFound(t *testing.T) {
	ctx := context.Background()

	_, err := ExtractClaims(ctx)
	if err == nil {
		t.Error("ExtractClaims() without claims expected error, got nil")
	}
}

func TestJWTAuthenticator_TokenSigningMethod(t *testing.T) {
	authenticator := NewJWTAuthenticator("test-secret", "test-issuer", 1*time.Hour)

	tokenString, _ := authenticator.GenerateToken("user-123", []string{"admin"})

	// Manually parse to check signing method
	token, _ := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte("test-secret"), nil
	})

	if token.Method.Alg() != "HS256" {
		t.Errorf("Token signing method = %q, want HS256", token.Method.Alg())
	}
}
