package login_server

import (
	"net/http"
)

// Ship representation used for responding to requests for the ship list.
type ShipJsonEntry struct {
	Shipname   [23]byte
	Hostname   string
	Port       string
	NumPlayers int
}

// Return a JSON string to the client with the name, hostname, port,
// and player count.
func handleShipCountRequest(w http.ResponseWriter, req *http.Request) {
	if shipConnections.Count() == 0 {
		w.Write([]byte("[]"))
	} else {
		// TODO: Pull this from a cache
		w.Write([]byte("[]"))
	}
}

// Distributing a shared symmetric key is insecure, so in order to
// allow symmetric encryption an initial handshake is performed
// using PKCS1 and known keys. The shipgate keeps all ship public
// keys (along with its private key) locally and assumes that
// connecting ships in turn have its public key. This doubles as a
// registration mechanism since we only allow ships whose public keys
// we have stored to connect.
func authenticateClient(client *LoginClient) {

}

func processShipgatePacket(client *LoginClient) error {
	return nil
}
