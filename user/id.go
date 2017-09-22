package user

import (
	"golang.org/x/net/context"

	"github.com/weaveworks/common/errors"
)

type contextKey int

const (
	// Keys used in contexts to find the org or user ID
	orgIDContextKey  contextKey = 0
	userIDContextKey contextKey = 1
)

// Errors that we return
const (
	ErrNoOrgID  = errors.Error("no org id")
	ErrNoUserID = errors.Error("no user id")
)

// ExtractOrgID gets the org ID from the context.
func ExtractOrgID(ctx context.Context) (string, error) {
	orgID, ok := ctx.Value(orgIDContextKey).(string)
	if !ok {
		return "", ErrNoOrgID
	}
	return orgID, nil
}

// InjectOrgID returns a derived context containing the org ID.
func InjectOrgID(ctx context.Context, orgID string) context.Context {
	return context.WithValue(ctx, interface{}(orgIDContextKey), orgID)
}

// ExtractUserID gets the user ID from the context.
func ExtractUserID(ctx context.Context) (string, error) {
	userID, ok := ctx.Value(userIDContextKey).(string)
	if !ok {
		return "", ErrNoUserID
	}
	return userID, nil
}

// InjectUserID returns a derived context containing the user ID.
func InjectUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, interface{}(userIDContextKey), userID)
}
