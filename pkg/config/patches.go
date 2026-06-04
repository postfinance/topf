// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/postfinance/topf/internal/decryption"
	"github.com/siderolabs/talos/pkg/machinery/config/configpatcher"
	"gopkg.in/yaml.v3"
)

// PatchContext are the things that can be templated in patches
type PatchContext struct {
	ClusterName       string
	ClusterEndpoint   string
	KubernetesVersion string
	TalosVersion      string
	SchematicID       string
	Data              map[string]any
	Node              *Node
	PatchesDir        string
	DecryptCache      *decryption.Cache
}

// Load loads all patches applicable for the node This includes general patches,
// role (worker/control-plane) specific patches and node specific patches in
// that order. It also returns any secrets discovered from SOPS-encrypted patch files.
func (p *PatchContext) Load() (patches []configpatcher.Patch, secrets []string, err error) {
	patches, secrets, err = p.loadFolder(filepath.Join(p.PatchesDir, "all"))
	if err != nil {
		return nil, nil, err
	}

	// patches relating to role of node, control-plane or worker
	rolePatches, roleSecrets, err := p.loadFolder(filepath.Join(p.PatchesDir, string(p.Node.Role)))
	if err != nil {
		return nil, nil, err
	}

	patches = append(patches, rolePatches...)
	secrets = append(secrets, roleSecrets...)

	// patches relating to single specific node
	nodePatches, nodeSecrets, err := p.loadFolder(filepath.Join(p.PatchesDir, "node", p.Node.Host))
	if err != nil {
		return nil, nil, err
	}

	patches = append(patches, nodePatches...)
	secrets = append(secrets, nodeSecrets...)

	return patches, secrets, nil
}

func (p *PatchContext) loadFolder(folder string) ([]configpatcher.Patch, []string, error) {
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
			return nil, nil, nil
		}

		return nil, nil, err
	}

	var (
		patches []configpatcher.Patch
		secrets []string
	)

	for _, filePath := range filePaths {
		templatedFileContent, fileSecrets, err := p.loadFile(filePath)
		if err != nil {
			return nil, nil, err
		}

		secrets = append(secrets, fileSecrets...)

		patchesInFile, err := parsePatches(templatedFileContent)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse patches in file %s: %w", filePath, err)
		}

		patches = append(patches, patchesInFile...)
	}

	return patches, secrets, nil
}

// RenderTemplate renders a Go template file with the given data.
// The template has access to the sprig function library (http://masterminds.github.io/sprig/)
// and uses "missingkey=error".
func RenderTemplate(filename string, data any) ([]byte, error) {
	//nolint:gosec // loading arbitrary template files is by design
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	tmpl, err := template.New(filepath.Base(filename)).Funcs(sprig.TxtFuncMap()).Option("missingkey=error").Parse(string(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse template %s: %w", filename, err)
	}

	var buf bytes.Buffer

	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute template %s: %w", filename, err)
	}

	return buf.Bytes(), nil
}

func (p *PatchContext) loadFile(filename string) ([]byte, []string, error) {
	var (
		content []byte
		secrets []string
		err     error
	)

	if strings.HasSuffix(filename, ".tpl") {
		content, err = RenderTemplate(filename, p)
		if err != nil {
			return nil, nil, err
		}
	} else {
		content, secrets, err = p.DecryptCache.ReadFile(filename)
		if err != nil {
			return nil, nil, err
		}
	}

	return content, secrets, nil
}

func parsePatches(data []byte) (patches []configpatcher.Patch, err error) {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	documentIndex := 0

	for {
		var doc any

		err := decoder.Decode(&doc)
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return nil, fmt.Errorf("invalid patch at document index %d: %w", documentIndex, err)
		}

		if doc == nil {
			continue // skip empty documents
		}

		data, ok := doc.(map[string]any)
		if !ok {
			if _, isArray := doc.([]any); isArray {
				return nil, fmt.Errorf("document at index %d looks like a JSON patch (array of operations), which is not supported - use strategic merge patches instead", documentIndex)
			}

			return nil, fmt.Errorf("invalid patch at document index %d: expected a YAML mapping, got %T", documentIndex, doc)
		}

		if len(data) == 0 {
			continue // skip empty documents
		}

		patchBytes, err := yaml.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal YAML at document index %d: %w", documentIndex, err)
		}

		patch, err := configpatcher.LoadPatch(patchBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to load patch at document index %d: %w", documentIndex, err)
		}

		patches = append(patches, patch)
		documentIndex++
	}

	return patches, nil
}
