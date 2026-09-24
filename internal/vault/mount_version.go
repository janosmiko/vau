package vault

import (
	"context"
	"errors"
	"net/http"
	"sync"

	vaultapi "github.com/hashicorp/vault/api"
)

// versionCache maps mount path -> KV version (1 or 2). Copies of a Client share it.
type versionCache struct {
	mu sync.Mutex
	m  map[string]int
}

// getMountVersion detects and caches the KV engine version for the given mount.
// It queries sys/mounts/{mount} and inspects options.version.
// Returns 1 for KV v1 or 2 for KV v2 (the default).
func (c *Client) getMountVersion(mount string) int {
	return c.getMountVersionCtx(context.Background(), mount)
}

func (c *Client) getMountVersionCtx(ctx context.Context, mount string) int {
	c.mountVersions.mu.Lock()
	v, ok := c.mountVersions.m[mount]
	c.mountVersions.mu.Unlock()
	if ok {
		return v
	}
	v = 2
	secret, err := c.raw.Logical().ReadWithContext(ctx, "sys/mounts/"+mount)
	var respErr *vaultapi.ResponseError
	if err != nil && (!errors.As(err, &respErr) || respErr.StatusCode >= http.StatusInternalServerError) {
		// Cache a 4xx, because it does not go away and would cost a probe per call.
		// A 5xx or network error can be transient, and a cached v2 fallback would
		// send all later KV v1 calls to v2 paths.
		return v
	}
	if err == nil && secret != nil {
		if options, ok := secret.Data["options"].(map[string]any); ok {
			if version, ok := options["version"].(string); ok && version == "1" {
				v = 1
			}
		}
	}
	c.mountVersions.mu.Lock()
	c.mountVersions.m[mount] = v
	c.mountVersions.mu.Unlock()
	return v
}

// IsKV1 reports whether the current mount uses the KV v1 engine.
func (c *Client) IsKV1() bool {
	return c.getMountVersion(c.mount) == 1
}
