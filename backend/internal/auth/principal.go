package auth

import (
	"context"

	"github.com/google/uuid"
)

type Principal struct {
	UserID       uuid.UUID
	TenantID     uuid.UUID
	SessionID    uuid.UUID
	Email        string
	Roles        []string
	Permissions  map[string]bool
	DataScope    string
	DivisionIDs  []uuid.UUID
	TerritoryIDs []uuid.UUID
	IsPrivileged bool
}

func (p Principal) HasPermission(permission string) bool {
	if p.IsPrivileged {
		return true
	}
	return p.Permissions[permission]
}

func (p Principal) HasRole(role string) bool {
	for _, value := range p.Roles {
		if value == role {
			return true
		}
	}
	return false
}

type principalKey struct{}

func WithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, principal)
}

func PrincipalFrom(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(principalKey{}).(Principal)
	return principal, ok
}

func MustPrincipal(ctx context.Context) Principal {
	principal, _ := PrincipalFrom(ctx)
	return principal
}
