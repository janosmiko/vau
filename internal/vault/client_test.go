package vault

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newKV1Server serves every mount as KV v1 and every secret as {"k": "<mount>"}.
func newKV1Server(t *testing.T) *Client {
	t.Helper()
	return newServerClient(t, func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, "/v1/")
		if strings.HasPrefix(p, "sys/mounts/") {
			writeKV1Mount(w)
			return
		}
		mount, _, _ := strings.Cut(p, "/")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"k": mount}})
	})
}

func writeKV1Mount(w http.ResponseWriter) {
	_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"options": map[string]any{"version": "1"}}})
}

func newServerClient(t *testing.T, h http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		h(w, r)
	}))
	t.Cleanup(srv.Close)

	t.Setenv("VAULT_ADDR", srv.URL)
	t.Setenv("VAULT_TOKEN", "test-token")
	t.Setenv("VAULT_MOUNT_PATH", "secret")
	t.Setenv("VAULT_MAX_RETRIES", "0")
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

func TestCountRecursive_StopsWhenCancelled(t *testing.T) {
	c := newKV1Server(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := c.CountRecursive(ctx, "dir")

	assert.ErrorIs(t, err, context.Canceled)
}

func TestCountRecursive_CancelInterruptsBlockedList(t *testing.T) {
	c := newServerClient(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "sys/mounts/") {
			writeKV1Mount(w)
			return
		}
		<-r.Context().Done()
	})
	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := c.CountRecursive(ctx, "dir")

	require.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Less(t, time.Since(start), 5*time.Second)
}

func TestGetMountVersion_TransientErrorIsNotCached(t *testing.T) {
	var probes atomic.Int32
	c := newServerClient(t, func(w http.ResponseWriter, _ *http.Request) {
		if probes.Add(1) == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		writeKV1Mount(w)
	})

	assert.False(t, c.IsKV1(), "the first probe fails, so the client falls back to v2")
	assert.True(t, c.IsKV1(), "a transient failure must not pin the v2 fallback")
}

func TestGetMountVersion_ForbiddenIsCached(t *testing.T) {
	var probes atomic.Int32
	c := newServerClient(t, func(w http.ResponseWriter, _ *http.Request) {
		probes.Add(1)
		w.WriteHeader(http.StatusForbidden)
	})

	c.IsKV1()
	c.IsKV1()

	assert.Equal(t, int32(1), probes.Load(), "a permission error must not trigger a probe on every call")
}
