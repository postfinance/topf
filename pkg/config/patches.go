package config

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/siderolabs/talos/pkg/machinery/config/configpatcher"
)

// PatchContext are the things that can be templated in patches
type PatchContext struct {
	ClusterName     string
	ClusterEndpoint string
	*Node
}

// Load loads all patches applicable for the node This includes general patches,
// role (worker/control-plane) specific patches and node specific patches in
// that order
func (p *PatchContext) Load() (patches []configpatcher.Patch, err error) {
	patches, err = p.loadFolder("patches")
	if err != nil {
		return
	}

	// patches relating to role of node, control-plane or worker
	rolePatches, err := p.loadFolder(string(p.Role))
	if err != nil {
		return
	}

	patches = append(patches, rolePatches...)

	// patches relating to single specific node
	nodePatches, err := p.loadFolder(filepath.Join("node", p.Host))
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

	patches := make([]configpatcher.Patch, len(filePaths))

	for i, filePath := range filePaths {
		patch, err := p.loadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read patch %s: %w", filePath, err)
		}

		patches[i] = patch
	}

	return patches, nil
}

func (p *PatchContext) loadFile(filename string) (configpatcher.Patch, error) {
	//nolint:gosec // loading arbitrary patche files is by design
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	if strings.HasSuffix(filename, ".tpl") {
		tmpl, err := template.New("config").Parse(string(content))
		if err != nil {
			return nil, fmt.Errorf("failed to parse template for patch %s: %w", filename, err)
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, p); err != nil {
			return nil, fmt.Errorf("failed to execute template for patch %s: %w", filename, err)
		}

		content = buf.Bytes()
	}

	return configpatcher.LoadPatch(content)
}
