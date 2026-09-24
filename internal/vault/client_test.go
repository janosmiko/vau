package vault

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newKV1Server serves every mount as KV v1 and every secret as {"k": "<mount>"}.
func newKV1Server(t *testing.T) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		p := strings.TrimPrefix(r.URL.Path, "/v1/")
		if strings.HasPrefix(p, "sys/mounts/") {
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"options": map[string]any{"version": "1"}}})
			return
		}
		mount, _, _ := strings.Cut(p, "/")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"k": mount}})
	}))
	t.Cleanup(srv.Close)

	t.Setenv("VAULT_ADDR", srv.URL)
	t.Setenv("VAULT_TOKEN", "test-token")
	t.Setenv("VAULT_MOUNT_PATH", "secret")
	c, err := NewClient()
	require.NoError(t, err)
	return c
}

func TestWithMount_LeavesOriginalUnchanged(t *testing.T) {
	c := newKV1Server(t)

	other := c.WithMount("other/")

	assert.Equal(t, "secret", c.Mount())
	assert.Equal(t, "other", other.Mount())

	s, err := c.Read("x")
	require.NoError(t, err)
	assert.Equal(t, "secret", s.Data["k"])
	s, err = other.Read("x")
	require.NoError(t, err)
	assert.Equal(t, "other", s.Data["k"])
}

func TestClient_ConcurrentMountVersionCache(t *testing.T) {
	c := newKV1Server(t)

	var wg sync.WaitGroup
	for i := range 20 {
		wg.Go(func() {
			mount := fmt.Sprintf("m%d", i%5)
			_, err := c.ListWithMount(mount, "")
			assert.NoError(t, err)
			s, err := c.WithMount(mount).Read("x")
			if assert.NoError(t, err) {
				assert.Equal(t, mount, s.Data["k"])
			}
		})
	}
	wg.Wait()
}
