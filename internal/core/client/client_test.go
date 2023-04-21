package client

import (
	"net"
	"testing"

	"github.com/dcrodman/archon/internal/core/bytes"
	"github.com/dcrodman/archon/internal/packets"
	"github.com/google/go-cmp/cmp"
)

var (
	testPacket = &packets.CharacterAck{
		Header: packets.BBHeader{
			Size: 0x10,
			Type: packets.LoginCharAckType,
		},
		Slot: 1,
		Flag: 1,
	}
	testPacketBytes, _ = bytes.BytesFromStruct(testPacket)
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
	if err := client.SendRaw(testPacket); err != nil {
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
	serverListener, addr := newTestListener(t)
	// Connect to the server as if from a PSO client.
	conn := newTestConnection(t, addr)

	// Handle the connection on the server side and drop it into a Client.
	clientConn, err := serverListener.AcceptTCP()
	if err != nil {
		t.Fatalf("error initializing client connection: %s", err)
	}
	client := NewClient(clientConn)
	client.CryptoSession = NewBlueBurstCryptoSession()

	// Send bytes from the client and make sure they were encrypted.
	if err := client.Send(testPacket); err != nil {
		t.Fatalf("SendRaw() returned an unexpected error: %s", err)
	}
	client.Close()

	buf := make([]byte, 16)
	if _, err := conn.Read(buf); err != nil {
		t.Fatalf("error reading from test connection: %s", err)
	}

	if diff := cmp.Diff(testPacketBytes, buf); diff == "" {
		t.Fatalf("bytes read from test connection were not encrypted")
	}

	// Hack around the abstraction to decrypt the packet since we never
	// need to do this and the design intentionally hides it.
	client.CryptoSession.(*blueBurstCryptSession).serverCrypt.Decrypt(buf, uint32(len(testPacketBytes)))

	if diff := cmp.Diff(testPacketBytes, buf); diff != "" {
		t.Fatalf("bytes decrypted from test connection did not match expected; diff:\n%s", diff)
	}
}

func Test_adjustPacketLength(t *testing.T) {
	testPacketNoSize := &packets.CharacterAck{
		Header: packets.BBHeader{
			Type: packets.LoginCharAckType,
		},
		Slot: 1,
		Flag: 1,
	}
	testPacketBytesNoSize, _ := bytes.BytesFromStruct(testPacketNoSize)

	longerTestPacket := make([]byte, len(testPacketBytes))
	copy(longerTestPacket, testPacketBytes)
	longerTestPacket = append(longerTestPacket, 0x01, 0x01)

	expectedLongerTestPacket := make([]byte, len(longerTestPacket))
	copy(expectedLongerTestPacket, longerTestPacket)
	expectedLongerTestPacket = append(expectedLongerTestPacket, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00)
	expectedLongerTestPacket[0] = 24

	type args struct {
		data       []byte
		length     uint16
		headerSize uint16
	}
	tests := []struct {
		name       string
		args       args
		want       []byte
		wantLength uint16
	}{
		{
			name: "packet is the correct size",
			args: args{
				data:       testPacketBytes,
				length:     uint16(len(testPacketBytes)),
				headerSize: packets.BBHeaderSize,
			},
			want:       testPacketBytes,
			wantLength: uint16(len(testPacketBytes)),
		},
		{
			name: "packet size is not set",
			args: args{
				data:       testPacketBytesNoSize,
				length:     uint16(len(testPacketBytesNoSize)),
				headerSize: packets.BBHeaderSize,
			},
			want:       testPacketBytes,
			wantLength: uint16(len(testPacketBytes)),
		},
		{
			name: "packet length is not a multiple of the header size",
			args: args{
				data:       longerTestPacket,
				length:     uint16(len(longerTestPacket)),
				headerSize: packets.BBHeaderSize,
			},
			want:       expectedLongerTestPacket,
			wantLength: uint16(len(expectedLongerTestPacket)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkt, length := adjustPacketLength(tt.args.data, tt.args.length, tt.args.headerSize)
			if diff := cmp.Diff(tt.want, pkt); diff != "" {
				t.Errorf("adjustPacketLength() want = %v, got = %v", tt.want, pkt)
			}

			if length != tt.wantLength {
				t.Errorf("adjustPacketLength() want = %v, got = %v", tt.wantLength, length)
			}

			if pkt[00] != byte(tt.wantLength) {
				t.Errorf("header size was not updated; want = %d, got = %d", tt.wantLength, pkt[00])
			}
		})
	}
}
