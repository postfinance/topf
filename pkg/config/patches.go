// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package config

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/postfinance/topf/pkg/sops"
	"github.com/siderolabs/talos/pkg/machinery/config/configpatcher"
	"gopkg.in/yaml.v3"
)

// PatchContext are the things that can be templated in patches
type PatchContext struct {
	ClusterName       string
	ClusterEndpoint   string
	KubernetesVersion string
	Data              map[string]any
	Node              *Node
	ConfigDir         string
}

// Load loads all patches applicable for the node This includes general patches,
// role (worker/control-plane) specific patches and node specific patches in
// that order
func (p *PatchContext) Load() (patches []configpatcher.Patch, err error) {
	// warn about legacy patches/ directory
	oldDir := filepath.Join(p.ConfigDir, "patches")
	if info, statErr := os.Stat(oldDir); statErr == nil && info.IsDir() {
		slog.Warn("legacy patches/ directory found, rename it to all/", "path", oldDir)
	}

	patches, err = p.loadFolder(filepath.Join(p.ConfigDir, "all"))
	if err != nil {
		return
	}

	// patches relating to role of node, control-plane or worker
	rolePatches, err := p.loadFolder(filepath.Join(p.ConfigDir, string(p.Node.Role)))
	if err != nil {
		return
	}

	patches = append(patches, rolePatches...)

	// patches relating to single specific node
	nodePatches, err := p.loadFolder(filepath.Join(p.ConfigDir, "node", p.Node.Host))
	if err != nil {
		return
	}

	patches = append(patches, nodePatches...)

	return
}

func (p *PatchContext) loadFolder(folder string) ([]configpatcher.Patch, error) {
	var filePaths []string

	pattern := regexp.MustCompile(`.*\.ya?ml(\.tpl)?`)

	var walkFunc fs.WalkDirFunc

	walkFunc = func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// handle symlinks
		if d.Type()&os.ModeSymlink != 0 {
			realPath, err := filepath.EvalSymlinks(path)
			if err != nil {
				return err
			}

			info, err := os.Stat(realPath)
			if err != nil {
				return err
			}

			// If symlink points to a directory, walk it recursively
			if info.IsDir() {
				return filepath.WalkDir(realPath, walkFunc)
			}

			// If symlink points to a file, check the pattern
			if pattern.MatchString(info.Name()) {
				filePaths = append(filePaths, path)
			}

			return nil
		}

		if !d.IsDir() && pattern.MatchString(d.Name()) {
			filePaths = append(filePaths, path)
		}

		return nil
	}

	err := filepath.WalkDir(folder, walkFunc)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, err
	}

	patches := make([]configpatcher.Patch, 0, len(filePaths))

	for _, filePath := range filePaths {
		patch, err := p.loadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read patch %s: %w", filePath, err)
		}

		// Skip nil patches (empty files)
		if patch != nil {
			patches = append(patches, patch)
		}
	}

	return patches, nil
}

func (p *PatchContext) loadFile(filename string) (configpatcher.Patch, error) {
	var (
		content []byte
		err     error
	)

	//nolint:nestif // complexity due to template vs SOPS handling
	if strings.HasSuffix(filename, ".tpl") {
		// Template files: read without SOPS decryption
		//nolint:gosec // loading arbitrary patch files is by design
		content, err = os.ReadFile(filename)
		if err != nil {
			return nil, err
		}

		tmpl, err := template.New("config").Option("missingkey=error").Parse(string(content))
		if err != nil {
			return nil, fmt.Errorf("failed to parse template for patch %s: %w", filename, err)
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, p); err != nil {
			return nil, fmt.Errorf("failed to execute template for patch %s: %w", filename, err)
		}

		content = buf.Bytes()
	} else {
		// Non-template files: use SOPS auto-detection
		content, err = sops.ReadFileWithSOPS(filename)
		if err != nil {
			return nil, err
		}

		if content == nil {
			return nil, fmt.Errorf("patch file not found: %s", filename)
		}
	}

	// Check if patch is empty (only whitespace or comments)
	if isEmpty(content) {
		// Skip empty patches gracefully - returning nil patch is intentional, not an error
		//nolint:nilnil // empty patch is valid, not an error condition
		return nil, nil
	}

	// early error with JSON patches, as TOPF (and talos v1.12+) are not supporting those.
	patch, err := configpatcher.LoadPatch(content)
	if _, isJSONPatch := patch.(jsonpatch.Patch); isJSONPatch {
		return nil, errors.New("TOPF doesn't not support JSON patches")
	}

	return patch, err
}

// isEmpty checks if content is effectively empty by attempting to unmarshal it
func isEmpty(content []byte) bool {
	// Trim whitespace first
	trimmed := strings.TrimSpace(string(content))
	if trimmed == "" {
		return true
	}

	// Try to unmarshal as generic YAML
	var data any
	if err := yaml.Unmarshal(content, &data); err != nil {
		// If it doesn't unmarshal, it's not valid YAML, so not empty
		return false
	}

	// Check if unmarshaled data is nil or empty
	if data == nil {
		return true
	}

	// Check if it's an empty map
	if m, ok := data.(map[string]any); ok && len(m) == 0 {
		return true
	}

	// Check if it's an empty slice
	if s, ok := data.([]any); ok && len(s) == 0 {
		return true
	}

	// Has actual data
	return false
}
