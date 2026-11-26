package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Manager struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
	issuer     string
}

func NewManager(secret string, accessTTL, refreshTTL time.Duration) *Manager {
	return &Manager{secret: []byte(secret), accessTTL: accessTTL, refreshTTL: refreshTTL, issuer: "pharmaos"}
}

type AccessClaims struct {
	jwt.RegisteredClaims
	TenantID     string   `json:"tid"`
	SessionID    string   `json:"sid"`
	Email        string   `json:"email,omitempty"`
	Roles        []string `json:"roles,omitempty"`
	Permissions  []string `json:"perms,omitempty"`
	DataScope    string   `json:"scope,omitempty"`
	DivisionIDs  []string `json:"divs,omitempty"`
	TerritoryIDs []string `json:"terrs,omitempty"`
	IsPrivileged bool     `json:"priv,omitempty"`
}

type RefreshClaims struct {
	jwt.RegisteredClaims
	TenantID  string `json:"tid"`
	SessionID string `json:"sid"`
	FamilyID  string `json:"fid"`
}

func (m *Manager) GenerateAccess(principal Principal, sessionID uuid.UUID) (string, string, error) {
	now := time.Now().UTC()
	jti := uuid.New()
	permissions := make([]string, 0, len(principal.Permissions))
	for code := range principal.Permissions {
		permissions = append(permissions, code)
	}
	claims := AccessClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: m.issuer, Subject: principal.UserID.String(), ID: jti.String(),
			IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(m.accessTTL)),
		},
		TenantID: principal.TenantID.String(), SessionID: sessionID.String(), Email: principal.Email,
		Roles: principal.Roles, Permissions: permissions, DataScope: principal.DataScope,
		DivisionIDs: uuidStrings(principal.DivisionIDs), TerritoryIDs: uuidStrings(principal.TerritoryIDs),
		IsPrivileged: principal.IsPrivileged,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	return signed, jti.String(), err
}

func (m *Manager) GenerateRefresh(userID, tenantID, sessionID, familyID uuid.UUID) (string, string, error) {
	now := time.Now().UTC()
	jti := uuid.New()
	claims := RefreshClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: m.issuer, Subject: userID.String(), ID: jti.String(),
			IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(m.refreshTTL)),
		},
		TenantID: tenantID.String(), SessionID: sessionID.String(), FamilyID: familyID.String(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	return signed, jti.String(), err
}

func (m *Manager) ParseAccess(tokenString string) (*AccessClaims, error) {
	claims := &AccessClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, jwt.ErrSignatureInvalid
		}
		return m.secret, nil
	}, jwt.WithIssuer(m.issuer), jwt.WithExpirationRequired())
	if err != nil || !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return claims, nil
}

func (m *Manager) ParseRefresh(tokenString string) (*RefreshClaims, error) {
	claims := &RefreshClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, jwt.ErrSignatureInvalid
		}
		return m.secret, nil
	}, jwt.WithIssuer(m.issuer), jwt.WithExpirationRequired())
	if err != nil || !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return claims, nil
}

func (m *Manager) RefreshTTL() time.Duration { return m.refreshTTL }

func uuidStrings(values []uuid.UUID) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, value.String())
	}
	return out
}
