package vault

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/janosmiko/vau/internal/model"

	vaultapi "github.com/hashicorp/vault/api"
)

// ErrKV1NotSupported is returned when a KV v2-only operation is attempted on a KV v1 mount.
var ErrKV1NotSupported = fmt.Errorf("operation not supported for KV v1 engine")

// Client wraps the Vault API client.
type Client struct {
	raw           *vaultapi.Client
	mount         string
	mountVersions map[string]int // cache: mount path -> KV version (1 or 2)
}

// NewClient creates a new Vault client from environment variables.
func NewClient() (*Client, error) {
	config := vaultapi.DefaultConfig()
	if addr := os.Getenv("VAULT_ADDR"); addr != "" {
		config.Address = addr
	}

	raw, err := vaultapi.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("creating vault client: %w", err)
	}

	token := os.Getenv("VAULT_TOKEN")
	if token == "" {
		// Fall back to ~/.vault-token file (written by "vault login").
		home, err := os.UserHomeDir()
		if err == nil {
			data, err := os.ReadFile(filepath.Join(home, ".vault-token"))
			if err == nil {
				token = strings.TrimSpace(string(data))
			}
		}
	}
	if token == "" {
		return nil, fmt.Errorf("vault token not found: set VAULT_TOKEN or log in with \"vault login\" (~/.vault-token)")
	}
	raw.SetToken(token)

	mount := os.Getenv("VAULT_MOUNT_PATH")
	if mount == "" {
		mount = "secret"
	}
	mount = strings.TrimSuffix(mount, "/")

	return &Client{raw: raw, mount: mount, mountVersions: make(map[string]int)}, nil
}

// Mount returns the configured mount path.
func (c *Client) Mount() string {
	return c.mount
}

// SetMount changes the active mount path.
func (c *Client) SetMount(mount string) {
	c.mount = strings.TrimSuffix(mount, "/")
}

// getMountVersion detects and caches the KV engine version for the given mount.
// It queries sys/mounts/{mount} and inspects options.version.
// Returns 1 for KV v1 or 2 for KV v2 (the default).
func (c *Client) getMountVersion(mount string) int {
	if v, ok := c.mountVersions[mount]; ok {
		return v
	}
	secret, err := c.raw.Logical().Read("sys/mounts/" + mount)
	if err != nil || secret == nil {
		c.mountVersions[mount] = 2
		return 2
	}
	if options, ok := secret.Data["options"].(map[string]interface{}); ok {
		if version, ok := options["version"].(string); ok && version == "1" {
			c.mountVersions[mount] = 1
			return 1
		}
	}
	c.mountVersions[mount] = 2
	return 2
}

// IsKV1 reports whether the current mount uses the KV v1 engine.
func (c *Client) IsKV1() bool {
	return c.getMountVersion(c.mount) == 1
}

// ListMounts returns available KV secret engine mounts.
func (c *Client) ListMounts() ([]string, error) {
	mounts, err := c.raw.Sys().ListMounts()
	if err != nil {
		return nil, fmt.Errorf("listing mounts: %w", err)
	}

	var kvMounts []string
	for path, mount := range mounts {
		if mount.Type == "kv" || mount.Type == "generic" {
			kvMounts = append(kvMounts, strings.TrimSuffix(path, "/"))
		}
	}
	sort.Strings(kvMounts)
	return kvMounts, nil
}

// ListWithMount returns entries at the given path using the specified mount,
// without modifying the client's active mount. Safe for concurrent use.
func (c *Client) ListWithMount(mount, path string) ([]model.Entry, error) {
	var apiPath string
	if c.getMountVersion(mount) == 1 {
		apiPath = fmt.Sprintf("%s/%s", mount, path)
	} else {
		apiPath = fmt.Sprintf("%s/metadata/%s", mount, path)
	}
	secret, err := c.raw.Logical().List(apiPath)
	if err != nil {
		return nil, fmt.Errorf("listing %s: %w", path, err)
	}
	if secret == nil || secret.Data == nil {
		return nil, nil
	}

	keysRaw, ok := secret.Data["keys"]
	if !ok {
		return nil, nil
	}

	keysList, ok := keysRaw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected keys type at %s", path)
	}

	entries := make([]model.Entry, 0, len(keysList))
	for _, k := range keysList {
		name := fmt.Sprintf("%v", k)
		isDir := strings.HasSuffix(name, "/")
		entries = append(entries, model.Entry{
			Name:  name,
			IsDir: isDir,
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return entries[i].Name < entries[j].Name
	})

	return entries, nil
}

// List returns entries at the given path within the KV engine.
func (c *Client) List(path string) ([]model.Entry, error) {
	var apiPath string
	if c.getMountVersion(c.mount) == 1 {
		apiPath = fmt.Sprintf("%s/%s", c.mount, path)
	} else {
		apiPath = fmt.Sprintf("%s/metadata/%s", c.mount, path)
	}
	secret, err := c.raw.Logical().List(apiPath)
	if err != nil {
		return nil, fmt.Errorf("listing %s: %w", path, err)
	}
	if secret == nil || secret.Data == nil {
		return nil, nil
	}

	keysRaw, ok := secret.Data["keys"]
	if !ok {
		return nil, nil
	}

	keysList, ok := keysRaw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected keys type at %s", path)
	}

	entries := make([]model.Entry, 0, len(keysList))
	for _, k := range keysList {
		name := fmt.Sprintf("%v", k)
		isDir := strings.HasSuffix(name, "/")
		entries = append(entries, model.Entry{
			Name:  name,
			IsDir: isDir,
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		// directories first, then alphabetical
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return entries[i].Name < entries[j].Name
	})

	return entries, nil
}

// Read reads a secret at the given path.
func (c *Client) Read(path string) (*model.Secret, error) {
	isV1 := c.getMountVersion(c.mount) == 1

	var apiPath string
	if isV1 {
		apiPath = fmt.Sprintf("%s/%s", c.mount, path)
	} else {
		apiPath = fmt.Sprintf("%s/data/%s", c.mount, path)
	}

	secret, err := c.raw.Logical().Read(apiPath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("secret not found at %s", path)
	}

	var dataMap map[string]interface{}
	if isV1 {
		// KV v1: data is directly in secret.Data
		dataMap = secret.Data
	} else {
		// KV v2: data is nested under secret.Data["data"]
		dataRaw, ok := secret.Data["data"]
		if !ok || dataRaw == nil {
			return &model.Secret{Path: path, Data: make(map[string]string), Keys: nil}, nil
		}
		dataMap, ok = dataRaw.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("unexpected data type at %s", path)
		}
	}

	data := make(map[string]string, len(dataMap))
	keys := make([]string, 0, len(dataMap))
	for k, v := range dataMap {
		data[k] = fmt.Sprintf("%v", v)
		keys = append(keys, k)
	}
	sort.Strings(keys)

	return &model.Secret{
		Path: path,
		Data: data,
		Keys: keys,
	}, nil
}

// Write writes a secret at the given path.
func (c *Client) Write(path string, data map[string]string) error {
	isV1 := c.getMountVersion(c.mount) == 1

	var apiPath string
	var payload map[string]interface{}

	if isV1 {
		apiPath = fmt.Sprintf("%s/%s", c.mount, path)
		// KV v1: write data directly
		payload = make(map[string]interface{}, len(data))
		for k, v := range data {
			payload[k] = v
		}
	} else {
		apiPath = fmt.Sprintf("%s/data/%s", c.mount, path)
		// KV v2: wrap in {"data": ...}
		payload = map[string]interface{}{
			"data": data,
		}
	}

	_, err := c.raw.Logical().Write(apiPath, payload)
	if err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}

// Delete deletes a secret at the given path.
func (c *Client) Delete(path string) error {
	var apiPath string
	if c.getMountVersion(c.mount) == 1 {
		apiPath = fmt.Sprintf("%s/%s", c.mount, path)
	} else {
		apiPath = fmt.Sprintf("%s/metadata/%s", c.mount, path)
	}
	_, err := c.raw.Logical().Delete(apiPath)
	if err != nil {
		return fmt.Errorf("deleting %s: %w", path, err)
	}
	return nil
}

// DeleteRecursive deletes a secret or directory and all its children.
func (c *Client) DeleteRecursive(path string) (int, error) {
	// First try to list children (it's a directory)
	entries, err := c.List(path)
	if err != nil || entries == nil {
		// Not a directory or doesn't exist — delete as a single secret
		if err := c.Delete(path); err != nil {
			return 0, err
		}
		return 1, nil
	}

	// Ensure path has trailing slash for child path construction.
	dirPath := path
	if !strings.HasSuffix(dirPath, "/") {
		dirPath += "/"
	}

	count := 0
	for _, e := range entries {
		childPath := dirPath + e.Name
		if e.IsDir {
			n, err := c.DeleteRecursive(strings.TrimSuffix(childPath, "/"))
			if err != nil {
				return count, fmt.Errorf("deleting %s: %w", childPath, err)
			}
			count += n
		} else {
			if err := c.Delete(childPath); err != nil {
				return count, fmt.Errorf("deleting %s: %w", childPath, err)
			}
			count++
		}
	}
	return count, nil
}

// ReadVersionMetadata reads version metadata for a secret (KV v2 only).
func (c *Client) ReadVersionMetadata(path string) ([]model.SecretVersion, error) {
	if c.getMountVersion(c.mount) == 1 {
		return nil, ErrKV1NotSupported
	}
	apiPath := fmt.Sprintf("%s/metadata/%s", c.mount, path)
	secret, err := c.raw.Logical().Read(apiPath)
	if err != nil {
		return nil, fmt.Errorf("reading metadata %s: %w", path, err)
	}
	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("no metadata at %s", path)
	}

	versionsRaw, ok := secret.Data["versions"]
	if !ok {
		return nil, fmt.Errorf("no versions data at %s", path)
	}

	versionsMap, ok := versionsRaw.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected versions type at %s", path)
	}

	var versions []model.SecretVersion
	for vNum, vData := range versionsMap {
		vMap, ok := vData.(map[string]interface{})
		if !ok {
			continue
		}
		sv := model.SecretVersion{
			Version: vNum,
		}
		if ct, ok := vMap["created_time"].(string); ok {
			sv.CreatedTime = ct
		}
		if dt, ok := vMap["deletion_time"].(string); ok {
			sv.DeletionTime = dt
		}
		if d, ok := vMap["destroyed"].(bool); ok {
			sv.Destroyed = d
		}
		versions = append(versions, sv)
	}

	// Sort by version number (newest first)
	sort.Slice(versions, func(i, j int) bool {
		vi, _ := strconv.Atoi(versions[i].Version)
		vj, _ := strconv.Atoi(versions[j].Version)
		return vi > vj
	})

	return versions, nil
}

// ReadVersion reads a specific version of a secret (KV v2 only).
func (c *Client) ReadVersion(path string, version int) (*model.Secret, error) {
	if c.getMountVersion(c.mount) == 1 {
		return nil, ErrKV1NotSupported
	}
	apiPath := fmt.Sprintf("%s/data/%s", c.mount, path)
	secret, err := c.raw.Logical().ReadWithData(apiPath, map[string][]string{
		"version": {strconv.Itoa(version)},
	})
	if err != nil {
		return nil, fmt.Errorf("reading %s version %d: %w", path, version, err)
	}
	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("version %d not found at %s", version, path)
	}

	dataRaw, ok := secret.Data["data"]
	if !ok || dataRaw == nil {
		return &model.Secret{Path: path, Data: make(map[string]string), Keys: nil}, nil
	}

	dataMap, ok := dataRaw.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected data type at %s v%d", path, version)
	}

	data := make(map[string]string, len(dataMap))
	keys := make([]string, 0, len(dataMap))
	for k, v := range dataMap {
		data[k] = fmt.Sprintf("%v", v)
		keys = append(keys, k)
	}
	sort.Strings(keys)

	return &model.Secret{Path: path, Data: data, Keys: keys}, nil
}

// DestroyVersions permanently destroys specific versions of a secret (KV v2 only).
func (c *Client) DestroyVersions(path string, versions []int) error {
	if c.getMountVersion(c.mount) == 1 {
		return ErrKV1NotSupported
	}
	apiPath := fmt.Sprintf("%s/destroy/%s", c.mount, path)
	versionStrs := make([]interface{}, len(versions))
	for i, v := range versions {
		versionStrs[i] = v
	}
	_, err := c.raw.Logical().Write(apiPath, map[string]interface{}{
		"versions": versionStrs,
	})
	if err != nil {
		return fmt.Errorf("destroying versions of %s: %w", path, err)
	}
	return nil
}

// DestroyOldVersions destroys all versions except the latest for a secret (KV v2 only).
// Returns the number of versions destroyed.
func (c *Client) DestroyOldVersions(path string) (int, error) {
	if c.getMountVersion(c.mount) == 1 {
		return 0, ErrKV1NotSupported
	}
	versions, err := c.ReadVersionMetadata(path)
	if err != nil {
		return 0, err
	}
	if len(versions) <= 1 {
		return 0, nil
	}

	// versions is sorted newest-first; skip the first (latest).
	var toDestroy []int
	for _, v := range versions[1:] {
		if v.Destroyed {
			continue
		}
		vNum, _ := strconv.Atoi(v.Version)
		toDestroy = append(toDestroy, vNum)
	}
	if len(toDestroy) == 0 {
		return 0, nil
	}

	if err := c.DestroyVersions(path, toDestroy); err != nil {
		return 0, err
	}
	return len(toDestroy), nil
}

// CopyWithHistory copies a secret from src to dst, preserving all KV v2 versions.
// For KV v1, copies only the current data (no version history exists).
func (c *Client) CopyWithHistory(src, dst string) error {
	if c.getMountVersion(c.mount) == 1 {
		secret, err := c.Read(src)
		if err != nil {
			return fmt.Errorf("reading source %s: %w", src, err)
		}
		return c.Write(dst, secret.Data)
	}

	// KV v2: read all versions and replay them oldest-first.
	versions, err := c.ReadVersionMetadata(src)
	if err != nil {
		// Fall back to single-version copy if metadata is unavailable.
		secret, err := c.Read(src)
		if err != nil {
			return fmt.Errorf("reading source %s: %w", src, err)
		}
		return c.Write(dst, secret.Data)
	}

	// Sort oldest-first (ReadVersionMetadata returns newest-first).
	sort.Slice(versions, func(i, j int) bool {
		vi, _ := strconv.Atoi(versions[i].Version)
		vj, _ := strconv.Atoi(versions[j].Version)
		return vi < vj
	})

	written := 0
	for _, v := range versions {
		if v.Destroyed || (v.DeletionTime != "" && v.DeletionTime != "0001-01-01T00:00:00Z") {
			continue
		}
		vNum, _ := strconv.Atoi(v.Version)
		secret, err := c.ReadVersion(src, vNum)
		if err != nil {
			continue // skip unreadable versions
		}
		if err := c.Write(dst, secret.Data); err != nil {
			return fmt.Errorf("writing version %d to %s: %w", vNum, dst, err)
		}
		written++
	}

	// If no versions were written (all destroyed/deleted), copy latest as fallback.
	if written == 0 {
		secret, err := c.Read(src)
		if err != nil {
			return fmt.Errorf("reading source %s: %w", src, err)
		}
		return c.Write(dst, secret.Data)
	}
	return nil
}

// CopyRecursiveWithHistory copies a secret or directory recursively, preserving version history.
// Returns the number of secrets copied.
func (c *Client) CopyRecursiveWithHistory(src, dst string) (int, error) {
	entries, err := c.List(src)
	if err != nil || entries == nil {
		// Single secret.
		if err := c.CopyWithHistory(src, dst); err != nil {
			return 0, err
		}
		return 1, nil
	}

	srcDir := src
	if !strings.HasSuffix(srcDir, "/") {
		srcDir += "/"
	}
	dstDir := dst
	if !strings.HasSuffix(dstDir, "/") {
		dstDir += "/"
	}

	count := 0
	for _, e := range entries {
		childSrc := srcDir + e.Name
		childDst := dstDir + e.Name
		if e.IsDir {
			n, err := c.CopyRecursiveWithHistory(
				strings.TrimSuffix(childSrc, "/"),
				strings.TrimSuffix(childDst, "/"),
			)
			if err != nil {
				return count, fmt.Errorf("copying %s: %w", childSrc, err)
			}
			count += n
		} else {
			if err := c.CopyWithHistory(childSrc, childDst); err != nil {
				return count, fmt.Errorf("copying %s: %w", childSrc, err)
			}
			count++
		}
	}
	return count, nil
}

// Move copies a secret from src to dst (preserving version history) and deletes the source.
func (c *Client) Move(src, dst string) error {
	if err := c.CopyWithHistory(src, dst); err != nil {
		return fmt.Errorf("copying %s to %s: %w", src, dst, err)
	}
	if err := c.Delete(src); err != nil {
		return fmt.Errorf("deleting source %s: %w", src, err)
	}
	return nil
}

// MoveRecursive moves a secret or directory (and all its children) from src to dst.
// Preserves version history. Returns the number of secrets moved.
func (c *Client) MoveRecursive(src, dst string) (int, error) {
	n, err := c.CopyRecursiveWithHistory(src, dst)
	if err != nil {
		return n, err
	}
	if _, err := c.DeleteRecursive(src); err != nil {
		return n, fmt.Errorf("deleting source %s after copy: %w", src, err)
	}
	return n, nil
}
