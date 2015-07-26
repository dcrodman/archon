/*
* Archon Patch Server
* Copyright (C) 2014 Andrew Rodman
*
* This program is free software: you can redistribute it and/or modify
* it under the terms of the GNU General Public License as published by
* the Free Software Foundation, either version 3 of the License, or
* (at your option) any later version.
*
* This program is distributed in the hope that it will be useful,
* but WITHOUT ANY WARRANTY; without even the implied warranty of
* MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
* GNU General Public License for more details.
*
* You should have received a copy of the GNU General Public License
* along with this program.  If not, see <http://www.gnu.org/licenses/>.
* ---------------------------------------------------------------------
* The PATCH and DATA server logic. Both are included here since they're
* neither are particularly complicated.
 */
package patch

import (
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"io/ioutil"
	"libarchon/logger"
	"libarchon/server"
	"libarchon/util"
	"net"
	"net/http"
	"os"
	"runtime/debug"
	"runtime/pprof"
	"strings"
	"sync"
)

var (
	log              *logger.ServerLogger
	patchConnections *server.ConnectionList = server.NewClientList()

	// File names that should be ignored when searching for patch files.
	SkipPaths = []string{".", "..", ".DS_Store", ".rid"}

	// Each index corresponds to a patch file. This is constructed in the order
	// that the patch tree will be traversed and makes it faster to locate a
	// patch entry when the client sends us an index in the FileStatusPacket.
	patchTree  PatchDir
	patchIndex []*PatchEntry
)

const MaxChunkSize = 24576

type pktHandler func(p *PatchClient) error

// Struct for holding client-specific data.
type PatchClient struct {
	c          *server.PSOClient
	updateList []*PatchEntry
}

func (pc *PatchClient) IPAddr() string        { return pc.c.IPAddr() }
func (pc *PatchClient) Client() server.Client { return pc.c }
func (pc *PatchClient) Data() []byte          { return pc.c.Data() }

// Data for one patch file.
type PatchEntry struct {
	filename string
	// Path relative to the patch dir for convenience.
	relativePath string
	pathDirs     []string
	index        uint32
	checksum     uint32
	fileSize     uint32
}

// Basic tree structure for holding patch data that more closely represents
// a file hierarchy and makes it easier to handle the client working dir.
// Patch files and subdirectories are represented as lists in order to make
// a breadth-first search easier and the order predictable.
type PatchDir struct {
	dirname string
	patches []*PatchEntry
	subdirs []*PatchDir
}

// Create and initialize a new struct to hold client information.
func newClient(conn *net.TCPConn) (*PatchClient, error) {
	pc := new(PatchClient)
	pc.c = server.NewPSOClient(conn, PCHeaderSize)

	var err error
	if SendWelcome(pc) != 0 {
		err = errors.New("Error sending welcome packet to: " + pc.IPAddr())
		pc = nil
	}
	return pc, err
}

// Traverse the patch tree depth-first and send the check file requests.
func sendFileList(client *PatchClient, node *PatchDir) {
	// Step into the next directory.
	SendChangeDir(client, node.dirname)
	for _, subdir := range node.subdirs {
		sendFileList(client, subdir)
		// Move them back up each time we leave a directory.
		SendDirAbove(client)
	}
	for _, patch := range node.patches {
		SendCheckFile(client, patch.index, patch.filename)
	}
}

// The client sent us a checksum for one of the patch files. Compare it
// to what we have and add it to the list of files to update if there
// is any discrepancy.
func handleFileStatus(client *PatchClient) {
	var fileStatus FileStatusPacket
	util.StructFromBytes(client.Data(), &fileStatus)

	patch := patchIndex[fileStatus.PatchId]
	if fileStatus.Checksum != patch.checksum || fileStatus.FileSize != patch.fileSize {
		client.updateList = append(client.updateList, patch)
	}
}

// The client finished sending all of the file check packets. If they have
// any files that need updating, now's the time to do it.
func updateClientFiles(client *PatchClient) error {
	var numFiles, totalSize uint32 = 0, 0
	for _, patch := range client.updateList {
		numFiles++
		totalSize += patch.fileSize
	}

	// If we have any files to send, do it now.
	if numFiles > 0 {
		SendUpdateFiles(client, numFiles, totalSize)
		SendChangeDir(client, ".")
		chunkBuf := make([]byte, MaxChunkSize)

		for _, patch := range client.updateList {
			// Descend into the correct directory if needed.
			ascendCtr := 0
			for i := 1; i < len(patch.pathDirs); i++ {
				ascendCtr++
				SendChangeDir(client, patch.pathDirs[i])
			}
			SendFileHeader(client, patch)

			// Divide the file into chunks and send each one.
			chunks := int((patch.fileSize / MaxChunkSize) + 1)
			file, err := os.Open(patch.relativePath)
			if err != nil {
				// Critical since this is most likely a filesystem error.
				log.Error(err.Error())
				return err
			}
			for i := 0; i < chunks; i++ {
				bytes, err := file.ReadAt(chunkBuf, int64(MaxChunkSize*i))
				if err != nil && err != io.EOF {
					return err
				}
				chksm := crc32.ChecksumIEEE(chunkBuf)
				SendFileChunk(client, uint32(i), chksm, uint32(bytes), chunkBuf)
			}

			SendFileComplete(client)
			// Change back to the top level directory.
			for ascendCtr > 0 {
				ascendCtr--
				SendDirAbove(client)
			}
		}
	}
	SendUpdateComplete(client)
	return nil
}

// Handle a packet sent to the PATCH server.
func processPatchPacket(client *PatchClient) error {
	var pktHeader PCPktHeader
	util.StructFromBytes(client.Data()[:PCHeaderSize], &pktHeader)

	if config.DebugMode {
		fmt.Printf("PATCH: Got %v bytes from client:\n", pktHeader.Size)
		util.PrintPayload(client.Data(), int(pktHeader.Size))
		fmt.Println()
	}
	var err error
	switch pktHeader.Type {
	case WelcomeType:
		SendWelcomeAck(client)
	case LoginType:
		if SendWelcomeMessage(client) == 0 {
			SendRedirect(client, config.RedirectPort(), config.HostnameBytes())
		}
	default:
		log.Info("Received unknown packet %2x from %s", pktHeader.Type, client.IPAddr())
	}
	return err
}

// Handle a packet sent to the DATA server.
func processDataPacket(client *PatchClient) error {
	var pktHeader PCPktHeader
	util.StructFromBytes(client.Data()[:PCHeaderSize], &pktHeader)

	if config.DebugMode {
		fmt.Printf("DATA: Got %v bytes from client:\n", pktHeader.Size)
		util.PrintPayload(client.Data(), int(pktHeader.Size))
		fmt.Println()
	}
	var err error
	switch pktHeader.Type {
	case WelcomeType:
		SendWelcomeAck(client)
	case LoginType:
		SendDataAck(client)
		sendFileList(client, &patchTree)
		SendFileListDone(client)
	case FileStatusType:
		handleFileStatus(client)
	case ClientListDoneType:
		err = updateClientFiles(client)
	default:
		log.Info("Received unknown packet %02x from %s", pktHeader.Type, client.IPAddr())
	}
	return err
}

// Handle communication with a particular client until the connection is
// closed or an error is encountered.
func handleClient(client *PatchClient, desc string, handler pktHandler) {
	defer func() {
		if err := recover(); err != nil {
			log.Error("Error in client communication: %s: %s\n%s\n",
				client.IPAddr(), err, debug.Stack())
		}
		client.c.Close()
		patchConnections.RemoveClient(client)
		log.Info("Disconnected %s client %s", desc, client.IPAddr())
	}()

	log.Info("Accepted %s connection from %s", desc, client.IPAddr())
	for {
		err := client.c.Process()
		if err == io.EOF {
			break
		} else if err != nil {
			// Error communicating with the client.
			log.Warn(err.Error())
			break
		}

		if err = handler(client); err != nil {
			log.Warn(err.Error())
			break
		}
	}
}

// Creates the socket and starts listening for connections on the specified
// port, spawning off goroutines to handle communications for each client.
func startWorker(wg *sync.WaitGroup, id, port string, handler pktHandler) {
	socket, err := server.OpenSocket(config.Hostname, port)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
	log.Important("Waiting for %s connections on %s:%s...", id, config.Hostname, port)
	for {
		// Poll until we can accept more clients.
		for patchConnections.Count() < config.MaxConnections {
			connection, err := socket.AcceptTCP()
			if err != nil {
				log.Warn("Failed to accept connection: %v", err.Error())
				continue
			}
			client, err := newClient(connection)
			if err != nil {
				continue
			}
			patchConnections.AddClient(client)
			go handleClient(client, id, handler)
		}
	}
	wg.Done()
}

// Recursively build the list of patch files present in the patch directory
// to sync with the client. Files are represented in a tree, directories act
// as nodes (PatchDir) and each keeps a list of patches/subdirectories.
func loadPatches(node *PatchDir, path string) error {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		fmt.Printf("Couldn't parse %s\n", path)
		return err
	}
	dirs := strings.Split(path, "/")
	node.dirname = dirs[len(dirs)-1]

	for _, file := range files {
		filename := file.Name()
		skip := false
		for _, path := range SkipPaths {
			if filename == path {
				skip = true
				break
			}
		}

		if skip {
			continue
		} else if file.IsDir() {
			subdir := new(PatchDir)
			node.subdirs = append(node.subdirs, subdir)
			loadPatches(subdir, path+"/"+filename)
		} else {
			data, err := ioutil.ReadFile(path + "/" + filename)
			if err != nil {
				return err
			}
			patch := &PatchEntry{
				filename:     filename,
				relativePath: path + "/" + filename,
				pathDirs:     dirs,
				fileSize:     uint32(file.Size()),
				checksum:     crc32.ChecksumIEEE(data),
			}

			node.patches = append(node.patches, patch)
			fmt.Printf("%s (%d bytes, checksum: %v)\n",
				path+"/"+filename, patch.fileSize, patch.checksum)
		}
	}
	return nil
}

// Build the patch index, performing a depth-first search and mapping
// each patch entry to an array so that they're quickly indexable when
// we need to look up the patch data.
func buildPatchIndex(node *PatchDir) {
	for _, dir := range node.subdirs {
		buildPatchIndex(dir)
	}
	for _, patch := range node.patches {
		patchIndex = append(patchIndex, patch)
		patch.index = uint32(len(patchIndex) - 1)
	}
}

func StartServer() {
	// Initialize our config singleton from one of two expected file locations.
	fmt.Printf("Loading config file %v...", PatchConfigFile)
	err := config.InitFromFile(PatchConfigFile)
	if err != nil {
		os.Chdir(ServerConfigDir)
		fmt.Printf("Failed.\nLoading config from %v...", ServerConfigDir+"/"+PatchConfigFile)
		err = config.InitFromFile(PatchConfigFile)
		if err != nil {
			fmt.Println("Failed.\nPlease check that one of these files exists and restart the server.")
			fmt.Printf("%s\n", err.Error())
			os.Exit(1)
		}
	}
	fmt.Printf("Done.\n\n--Configuration Parameters--\n%v\n\n", config.String())

	// Construct our patch tree from the specified directory.
	fmt.Printf("Loading patches from %s...\n", config.PatchDir)
	os.Chdir(config.PatchDir)
	if err := loadPatches(&patchTree, "."); err != nil {
		fmt.Printf("Failed to load patches: %s\n", err.Error())
		os.Exit(1)
	}
	fmt.Println()
	buildPatchIndex(&patchTree)
	if len(patchIndex) < 1 {
		fmt.Println("Failed: At least one patch file must be present.")
		os.Exit(1)
	}

	// If we're in debug mode, spawn off an HTTP server that, when hit, dumps
	// pprof output containing the stack traces of all running goroutines.
	if config.DebugMode {
		http.HandleFunc("/", func(resp http.ResponseWriter, req *http.Request) {
			pprof.Lookup("goroutine").WriteTo(resp, 1)
		})
		go http.ListenAndServe("127.0.0.1:8080", nil)
	}

	// Initialize the logger.
	log, err = logger.New(config.Logfile, config.LogLevel)
	if err != nil {
		fmt.Println("ERROR: Failed to open log file " + config.Logfile)
		os.Exit(1)
	}
	log.Important("Server Initialized")

	// Create a WaitGroup so that main won't exit until the server threads have exited.
	var wg sync.WaitGroup
	wg.Add(2)
	go startWorker(&wg, "PATCH", config.PatchPort, processPatchPacket)
	go startWorker(&wg, "DATA", config.DataPort, processDataPacket)
	wg.Wait()
}
