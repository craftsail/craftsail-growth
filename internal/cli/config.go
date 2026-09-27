// SPDX-License-Identifier: AGPL-3.0-or-later

package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ensureConfig creates path from the sibling *.example.toml when path is
// missing. It never overwrites an existing file.
func ensureConfig(path string) (bool, error) {
	if _, err := os.Stat(path); err == nil {
		return false, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	ext := filepath.Ext(path)
	example := strings.TrimSuffix(path, ext) + ".example" + ext
	src, err := os.ReadFile(example)
	if err != nil {
		return false, fmt.Errorf("config %s not found, and no example at %s", path, example)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}
	if err := os.WriteFile(path, src, 0o600); err != nil {
		return false, err
	}
	return true, nil
}
