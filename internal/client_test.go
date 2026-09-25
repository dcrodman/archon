package internal

import (
	"context"
	"encoding/binary"
	"io"
	"net"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/dcrodman/archon/internal/commands"
	"github.com/dcrodman/archon/internal/encryption"
)

var (
	testPacket = &commands.CharacterSelectionAck{
		Header: commands.BBHeader{
			Size: 0x10,
			Type: commands.CharacterSelectionAckType,
		},
		Slot: 1,
		Flag: 1,
	}
	testPacketBytes, _ = MarshalStruct(testPacket)
)

func newTestListener(t *testing.T) (*net.TCPListener, *net.TCPAddr) {
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatalf("error initializing test listener: %v", err)
	}
	return listener, listener.Addr().(*net.TCPAddr)
}

func newTestConnection(t *testing.T, addr *net.TCPAddr) *net.TCPConn {
	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		t.Fatalf("error initializing test connection: %v", err)
	}
	return conn
}

func TestClient_Read(t *testing.T) {
	serverListener, addr := newTestListener(t)
	// Connect to the server as if from a PSO client.
	conn := newTestConnection(t, addr)

	// Handle the connection on the server side and drop it into a Client.
	clientConn, err := serverListener.AcceptTCP()
	if err != nil {
		t.Fatalf("error initializing client connection: %s", err)
	}
	client := NewClient(clientConn)

	// Write a packet from the "PSO client" side.
	if _, err = conn.Write(testPacketBytes); err != nil {
		t.Fatalf("error writing to test connection: %s", err)
	}

	// Read the packet via Client and make sure it's sane.
	buf := make([]byte, 16)
	bytesRead, err := client.Read(buf)
	if err != nil {
		t.Fatalf("Read() returned an unexpected error: %s", err)
	} else if bytesRead != len(testPacketBytes) {
		t.Fatalf("expected to have read %d bytes, got %d", bytesRead, len(testPacketBytes))
	}

	if diff := cmp.Diff(testPacketBytes, buf); diff != "" {
		t.Fatalf("Read() result did not match expected; diff:\n%s", diff)
	}
}

func TestClient_SendRaw(t *testing.T) {
	serverListener, addr := newTestListener(t)
	// Connect to the server as if from a PSO client.
	conn := newTestConnection(t, addr)

	// Handle the connection on the server side and drop it into a Client.
	clientConn, err := serverListener.AcceptTCP()
	if err != nil {
		t.Fatalf("error initializing client connection: %s", err)
	}
	client := NewClient(clientConn)

	// Send bytes from the client and make sure they weren't altered.
	if err := client.SendRaw(context.TODO(), testPacket); err != nil {
		t.Fatalf("SendRaw() returned an unexpected error: %s", err)
	}
	client.Close()

	buf := make([]byte, 16)
	if _, err := conn.Read(buf); err != nil {
		t.Fatalf("error reading from test connection: %s", err)
	}

	if diff := cmp.Diff(testPacketBytes, buf); diff != "" {
		t.Fatalf("bytes read from test connection did not match expected; diff:\n%s", diff)
	}
}

func TestClient_Send(t *testing.T) {
	// withSize returns a copy of data with the header size set to size and zero padding
	// appended out to length bytes.
	withSize := func(data []byte, size uint16, length int) []byte {
		out := make([]byte, length)
		copy(out, data)
		binary.LittleEndian.PutUint16(out, size)
		return out
	}

	noSizePacket := &commands.CharacterSelectionAck{
		Header: commands.BBHeader{Type: commands.CharacterSelectionAckType},
		Slot:   1,
		Flag:   1,
	}
	noSizePacketBytes, _ := MarshalStruct(noSizePacket)

	// 12 bytes, like a 6x23 broadcast: the header size should stay 0x0C, but 16 bytes
	// should be sent to satisfy the block size.
	shortPacket := &commands.Broadcast{
		Header: commands.BBHeader{Type: commands.BroadcastType},
		Data:   []byte{0x23, 0x01, 0x01, 0x00},
	}
	shortPacketBytes, _ := MarshalStruct(shortPacket)

	// 18 bytes: the header size should be rounded up to 0x14 and 24 bytes should be sent.
	unalignedPacket := &commands.Broadcast{
		Header: commands.BBHeader{Type: commands.BroadcastType},
		Data:   []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A},
	}
	unalignedPacketBytes, _ := MarshalStruct(unalignedPacket)

	tests := []struct {
		name   string
		packet any
		want   []byte
	}{
		{
			name:   "packet is the correct size",
			packet: testPacket,
			want:   testPacketBytes,
		},
		{
			name:   "packet size is not set",
			packet: noSizePacket,
			want:   withSize(noSizePacketBytes, 0x10, 0x10),
		},
		{
			name:   "packet size is a multiple of 4 but not 8",
			packet: shortPacket,
			want:   withSize(shortPacketBytes, 0x0C, 0x10),
		},
		{
			name:   "packet size is not a multiple of 4",
			packet: unalignedPacket,
			want:   withSize(unalignedPacketBytes, 0x14, 0x18),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serverListener, addr := newTestListener(t)
			defer serverListener.Close()
			// Connect to the server as if from a PSO client.
			conn := newTestConnection(t, addr)
			defer conn.Close()

			// Handle the connection on the server side and drop it into a Client.
			clientConn, err := serverListener.AcceptTCP()
			if err != nil {
				t.Fatalf("error initializing client connection: %s", err)
			}
			client := NewClient(clientConn)
			client.CryptoSession = encryption.NewBlueBurstCryptoSession()

			// Send bytes from the client and make sure they were encrypted.
			if err := client.Send(context.TODO(), tt.packet); err != nil {
				t.Fatalf("Send() returned an unexpected error: %s", err)
			}
			client.Close()

			buf, err := io.ReadAll(conn)
			if err != nil {
				t.Fatalf("error reading from test connection: %s", err)
			}
			if len(buf) != len(tt.want) {
				t.Fatalf("expected %d bytes to be sent, got %d", len(tt.want), len(buf))
			}

			if diff := cmp.Diff(tt.want, buf); diff == "" {
				t.Fatalf("bytes read from test connection were not encrypted")
			}

			client.CryptoSession.DecryptServer(buf, uint32(len(buf)))

			if diff := cmp.Diff(tt.want, buf); diff != "" {
				t.Fatalf("bytes decrypted from test connection did not match expected; diff:\n%s", diff)
			}
		})
	}
}
