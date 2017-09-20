package middleware_test

import (
	"net/http"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"net/http/httptest"

	"github.com/weaveworks/common/middleware"
	"github.com/weaveworks/common/user"
)

func TestLogContextHandler(t *testing.T) {
	r := mux.NewRouter()
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		orgID, err := user.ExtractOrgID(r.Context())
		assert.NoError(t, err)
		assert.Equal(t, "11", orgID)

		userID, err := user.ExtractUserID(r.Context())
		assert.NoError(t, err)
		assert.Equal(t, "22", userID)
	})

	mw := middleware.LogContext{OrgIDName: "orgid", UserIDName: "userid"}
	r.Handle("/org/{orgid}/user/{userid}", mw.Wrap(h))
	w := httptest.NewRecorder()

	req, err := http.NewRequest("GET", "/org/11/user/22", nil)
	assert.NoError(t, err)

	r.ServeHTTP(w, req)
}
