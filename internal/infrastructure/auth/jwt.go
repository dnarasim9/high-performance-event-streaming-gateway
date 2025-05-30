package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Claims represents the JWT claims structure.
type Claims struct {
	UserID string   `json:"user_id"`
	Roles  []string `json:"roles"`
	jwt.RegisteredClaims
}

// JWTAuthenticator handles JWT token generation and validation.
type JWTAuthenticator struct {
	secret      string
	issuer      string
	tokenExpiry time.Duration
}

// NewJWTAuthenticator creates a new JWTAuthenticator.
func NewJWTAuthenticator(secret, issuer string, tokenExpiry time.Duration) *JWTAuthenticator {
	return &JWTAuthenticator{
		secret:      secret,
		issuer:      issuer,
		tokenExpiry: tokenExpiry,
	}
}

// GenerateToken generates a new JWT token for the given user and roles.
func (ja *JWTAuthenticator) GenerateToken(userID string, roles []string) (string, error) {
	if userID == "" {
		return "", fmt.Errorf("user ID cannot be empty")
	}

	now := time.Now()
	expiresAt := now.Add(ja.tokenExpiry)

	claims := &Claims{
		UserID: userID,
		Roles:  roles,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    ja.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(ja.secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return signedToken, nil
}

// ValidateToken validates and parses a JWT token.
func (ja *JWTAuthenticator) ValidateToken(tokenString string) (*Claims, error) {
	if tokenString == "" {
		return nil, fmt.Errorf("token cannot be empty")
	}

	claims := &Claims{}

	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(token *jwt.Token) (interface{}, error) {
			// Verify the signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(ja.secret), nil
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("token is invalid")
	}

	return claims, nil
}

// ExtractClaims extracts JWT claims from the context.
func ExtractClaims(ctx context.Context) (*Claims, error) {
	claims, ok := ctx.Value(claimsContextKey).(*Claims)
	if !ok {
		return nil, fmt.Errorf("claims not found in context")
	}
	return claims, nil
}

// Context key for storing claims
type contextKeyType string

const claimsContextKey contextKeyType = "jwt_claims"

// WithClaims returns a new context with claims attached.
func WithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, claimsContextKey, claims)
}

// UnaryInterceptor returns a gRPC unary interceptor for JWT authentication.
func (ja *JWTAuthenticator) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Extract token from metadata
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		tokens := md.Get("authorization")
		if len(tokens) == 0 {
			return nil, status.Error(codes.Unauthenticated, "missing authorization token")
		}

		// Extract token (format: "Bearer <token>")
		tokenString := tokens[0]
		if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
			tokenString = tokenString[7:]
		}

		// Validate token
		claims, err := ja.ValidateToken(tokenString)
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
		}

		// Add claims to context
		ctx = WithClaims(ctx, claims)

		// Call handler
		return handler(ctx, req)
	}
}

// StreamInterceptor returns a gRPC stream interceptor for JWT authentication.
func (ja *JWTAuthenticator) StreamInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		// Extract token from metadata
		md, ok := metadata.FromIncomingContext(ss.Context())
		if !ok {
			return status.Error(codes.Unauthenticated, "missing metadata")
		}

		tokens := md.Get("authorization")
		if len(tokens) == 0 {
			return status.Error(codes.Unauthenticated, "missing authorization token")
		}

		// Extract token (format: "Bearer <token>")
		tokenString := tokens[0]
		if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
			tokenString = tokenString[7:]
		}

		// Validate token
		claims, err := ja.ValidateToken(tokenString)
		if err != nil {
			return status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
		}

		// Create wrapped stream with context containing claims
		ctx := WithClaims(ss.Context(), claims)
		wrappedStream := &wrappedServerStream{
			ServerStream: ss,
			ctx:          ctx,
		}

		// Call handler
		return handler(srv, wrappedStream)
	}
}

// wrappedServerStream wraps a gRPC server stream with a custom context.
type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}
