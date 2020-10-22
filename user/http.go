package user

import (
	"net/http"
	"net/textproto"
	"reflect"

	"golang.org/x/net/context"
)

const (
	// 'Scope' in the below headers is a legacy from scope as a service.

	// OrgIDHeaderName denotes the OrgID the request has been authenticated as
	OrgIDHeaderName = "X-Scope-OrgID"
	// UserIDHeaderName denotes the UserID the request has been authenticated as
	UserIDHeaderName = "X-Scope-UserID"

	// LowerOrgIDHeaderName as gRPC / HTTP2.0 headers are lowercased.
	lowerOrgIDHeaderName = "x-scope-orgid"
)

// ExtractOrgIDFromHTTPRequest extracts the org ID from the request headers and returns
// the org ID and a context with the org ID embedded.
func ExtractOrgIDFromHTTPRequest(r *http.Request) (string, context.Context, error) {
	orgIDs, ok := r.Header[textproto.CanonicalMIMEHeaderKey(OrgIDHeaderName)]
	if !ok || len(orgIDs) == 0 {
		return "", r.Context(), ErrNoOrgID
	}
	return orgIDs[0], InjectOrgIDs(r.Context(), orgIDs), nil
}

// InjectOrgIDIntoHTTPRequest injects the orgID from the context into the request headers.
func InjectOrgIDIntoHTTPRequest(ctx context.Context, r *http.Request) error {
	orgIDs, err := ExtractOrgIDs(ctx)
	if err != nil {
		return err
	}

	existingIDs := r.Header[textproto.CanonicalMIMEHeaderKey(OrgIDHeaderName)]
	if len(existingIDs) > 0 && !reflect.DeepEqual(existingIDs, orgIDs) {
		return ErrDifferentOrgIDPresent
	}

	for _, orgID := range orgIDs {
		r.Header.Add(OrgIDHeaderName, orgID)
	}
	return nil
}

// ExtractUserIDFromHTTPRequest extracts the org ID from the request headers and returns
// the org ID and a context with the org ID embedded.
func ExtractUserIDFromHTTPRequest(r *http.Request) (string, context.Context, error) {
	userID := r.Header.Get(UserIDHeaderName)
	if userID == "" {
		return "", r.Context(), ErrNoUserID
	}
	return userID, InjectUserID(r.Context(), userID), nil
}

// InjectUserIDIntoHTTPRequest injects the userID from the context into the request headers.
func InjectUserIDIntoHTTPRequest(ctx context.Context, r *http.Request) error {
	userID, err := ExtractUserID(ctx)
	if err != nil {
		return err
	}
	existingID := r.Header.Get(UserIDHeaderName)
	if existingID != "" && existingID != userID {
		return ErrDifferentUserIDPresent
	}
	r.Header.Set(UserIDHeaderName, userID)
	return nil
}
