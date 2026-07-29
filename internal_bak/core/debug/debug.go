package debug

import (
	"bufio"
	stdbytes "bytes"
	"encoding/binary"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"reflect"
	"strconv"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"go.uber.org/zap"

	"github.com/dcrodman/archon/internal/commands"
	"github.com/dcrodman/archon/internal/core/bytes"
)

// This function starts the default pprof HTTP server that can be accessed via localhost
// to get runtime information about archon. See https://golang.org/pkg/net/http/pprof/
func StartPprofServer(logger *zap.SugaredLogger, pprofPort int) {
	listenerAddr := fmt.Sprintf("localhost:%d", pprofPort)
	logger.Infof("starting pprof server on %s", listenerAddr)

	go func() {
		if err := http.ListenAndServe(listenerAddr, nil); err != nil {
			logger.Infof("error starting pprof server: %s", err)
		}
	}()
}

// Used with Clients to attach debugging information.

type Tag string

var (
	ServerType Tag = "server_type"
)

type PrintPacketParams struct {
	Writer     *bufio.Writer
	ServerType string
	// True if this command is client->server.
	ClientCommand bool
	Data          []byte
	// Cut off the command output after a certain size.
	TruncateThreshold int
	// For known command types, read the data into each command and
	// emit it as formatted JSON.
	Interpret bool
}

// PrintCommand prints the contents of a command to a specified writer along with some
// inferred metadata about the command itself.
func PrintPacket(params PrintPacketParams) {
	var header commands.PCHeader
	bytes.StructFromBytes(params.Data[:commands.PCHeaderSize], &header)

	// Set the output colors depending on the direction of the command.
	if params.ClientCommand {
		_, _ = params.Writer.WriteString("\033[32m")
	} else {
		_, _ = params.Writer.WriteString("\033[33m")
	}

	// Write a header line for each command with some metadata.
	var headerLine strings.Builder
	headerLine.WriteString(fmt.Sprintf("[%s] %04X ", params.ServerType, header.Type))
	headerLine.WriteString(fmt.Sprintf("(%s) ", getCommandName(params.ServerType, header.Type)))
	if params.ClientCommand {
		headerLine.WriteString("| client->server ")
	} else {
		headerLine.WriteString("| server->client ")
	}
	headerLine.WriteString(fmt.Sprintf("(%d bytes)\n", header.Size))

	var err error
	if _, err = params.Writer.WriteString(headerLine.String()); err != nil {
		fmt.Printf("error writing command header: %v\n", err)
		return
	}

	// Write out the contents of the actual command.
	if params.Interpret {
		// Attempt to print out any known commands as JSON, falling back to the standard
		// format for any we don't recognize.
		err = writeInterpretedCommandBodyToFile(params, header)
	}

	if !params.Interpret || err != nil {
		if err := writeCommandBody(params, header.Size); err != nil {
			fmt.Printf("error writing command body: %v\n", err)
			return
		}
	}

	// Reset the color.
	_, _ = params.Writer.WriteString("\n\033[0m")
	params.Writer.Flush()
}

const lineLength = 16

// Takes the contents of params.Data and writes it to params.Writer. The data is written
// in two formats: a line of the raw bytes (in hexadecimal) followed by a line of those
// same bytes translated to unicode where possible.
func writeCommandBody(params PrintPacketParams, pktLen uint16) error {
	var (
		totalLines = pktLen / lineLength
		spillover  = pktLen % lineLength
	)
	for i := range totalLines {
		offset := lineLength * i

		if params.TruncateThreshold > 0 && int(offset) > params.TruncateThreshold {
			_, _ = params.Writer.WriteString("...(truncated)...\n")
			break
		}
		line := buildSingleLine(params.Data[offset:offset+lineLength], lineLength, offset)
		if _, err := params.Writer.WriteString(line); err != nil {
			return err
		}
	}
	if spillover > 0 {
		line := buildSingleLine(params.Data[pktLen-spillover:], int(spillover), totalLines*lineLength)
		if _, err := params.Writer.WriteString(line); err != nil {
			return err
		}
	}
	return nil
}

// Build one line of formatted command data.
func buildSingleLine(data []uint8, length int, offset uint16) string {
	var line strings.Builder

	line.WriteString(fmt.Sprintf("(%04X) ", offset))

	for i, j := 0, 0; i < length; i++ {
		if j == 8 {
			// Visual aid - spacing between groups of 8 bytes.
			j = 0
		}
		line.WriteString(fmt.Sprintf("%02x ", data[i]))
		j++
	}

	// Fill in rest of the line gap if we don't have enough bytes.
	for i := length; i < lineLength; i++ {
		if i == 8 {
			line.WriteString("  ")
		}
		line.WriteString("   ")
	}
	line.WriteString("    ")

	// Display the print characters as-is, others as periods.
	for i := 0; i < length; i++ {
		c := data[i]

		if strconv.IsPrint(rune(c)) {
			line.WriteString(string(data[i]))
		} else {
			line.WriteString(".")
		}
	}

	line.WriteString("\n")
	return line.String()
}

func writeInterpretedCommandBodyToFile(params PrintPacketParams, header commands.PCHeader) error {
	newCommand := getCommand(params.ServerType, params.ClientCommand, header.Type)
	if !newCommand.IsValid() {
		_, _ = params.Writer.WriteString("(cannot interpret - unrecognized command type)\n")
		return fmt.Errorf("unrecognized command type: %02X", header.Type)
	}

	command := newCommand.Elem()
	reader := stdbytes.NewReader(params.Data)

	var err error
	for i := 0; i < command.NumField(); i++ {
		field := command.Field(i)
		switch field.Kind() {
		case reflect.Ptr:
			err = binary.Read(reader, binary.LittleEndian, field.Interface())
		default:
			err = binary.Read(reader, binary.LittleEndian, field.Addr().Interface())
		}
		if err != nil {
			err = fmt.Errorf("error constructing field %s: %w", field.String(), err)
		}
	}

	spew.Config.Indent = "\t"
	_, _ = params.Writer.WriteString(spew.Sdump(command.Interface()))
	if err != nil {
		_, _ = params.Writer.WriteString("WARNING: PARTIAL RESULT\n")
		_, _ = params.Writer.WriteString("(make sure the command type is correctly mapped)\n")
		_, _ = params.Writer.WriteString(err.Error() + "\n")
	}
	return nil
}
