package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/janosmiko/vau/internal/vault"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeVault is an in-memory KV v1 mount named "secret". KV v1 keeps the
// HTTP surface small: every secret operation is a plain call on secret/<path>.
type fakeVault struct {
	mu         sync.Mutex
	secrets    map[string]map[string]string // path (no mount prefix) -> data
	failWrites map[string]bool              // paths whose writes return 500
}

// newFakeVault starts a fake Vault server and returns a real vault.Client
// that talks to it.
func newFakeVault(t *testing.T) (*vault.Client, *fakeVault) {
	t.Helper()
	fv := &fakeVault{
		secrets:    make(map[string]map[string]string),
		failWrites: make(map[string]bool),
	}
	srv := httptest.NewServer(http.HandlerFunc(fv.serve))
	t.Cleanup(srv.Close)

	t.Setenv("VAULT_ADDR", srv.URL)
	t.Setenv("VAULT_TOKEN", "test-token")
	t.Setenv("VAULT_MOUNT_PATH", "secret")
	// The API client retries a 500 with backoff, which adds seconds per failed write.
	t.Setenv("VAULT_MAX_RETRIES", "0")
	c, err := vault.NewClient()
	require.NoError(t, err)
	return c, fv
}

func (fv *fakeVault) put(path string, data map[string]string) {
	fv.mu.Lock()
	defer fv.mu.Unlock()
	fv.secrets[path] = copyMap(data)
}

func (fv *fakeVault) get(path string) (map[string]string, bool) {
	fv.mu.Lock()
	defer fv.mu.Unlock()
	d, ok := fv.secrets[path]
	return copyMap(d), ok
}

func (fv *fakeVault) failWrite(path string) {
	fv.mu.Lock()
	defer fv.mu.Unlock()
	fv.failWrites[path] = true
}

func (fv *fakeVault) serve(w http.ResponseWriter, r *http.Request) {
	fv.mu.Lock()
	defer fv.mu.Unlock()

	p := strings.TrimPrefix(r.URL.Path, "/v1/")
	if strings.HasPrefix(p, "sys/mounts/") {
		writeJSON(w, map[string]any{"data": map[string]any{"type": "kv", "options": map[string]any{"version": "1"}}})
		return
	}
	// Mount "secret" keys paths without a prefix. Any other mount keeps its "<mount>/" prefix.
	p = strings.TrimPrefix(p, "secret/")

	switch {
	case r.Method == "LIST" || (r.Method == http.MethodGet && r.URL.Query().Get("list") == "true"):
		fv.serveList(w, r, p)
	case r.Method == http.MethodGet:
		d, ok := fv.secrets[p]
		if !ok {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, map[string]any{"data": d})
	case r.Method == http.MethodPut || r.Method == http.MethodPost:
		if fv.failWrites[p] {
			http.Error(w, `{"errors":["write failed"]}`, http.StatusInternalServerError)
			return
		}
		var d map[string]string
		if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		fv.secrets[p] = d
		w.WriteHeader(http.StatusNoContent)
	case r.Method == http.MethodDelete:
		delete(fv.secrets, p)
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "unsupported", http.StatusMethodNotAllowed)
	}
}

func (fv *fakeVault) serveList(w http.ResponseWriter, r *http.Request, dir string) {
	prefix := strings.TrimSuffix(dir, "/")
	if prefix != "" {
		prefix += "/"
	}
	seen := map[string]bool{}
	for path := range fv.secrets {
		rest, ok := strings.CutPrefix(path, prefix)
		if !ok || rest == "" {
			continue
		}
		if i := strings.Index(rest, "/"); i >= 0 {
			rest = rest[:i+1]
		}
		seen[rest] = true
	}
	if len(seen) == 0 {
		http.NotFound(w, r)
		return
	}
	keys := make([]string, 0, len(seen))
	for k := range seen {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	writeJSON(w, map[string]any{"data": map[string]any{"keys": keys}})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func TestFakeVault_RoundTrip(t *testing.T) {
	c, fv := newFakeVault(t)
	fv.put("dir/a", map[string]string{"k": "v"})

	s, err := c.Read("dir/a")
	require.NoError(t, err)
	assert.Equal(t, "v", s.Data["k"])

	require.NoError(t, c.Write("dir/b", map[string]string{"x": "y"}))
	got, ok := fv.get("dir/b")
	assert.True(t, ok)
	assert.Equal(t, "y", got["x"])

	entries, err := c.List("dir/")
	require.NoError(t, err)
	assert.Len(t, entries, 2)

	require.NoError(t, c.Delete("dir/a"))
	_, ok = fv.get("dir/a")
	assert.False(t, ok)

	fv.failWrite("dir/c")
	assert.Error(t, c.Write("dir/c", map[string]string{"a": "b"}))
}
