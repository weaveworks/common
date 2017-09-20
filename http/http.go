package http

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/weaveworks/common/user"
)

// LogContextHandler wraps a HTTP handler to inject organization id and user id
// into the context to enrich log entries.
//
// It reads the IDs from named path parameters found in `mux.Vars()`.
func LogContextHandler(h http.HandlerFunc, orgIDName, userIDName string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		ctx := r.Context()
		if orgID, ok := vars[orgIDName]; ok {
			ctx = user.InjectOrgID(ctx, orgID)
		}
		if userID, ok := vars[userIDName]; ok {
			ctx = user.InjectUserID(ctx, userID)
		}
		h(w, r.WithContext(ctx))
	})
}
