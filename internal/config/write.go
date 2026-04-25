package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

// ResolvePath chooses the active config file path. Precedence: flag > env
// (TG_CONFIG) > defaultPath. Empty string for a tier means "this tier did not
// set a value". defaultPath must always be non-empty.
func ResolvePath(flag, defaultPath string) string {
	if flag != "" {
		return flag
	}
	if v, ok := os.LookupEnv("TG_CONFIG"); ok && v != "" {
		return v
	}
	return defaultPath
}

// ReadRawAt reads the TOML file at path into a map[string]any. A missing
// file returns an empty map and no error (caller decides whether to
// populate). Parse errors surface directly. Empty file returns empty map.
func ReadRawAt(path string) (map[string]any, error) {
	raw := map[string]any{}
	data, err := os.ReadFile(path)
	switch {
	case err == nil:
		if parseErr := toml.Unmarshal(data, &raw); parseErr != nil {
			return nil, fmt.Errorf("parse config %s: %w", path, parseErr)
		}
	case errors.Is(err, os.ErrNotExist):
		// leave raw empty; caller populates as needed
	default:
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	return raw, nil
}

// WriteRawAt serializes raw to TOML and writes path via tmp+rename with
// mode 0600. Creates parent directories as needed.
//
// The tmp file is created via os.CreateTemp in the destination dir so two
// concurrent writers do not collide on a fixed ".tmp" suffix. On any error
// after tmp creation, the deferred os.Remove unlinks the tmp file so we
// never leave stray ".tmp" droppings in the config directory.
func WriteRawAt(path string, raw map[string]any) error {
	out, err := toml.Marshal(raw)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, ".config-*.tmp")
	if err != nil {
		return fmt.Errorf("create tmp in %s: %w", dir, err)
	}
	tmpName := tmp.Name()
	// Defer cleanup unconditionally. A successful rename unlinks the tmp name
	// anyway; a failed rename or partial write leaves a file this defer removes.
	defer func() { _ = os.Remove(tmpName) }()
	if _, err := tmp.Write(out); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write tmp %s: %w", tmpName, err)
	}
	if err := tmp.Chmod(0600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("chmod tmp %s: %w", tmpName, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close tmp %s: %w", tmpName, err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("rename %s -> %s: %w", tmpName, path, err)
	}
	return nil
}

// SetDefaultAccount reads path (if present), sets or updates the
// `default_account` field, and writes the file back via tmp+rename (0600).
// Unknown keys are preserved; comments and ordering are NOT preserved (v1
// intentionally accepts this — config.toml is written far more often by this
// helper than by hand). If the file is absent, creates it with version = 1.
func SetDefaultAccount(path, name string) error {
	raw, err := ReadRawAt(path)
	if err != nil {
		return err
	}
	if _, ok := raw["version"]; !ok {
		raw["version"] = int64(1)
	}
	raw["default_account"] = name
	return WriteRawAt(path, raw)
}

// UnsetDefaultAccount reads path (if present), deletes the default_account
// key, and writes back atomically. Missing file or missing key is a
// zero-error no-op — the caller gets the same end state either way.
func UnsetDefaultAccount(path string) error {
	raw, err := ReadRawAt(path)
	if err != nil {
		return err
	}
	if _, ok := raw["default_account"]; !ok {
		return nil
	}
	delete(raw, "default_account")
	return WriteRawAt(path, raw)
}
