package user

import (
	"net/http"
	"net/textproto"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

func TestExtractOrgIDFromHTTPRequest(t *testing.T) {
	for _, tc := range []struct {
		name           string
		headerSet      func(*http.Request)
		expectedOrgID  string
		expectedOrgIDs []string
		expectedError  error
	}{
		{
			name:          "no org ID given",
			expectedError: ErrNoOrgID,
		},
		{
			name: "empty org ID",
			headerSet: func(r *http.Request) {
				r.Header.Set(OrgIDHeaderName, "")
			},
			expectedOrgID:  "",
			expectedOrgIDs: []string{""},
		},
		{
			name: "single org ID",
			headerSet: func(r *http.Request) {
				r.Header.Set(OrgIDHeaderName, "my-org")
			},
			expectedOrgID:  "my-org",
			expectedOrgIDs: []string{"my-org"},
		},
		{
			name: "multiple org IDs",
			headerSet: func(r *http.Request) {
				r.Header.Add(OrgIDHeaderName, "my-org")
				r.Header.Add(OrgIDHeaderName, "my-org-2")
			},
			expectedOrgID:  "my-org",
			expectedOrgIDs: []string{"my-org", "my-org-2"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "http://example.com", nil)
			if tc.headerSet != nil {
				tc.headerSet(req)
			}

			orgID, ctx, err := ExtractOrgIDFromHTTPRequest(req)
			if tc.expectedError != nil {
				assert.EqualError(t, err, tc.expectedError.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedOrgID, orgID)
			}

			orgID, err = ExtractOrgID(ctx)
			if tc.expectedError != nil {
				assert.EqualError(t, err, tc.expectedError.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedOrgID, orgID)
			}

			orgIDs, err := ExtractOrgIDs(ctx)
			if tc.expectedError != nil {
				assert.EqualError(t, err, tc.expectedError.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedOrgIDs, orgIDs)
			}
		})
	}
}

func TestInjectOgIDIntoHTTPRequest(t *testing.T) {
	for _, tc := range []struct {
		name           string
		contextSet     func(context.Context) context.Context
		expectedHeader []string
		expectedError  error
	}{
		{
			name:          "no org ID",
			expectedError: ErrNoOrgID,
		},
		{
			name: "empty org ID",
			contextSet: func(ctx context.Context) context.Context {
				return InjectOrgID(ctx, "")
			},
			expectedHeader: []string{""},
		},
		{
			name: "single org ID",
			contextSet: func(ctx context.Context) context.Context {
				return InjectOrgID(ctx, "my-org")
			},
			expectedHeader: []string{"my-org"},
		},
		{
			name: "multiple org IDs",
			contextSet: func(ctx context.Context) context.Context {
				return InjectOrgIDs(ctx, []string{"my-org", "my-org-2"})
			},
			expectedHeader: []string{"my-org", "my-org-2"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			if tc.contextSet != nil {
				ctx = tc.contextSet(ctx)
			}

			req, _ := http.NewRequest("GET", "http://example.com", nil)
			err := InjectOrgIDIntoHTTPRequest(ctx, req)
			if tc.expectedError != nil {
				assert.EqualError(t, err, tc.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}

			h := req.Header[textproto.CanonicalMIMEHeaderKey(OrgIDHeaderName)]

			assert.Equal(t, tc.expectedHeader, h)
		})
	}
}
