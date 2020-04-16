package patch

import (
	"fmt"
	"github.com/spf13/viper"
	"hash/crc32"
	"io/ioutil"
	"os"
	"path"
	"sync"
)

// maxFileChunkSize is the maximum number of bytes we can send of a file at a time.
const maxFileChunkSize = 24576

var (
	patchInitLock sync.Once

	// File names that should be ignored when searching for patch files.
	pathsToSkip = map[string]bool{".": true, "..": true, ".DS_Store": true, ".rid": true}

	// The top of our tree of patch files.
	rootNode *directoryNode

	// Each index corresponds to a patch file. This is constructed in the order
	// that the patch tree will be traversed and makes it faster to locate a
	// patch entry when the client sends us an index in the FileStatusPacket.
	patchIndex []*fileEntry
)

// fileEntry instances contain metadata about a patch file.
type fileEntry struct {
	filename string
	// path is the fully qualified path name on the server's filesystem.
	path string

	index    uint32
	checksum uint32
	fileSize uint32
}

// directoryNode is a tree structure for holding patch data that more closely represents
// a file hierarchy and makes it easier to handle the client working dir. Patch files and
// subdirectories are represented as lists in order to make a breadth-first search easier
// and the order predictable.
type directoryNode struct {
	name       string
	path       string
	clientPath string

	patchFiles []*fileEntry
	childNodes []*directoryNode
}

// Load all of the patch files from the configured directory and store the
// metadata in package-level constants for the DataServer instance(s).
func initializePatchData() error {
	var initErr error

	patchInitLock.Do(func() {
		dir := viper.GetString("patch_server.patch_dir")

		if _, err := os.Stat(dir); os.IsNotExist(err) {
			initErr = fmt.Errorf("error loading patch files: directory does not exist: %s", dir)
			return
		}

		rootNode = &directoryNode{path: dir, clientPath: "./"}

		fmt.Println("loading patch files from", dir)
		if err := buildPatchFileTree(rootNode); err != nil {
			initErr = fmt.Errorf("error loading patch files: ", err)
		}

		buildPatchIndex(rootNode)

		if len(patchIndex) < 1 {
			initErr = fmt.Errorf("error loading patch files: at least one patch file must be present")
		}

		fmt.Println()
	})

	return initErr
}

// Build the list of patch files present in the patch directory too sync with the
// client. Files are represented in a tree, directories act as nodes (directoryNode)
// and each keeps a list of patchFiles/subdirectories.
//
// Files in the patch directory mirror the expected file structure on the client side
// and in order to tell the client which files to check the server must instruct it to
// check files relative to the game's executable.
func buildPatchFileTree(rootNode *directoryNode) error {
	directories := make([]*directoryNode, 0)
	directories = append(directories, rootNode)

	for len(directories) > 0 {
		currentNode := directories[0]
		directories = directories[1:]

		files, err := ioutil.ReadDir(currentNode.path)
		if err != nil {
			return fmt.Errorf("failed to load directory %s: %v", currentNode.path, err)
		}

		patchFiles := make([]*fileEntry, 0)
		childDirs := make([]*directoryNode, 0)

		for _, file := range files {
			filename := file.Name()
			if _, ok := pathsToSkip[filename]; ok {
				continue
			}

			if file.IsDir() {
				node := &directoryNode{
					name:       filename,
					path:       path.Join(currentNode.path, filename),
					clientPath: path.Join(currentNode.clientPath, filename),
				}

				directories = append(directories, node)
				childDirs = append(childDirs, node)
			} else {
				data, err := ioutil.ReadFile(path.Join(currentNode.path, filename))
				if err != nil {
					return err
				}

				patchFiles = append(patchFiles, &fileEntry{
					filename: filename,
					path:     path.Join(currentNode.path, filename),
					checksum: crc32.ChecksumIEEE(data),
					fileSize: uint32(file.Size()),
				})
			}
		}

		currentNode.patchFiles = patchFiles
		currentNode.childNodes = childDirs
	}

	return nil
}

// Assign a unique index to each fileEntry in the tree and use those indices to
// populate the file lookup table.
func buildPatchIndex(node *directoryNode) {
	for _, patch := range node.patchFiles {
		patchIndex = append(patchIndex, patch)
		patch.index = uint32(len(patchIndex) - 1)
	}

	for _, dir := range node.childNodes {
		buildPatchIndex(dir)
	}
}
