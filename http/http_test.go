package http_test

import (
	"net/http"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"net/http/httptest"

	commonhttp "github.com/weaveworks/common/http"
	"github.com/weaveworks/common/user"
)

func TestLogContextHandler(t *testing.T) {
	r := mux.NewRouter()
	h := func(w http.ResponseWriter, r *http.Request) {
		orgID, err := user.ExtractOrgID(r.Context())
		assert.NoError(t, err)
		assert.Equal(t, "11", orgID)

		userID, err := user.ExtractUserID(r.Context())
		assert.NoError(t, err)
		assert.Equal(t, "22", userID)
	}

	r.HandleFunc("/org/{orgid}/user/{userid}",
		commonhttp.LogContextHandler(h, "orgid", "userid"))

	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/org/11/user/22", nil)
	assert.NoError(t, err)

	r.ServeHTTP(w, req)
}
