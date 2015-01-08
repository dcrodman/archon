package util

import "net"

// Create a TCP socket that is listening and ready to Accept().
func OpenSocket(host, port string) (*net.TCPListener, error) {
	hostAddress, err := net.ResolveTCPAddr("tcp", host+":"+port)
	if err != nil {
		return nil, &ServerError{message: "Error creating socket: " + err.Error()}
	}
	socket, err := net.ListenTCP("tcp", hostAddress)
	if err != nil {
		return nil, &ServerError{message: "Error Listening on Socket: " + err.Error()}
	}
	return socket, nil
}
