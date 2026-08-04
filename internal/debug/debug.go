package debug

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	_ "net/http/pprof"
	"reflect"
	"strconv"
	"strings"

	"github.com/dcrodman/archon/internal/commands"
)

const serverTypeKey = "serverType"

func WithServerContext(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, serverTypeKey, name)
}

type PrintPacketParams struct {
	Writer *bufio.Writer
	// True if this command is client->server.
	ClientCommand bool
	Data          []byte
	// Cut off the command output after a certain size.
	TruncateThreshold int
}

// PrintCommand prints the contents of a command to a specified writer along with some
// inferred metadata about the command itself.
func PrintPacket(ctx context.Context, params PrintPacketParams) {
	var header commands.PCHeader
	unmarshalStruct(params.Data[:commands.PCHeaderSize], &header)

	// Set the output colors depending on the direction of the command.
	if params.ClientCommand {
		_, _ = params.Writer.WriteString("\033[32m")
	} else {
		_, _ = params.Writer.WriteString("\033[33m")
	}

	serverType := ctx.Value(serverTypeKey)

	// Write a header line for each command with some metadata.
	var headerLine strings.Builder
	headerLine.WriteString(fmt.Sprintf("[%s] %04X ", serverType, header.Type))
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

	if err := writeCommandBody(params, header.Size); err != nil {
		fmt.Printf("error writing command body: %v\n", err)
		return
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

// UnmarshalStruct populates the struct pointed to by targetStruct by reading in a
// stream of bytes and filling the values in sequential order.
//
// Note: Duplicated from internal package to avoid a cyclical dependency.
func unmarshalStruct(data []byte, targetStruct interface{}) {
	targetVal := reflect.ValueOf(targetStruct)

	if valKind := targetVal.Kind(); valKind != reflect.Ptr {
		panic("StructFromBytes(): targetStruct must be a " +
			"ptr to struct, got: " + valKind.String())
	}

	reader := bytes.NewReader(data)
	val := targetVal.Elem()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)

		var err error
		switch field.Kind() {
		case reflect.Ptr:
			err = binary.Read(reader, binary.LittleEndian, field.Interface())
		default:
			err = binary.Read(reader, binary.LittleEndian, field.Addr().Interface())
		}
		if err != nil {
			panic(err.Error())
		}
	}
}
