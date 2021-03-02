package patch

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/dcrodman/archon"
)

var paths = [][2]string{
	{"GameGuard", "temp.txt"},
	{"Required", "good.txt"},
}

func Test_BuildPatchFileTree(t *testing.T) {
	// prevent panics on logging
	archon.Log = &logrus.Logger{}

	dir, cleanup := generateTestFiles(t)
	t.Cleanup(cleanup)
	rootNode = &directoryNode{path: dir, clientPath: "./"}
	if err := buildPatchFileTree(rootNode); err != nil {
		t.Fatalf("buildPathFileTree got %v, want nil", err)
	}
	var nodes []*directoryNode
	nodes = append(nodes, rootNode)
	for len(nodes) > 0 {
		node := nodes[0]
		nodes = nodes[1:]
		if node.path == "GameGuard" {
			t.Fatal("GameGuard present in directory nodes")
		}
		if node.path == "Required" {
			if len(node.patchFiles) != 1 {
				t.Fatalf("buildPatchFileTree() found more than one file for the good path, expected 1")
			}
			if node.patchFiles[0].filename != "good.txt" {
				t.Fatalf("buildPatchFileTree() child file got %s, want good.txt", node.patchFiles[0].filename)
			}
		}
		nodes = append(nodes, node.childNodes...)
	}
}

func generateTestFiles(t *testing.T) (string, func()) {
	// TODO: t.TempDir in Go 1.15/1.16
	tmp := os.TempDir()

	for _, path := range paths {
		dir := filepath.Join(tmp, path[0])
		if err := os.Mkdir(dir, os.ModePerm); err != nil {
			t.Fatalf("failed to create testing directory: %v", err)
		}

		file := filepath.Join(dir, path[1])
		if err := os.Mkdir(file, os.ModePerm); err != nil {
			t.Fatalf("failed to create testing directory: %v", err)
		}
	}

	return tmp, func() {
		os.RemoveAll(tmp)
	}
}
