package httpgrpc

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHTTPResponseFromError(t *testing.T) {
	t.Run("context canceled", func(t *testing.T) {
		err := fmt.Errorf("something failed: %w", context.Canceled)
		resp, ok := HTTPResponseFromError(err)
		require.True(t, ok)
		require.Equal(t, int32(StatusClientClosedRequest), resp.Code)
		require.Equal(t, err.Error(), string(resp.Body))
	})
}
