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
package patch_server

import (
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"io/ioutil"
	"libarchon/encryption"
	"libarchon/logger"
	"libarchon/server"
	"libarchon/util"
	"net"
	"os"
	"runtime/debug"
	"strings"
	"sync"
)

var log *logger.Logger
var patchConnections *server.ConnectionList = server.NewClientList()

const MaxChunkSize = 24576

type pktHandler func(p *PatchClient) error

// Struct for holding client-specific data.
type PatchClient struct {
	conn        *net.TCPConn
	ipAddr      string
	port        string
	clientCrypt *encryption.PSOCrypt
	serverCrypt *encryption.PSOCrypt
	recvData    []byte
	recvSize    int
	packetSize  uint16
	updateList  []*PatchEntry
}

func (pc PatchClient) Connection() *net.TCPConn { return pc.conn }
func (pc PatchClient) IPAddr() string           { return pc.ipAddr }
func (pc PatchClient) Decrypt(data []byte, size uint32) {
	pc.clientCrypt.Decrypt(data, size)
}

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

// File names that should be ignored when searching for patch files. This
// could also be an array but the map makes it quicker to compare.
var SkipPaths = map[string]byte{".": 1, "..": 1, ".DS_Store": 1, ".rid": 1}

// Each index corresponds to a patch file. This is constructed in the order
// that the patch tree will be traversed and makes it faster to locate a
// patch entry when the client sends us an index in the FileStatusPacket.
var patchTree PatchDir
var patchIndex []*PatchEntry

// Create and initialize a new struct to hold client information.
func newClient(conn *net.TCPConn) (*PatchClient, error) {
	client := new(PatchClient)
	client.conn = conn
	addr := strings.Split(conn.RemoteAddr().String(), ":")
	client.ipAddr = addr[0]
	client.port = addr[1]

	client.clientCrypt = encryption.NewCrypt()
	client.serverCrypt = encryption.NewCrypt()
	client.clientCrypt.CreateKeys()
	client.serverCrypt.CreateKeys()
	// The client doesn't send this server very big packets. We can
	// save some space and keep the buffer small, growing later if needed.
	client.recvData = make([]byte, 128)

	var err error = nil
	if SendWelcome(client) != 0 {
		err = errors.New("Error sending welcome packet to: " + client.ipAddr)
		client = nil
	}
	return client, err
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
	util.StructFromBytes(client.recvData[:], &fileStatus)

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
				log.Error(err.Error(), logger.LogPriorityCritical)
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
	util.StructFromBytes(client.recvData[:PCHeaderSize], &pktHeader)

	if GetConfig().DebugMode {
		fmt.Printf("Got %v bytes from client:\n", pktHeader.Size)
		util.PrintPayload(client.recvData, int(pktHeader.Size))
		fmt.Println()
	}
	var err error = nil
	switch pktHeader.Type {
	case WelcomeType:
		SendWelcomeAck(client)
	case LoginType:
		cfg := GetConfig()
		if SendWelcomeMessage(client) == 0 {
			SendRedirect(client, cfg.RedirectPort(), cfg.HostnameBytes())
		}
	default:
		msg := fmt.Sprintf("Received unknown packet %2x from %s", pktHeader.Type, client.ipAddr)
		log.Info(msg, logger.LogPriorityMedium)
	}
	return err
}

// Handle a packet sent to the DATA server.
func processDataPacket(client *PatchClient) error {
	var pktHeader PCPktHeader
	util.StructFromBytes(client.recvData[:PCHeaderSize], &pktHeader)

	if GetConfig().DebugMode {
		fmt.Printf("Got %v bytes from client:\n", pktHeader.Size)
		util.PrintPayload(client.recvData, int(pktHeader.Size))
		fmt.Println()
	}
	var err error = nil
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
		msg := fmt.Sprintf("Received unknown packet %02x from %s", pktHeader.Type, client.ipAddr)
		log.Info(msg, logger.LogPriorityMedium)
	}
	return err
}

// Handle communication with a particular client until the connection is
// closed or an error is encountered.
func handleClient(client *PatchClient, desc string, handler pktHandler) {
	defer func() {
		if err := recover(); err != nil {
			errMsg := fmt.Sprintf("Error in client communication: %s: %s\n%s\n",
				client.ipAddr, err, debug.Stack())
			log.Error(errMsg, logger.LogPriorityCritical)
		}
		client.conn.Close()
		patchConnections.RemoveClient(client)
		log.Info("Disconnected "+desc+" client "+client.ipAddr, logger.LogPriorityMedium)
	}()

	log.Info("Accepted "+desc+" connection from "+client.ipAddr, logger.LogPriorityMedium)
	// We're running inside a goroutine at this point, so we can block on this connection
	// and not interfere with any other clients.
	for {
		// Wait for the packet header.
		for client.recvSize < PCHeaderSize {
			bytes, err := client.conn.Read(client.recvData[client.recvSize:PCHeaderSize])
			if bytes == 0 || err == io.EOF {
				// The client disconnected, we're done.
				client.conn.Close()
				return
			} else if err != nil {
				// Socket error, nothing we can do now
				log.Warn("Socket Error ("+client.ipAddr+") "+err.Error(),
					logger.LogPriorityMedium)
				return
			}
			client.recvSize += bytes

			if client.recvSize >= PCHeaderSize {
				// We have our header; decrypt it.
				client.clientCrypt.Decrypt(client.recvData[:PCHeaderSize], PCHeaderSize)
				client.packetSize, err = util.GetPacketSize(client.recvData[:2])
				if err != nil {
					// Something is seriously wrong if this causes an error. Bail.
					panic(err.Error())
				}
				// PSO likes to occasionally send us packets that are longer than their
				// declared size. Adjust the expected length just in case in order to
				// avoid leaving stray bytes in the buffer.
				for client.packetSize%PCHeaderSize != 0 {
					client.packetSize++
				}
			}
		}
		pktSize := int(client.packetSize)
		// Grow the client's receive buffer if they send us a packet bigger
		// than its current capacity.
		if pktSize > cap(client.recvData) {
			newSize := pktSize + len(client.recvData)
			newBuf := make([]byte, newSize)
			copy(newBuf, client.recvData)
			client.recvData = newBuf
			msg := fmt.Sprintf("Reallocated buffer to %v bytes", newSize)
			log.Info(msg, logger.LogPriorityLow)
		}

		// Read in the rest of the packet.
		for client.recvSize < pktSize {
			remaining := pktSize - client.recvSize
			bytes, err := client.conn.Read(
				client.recvData[client.recvSize : client.recvSize+remaining])
			if err != nil {
				log.Warn("Socket Error ("+client.ipAddr+") "+err.Error(),
					logger.LogPriorityMedium)
				return
			}
			client.recvSize += bytes
		}

		// We have the whole thing; decrypt the rest of it if needed and pass it along.
		if client.packetSize > PCHeaderSize {
			client.clientCrypt.Decrypt(
				client.recvData[PCHeaderSize:client.packetSize],
				uint32(client.packetSize-PCHeaderSize))
		}
		if err := handler(client); err != nil {
			log.Info(err.Error(), logger.LogPriorityLow)
			break
		}

		// Extra bytes left in the buffer will just be ignored.
		client.recvSize = 0
		client.packetSize = 0
	}
}

// Creates the socket and starts listening for connections on the specified
// port, spawning off goroutines to handle communications for each client.
func startWorker(wg *sync.WaitGroup, id, port string, handler pktHandler) {
	cfg := GetConfig()
	socket, err := server.OpenSocket(cfg.Hostname, port)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
	fmt.Printf("Waiting for %s connections on %s:%s...\n", id, cfg.Hostname, port)
	for {
		// Poll until we can accept more clients.
		for patchConnections.Count() < cfg.MaxConnections {
			connection, err := socket.AcceptTCP()
			if err != nil {
				log.Error("Failed to accept connection: "+err.Error(), logger.LogPriorityHigh)
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
		if _, ignore := SkipPaths[filename]; ignore {
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
			patch := new(PatchEntry)
			patch.filename = filename
			patch.relativePath = path + "/" + filename
			patch.pathDirs = dirs
			patch.fileSize = uint32(file.Size())
			patch.checksum = crc32.ChecksumIEEE(data)

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
	fmt.Println("Initializing Archon PATCH and DATA servers...")

	// Initialize our config singleton from one of two expected file locations.
	config := GetConfig()
	fmt.Printf("Loading config file %v...", patchConfigFile)
	err := config.InitFromFile(patchConfigFile)
	if err != nil {
		os.Chdir(ServerConfigDir)
		fmt.Printf("Failed.\nLoading config from %v...", ServerConfigDir+"/"+patchConfigFile)
		err = config.InitFromFile(patchConfigFile)
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

	// Initialize the logger.
	log = logger.New(config.logWriter, config.LogLevel)
	log.Info("Server Initialized", logger.LogPriorityCritical)

	if config.DebugMode {
		go server.CreateStackTraceServer("127.0.0.1:8080", "/")
	}

	// Create a WaitGroup so that main won't exit until the server threads have exited.
	var wg sync.WaitGroup
	wg.Add(2)
	go startWorker(&wg, "PATCH", config.PatchPort, processPatchPacket)
	go startWorker(&wg, "DATA", config.DataPort, processDataPacket)
	wg.Wait()
}
