package shipgate

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	ioutil "io/ioutil"
	"log"
	"net"
	http "net/http"
	"strconv"
	"sync"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/dcrodman/archon/internal/core"
	"github.com/dcrodman/archon/internal/core/bytes"
	"github.com/dcrodman/archon/internal/core/data"
	"github.com/dcrodman/archon/internal/packets"
)

func NewRPCClient(cfg *core.Config) Shipgate {
	// Load client cert
	cert, err := tls.LoadX509KeyPair(cfg.ShipgateCertFile, cfg.ShipgateServer.SSLKeyFile)
	if err != nil {
		log.Fatal(err)
	}

	// Load CA cert
	caCert, err := ioutil.ReadFile(cfg.ShipgateCertFile)
	if err != nil {
		log.Fatal(err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	// Setup HTTPS client
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
	}
	tlsConfig.BuildNameToCertificate()
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	return NewShipgateProtobufClient(cfg.ShipgateAddress(), httpClient)
}

type ShipRegistrationClient struct {
	Logger         *logrus.Logger
	ShipgateClient Shipgate

	connectedShipsMutex sync.RWMutex
	connectedShips      []shipInfo
}

// Character servers internal representation of Ship connection information
// for the ship selection screen.
type shipInfo struct {
	id   int
	name []byte
	ip   string
	port string
}

func (s *ShipRegistrationClient) StartShipRefreshLoop(ctx context.Context) error {
	// The first set is fetched synchronously so that the ship list will start populated.
	// Also gives us a chance to validate that the shipgate address is valid.
	if err := s.refreshShipList(); err != nil {
		return err
	}
	go s.startShipListRefreshLoop(ctx)

	return nil
}

func (s *ShipRegistrationClient) GetConnectedShipList() []packets.ShipListEntry {
	s.connectedShipsMutex.RLock()
	defer s.connectedShipsMutex.RUnlock()

	shipList := make([]packets.ShipListEntry, 0)
	for i, ship := range s.connectedShips {
		entry := packets.ShipListEntry{
			MenuID:   uint16(i + 1),
			ShipID:   uint32(ship.id),
			ShipName: [36]byte{},
		}
		copy(entry.ShipName[:], ship.name)
		shipList = append(shipList, entry)
	}
	if len(shipList) == 0 {
		// A "No Ships!" entry is shown if we either can't connect to the shipgate or
		// the shipgate doesn't report any connected ships.
		shipList = append(shipList, packets.ShipListEntry{
			MenuID: 0xFF, ShipID: 0xFF, ShipName: [36]byte{},
		})
		copy(shipList[0].ShipName[:], ("No Ships!")[:])
	}
	return shipList
}

func (s *ShipRegistrationClient) GetSelectedShipAddress(selectedShip uint32) (net.IP, int, error) {
	s.connectedShipsMutex.Lock()
	defer s.connectedShipsMutex.Unlock()

	if selectedShip >= uint32(len(s.connectedShips)) {
		return nil, 0, fmt.Errorf("invalid ship selection: %d", selectedShip)
	}

	shipIP := net.ParseIP(s.connectedShips[selectedShip].ip).To4()
	shipPort, _ := strconv.Atoi(s.connectedShips[selectedShip].port)
	return shipIP, shipPort, nil
}

func (s *ShipRegistrationClient) AuthenticateAccount(ctx context.Context, username, password string) (*data.Account, error) {
	accountResp, err := s.ShipgateClient.AuthenticateAccount(ctx, &AccountAuthRequest{
		Username: username,
		Password: password,
	})
	if err != nil {
		return nil, err
	}

	rd, err := time.Parse(time.RFC3339, accountResp.GetRegistrationDate())
	if err != nil {
		return nil, err
	}

	var pl byte
	if b := accountResp.GetPriviledgeLevel(); len(b) != 0 {
		pl = b[0]
	}

	return &data.Account{
		Model: gorm.Model{
			ID: uint(accountResp.Id),
		},
		Username:         accountResp.GetUsername(),
		Password:         password,
		Email:            accountResp.GetEmail(),
		RegistrationDate: rd,
		Guildcard:        int(accountResp.GetGuildcard()),
		GM:               accountResp.GetGM(),
		Banned:           accountResp.GetBanned(),
		Active:           accountResp.GetActive(),
		TeamID:           int(accountResp.TeamId),
		PrivilegeLevel:   pl,
	}, nil
}

// Starts a loop that makes an API request to the shipgate server over an interval
// in order to query the list of active ships. The result is parsed and stored in
// the Server's ships field.
func (s *ShipRegistrationClient) startShipListRefreshLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second * 10):
			if err := s.refreshShipList(); err != nil {
				s.Logger.Errorf(err.Error())
			}
		}
	}
}

func (s *ShipRegistrationClient) refreshShipList() error {
	response, err := s.ShipgateClient.GetActiveShips(context.Background(), &empty.Empty{})
	if err != nil {
		return fmt.Errorf("failed to fetch ships from shipgate: %s", err)
	}

	ships := make([]shipInfo, 0)
	for _, s := range response.Ships {
		ships = append(ships, shipInfo{
			id:   int(s.Id),
			name: bytes.ConvertToUtf16(s.Name),
			ip:   s.Ip,
			port: s.Port,
		})
	}

	s.connectedShipsMutex.Lock()
	s.connectedShips = ships
	s.connectedShipsMutex.Unlock()
	return nil
}
