// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

// Package patches loads a node's Talos config patches from the patches directory.
package patches

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"

	"github.com/postfinance/topf/internal/secrets"
	"github.com/postfinance/topf/internal/tmpl"
	"github.com/postfinance/topf/internal/userfile"
	"github.com/siderolabs/talos/pkg/machinery/config/configpatcher"
	"go.yaml.in/yaml/v4"
)

var patchFilePattern = regexp.MustCompile(`.*\.ya?ml(\.tpl)?`)

// Loader loads patches from a patches directory.
type Loader struct {
	// Dir is the patches directory.
	Dir string
	// Secrets resolves SOPS and vals secrets in patch files.
	Secrets secrets.Resolver
}

// Load returns the patches of data.Node in merge order: all, <role>,
// class/<class> in listed order, node/<host>.
// Files ending in .tpl are rendered with data, then vals references are resolved.
func (l *Loader) Load(data *tmpl.Context) ([]configpatcher.Patch, error) {
	node := &data.Node
	folders := []string{"all", string(node.Role)}

	for _, c := range node.Classes {
		folder := filepath.Join("class", c)
		// unlike other folders, a missing class folder is most likely a typo
		if _, err := os.Stat(filepath.Join(l.Dir, folder)); err != nil {
			return nil, fmt.Errorf("class %q of node %s: %w", c, node.Host, err)
		}

		folders = append(folders, folder)
	}

	folders = append(folders, filepath.Join("node", node.Host))

	var patches []configpatcher.Patch

	for _, folder := range folders {
		folderPatches, err := l.loadFolder(filepath.Join(l.Dir, folder), data)
		if err != nil {
			return nil, err
		}

		patches = append(patches, folderPatches...)
	}

	return patches, nil
}

func (l *Loader) loadFolder(folder string, data *tmpl.Context) ([]configpatcher.Patch, error) {
	files, err := findFiles(folder)
	if err != nil {
		return nil, err
	}

	var patches []configpatcher.Patch

	for _, path := range files {
		content, err := userfile.Read(path, data, l.Secrets)
		if err != nil {
			return nil, err
		}

		filePatches, err := parsePatches(content)
		if err != nil {
			return nil, fmt.Errorf("failed to parse patches in file %s: %w", path, err)
		}

		patches = append(patches, filePatches...)
	}

	return patches, nil
}

// findFiles returns the patch files in folder in lexical order, following symlinks.
// A missing folder yields no files.
func findFiles(folder string) ([]string, error) {
	if _, err := os.Stat(folder); errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}

	var (
		files []string
		walk  fs.WalkDirFunc
	)

	walk = func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.Type()&fs.ModeSymlink != 0 {
			realPath, err := filepath.EvalSymlinks(path)
			if err != nil {
				return err
			}

			info, err := os.Stat(realPath)
			if err != nil {
				return err
			}

			if info.IsDir() {
				return filepath.WalkDir(realPath, walk)
			}

			if patchFilePattern.MatchString(info.Name()) {
				files = append(files, path)
			}

			return nil
		}

		if !d.IsDir() && patchFilePattern.MatchString(d.Name()) {
			files = append(files, path)
		}

		return nil
	}

	if err := filepath.WalkDir(folder, walk); err != nil {
		return nil, err
	}

	return files, nil
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
