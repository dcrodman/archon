package core

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestConfig_DatabaseURL(t *testing.T) {
	cfg := &Config{
		Database: struct {
			Engine   string `mapstructure:"engine"`
			Filename string `mapstructure:"filename"`
			Host     string `mapstructure:"host"`
			Port     int    `mapstructure:"port"`
			Name     string `mapstructure:"name"`
			Username string `mapstructure:"username"`
			Password string `mapstructure:"password"`
			SSLMode  string `mapstructure:"disable"`
		}{
			Engine:   "postgres",
			Host:     "localhost",
			Port:     5432,
			Name:     "testdb",
			Username: "testuser",
			Password: "testpassword",
		},
	}

	url := cfg.DatabaseURL()
	expected := "host=localhost port=5432 dbname=testdb user=testuser password=testpassword sslmode="
	if url != expected {
		t.Errorf("DatabaseURL() want = %s, got = %s", expected, url)
	}
}

func TestConfig_ShipgateAddress(t *testing.T) {
	cfg := &Config{
		Hostname: "127.0.0.1",
		ShipgateServer: struct {
			Port int `mapstructure:"port"`
		}{
			Port: 12345,
		},
	}

	addr := cfg.ShipgateAddress()
	expected := "http://127.0.0.1:12345"
	if addr != expected {
		t.Errorf("ShipgateAddress() want = %s, got = %s", expected, addr)
	}
}

func TestConfig_BroadcastIP(t *testing.T) {
	cfg := &Config{ExternalIP: "192.168.1.5"}

	ip := cfg.BroadcastIP()
	expected := [4]byte{192, 168, 1, 5}
	if diff := cmp.Diff(expected, ip); diff != "" {
		t.Errorf("BroadcastIP() generated the wrong IP; diff:\n%s", diff)
	}
}
