// Generates a self-signed X.509 certificate (valid for 10 years) and corresponding
// key for TLSv1 authentication between the shipgate's API  server and a client.
//
// Usage:
//     generate_cert
//
// The tool will prompt for an IP address _OR_ CIDR range for the C.509 certificate.
// If you want to make your life a little easier (albeit technically less secure),
// use 0.0.0.0/32 as the address.
//
// Some code borrowed from the go standard library:
// src/crypto/tls/generate_cert.go
package main

import (
	"bufio"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"time"
)

const (
	certificateFilename = "certificate.pem"
	privateKeyFilename  = "key.pem"
)

func main() {
	fmt.Print("server's external_ip (in config.yaml) or CIDR block: ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()

	serverIp := scanner.Text()

	template, err := createX509Template(serverIp)
	if err != nil {
		fmt.Println("failed to create X.509 template:", err)
		return
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		fmt.Printf("Error generating RSA key: %s\n", err.Error())
		return
	}

	generateCertificateFile(template, privateKey)
	generatePrivateKeyFile(privateKey)

	fmt.Printf(
		"\nDone! Place %s and %s in the config folder for the shipgate and\n"+
			"distrubute %s to any standalone ships that will be connecting to your\n"+
			"server. If the server installation is self-contained then you just need to copy\n"+
			"those two files to the server's config directory.\n",
		certificateFilename,
		privateKeyFilename,
		certificateFilename,
	)
}

func createX509Template(serverIP string) (*x509.Certificate, error) {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, err
	}

	ip, ipnet, err := net.ParseCIDR(serverIP)
	if err != nil {
		return nil, err
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(time.Hour * 24 * 235 * 10)

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Archon PSO Server"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		IPAddresses:           []net.IP{ip},
		PermittedIPRanges:     []*net.IPNet{ipnet},
	}
	return template, nil
}

func generateCertificateFile(template *x509.Certificate, privateKey *rsa.PrivateKey) {
	certBytes, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	certOut, err := os.Create("certificate.pem")
	if err != nil {
		fmt.Printf("failed to create %s: %s\n", certificateFilename, err)
		return
	}

	err = pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	if err != nil {
		fmt.Printf("failed to create %s: %s", certificateFilename, err)
		return
	}
	certOut.Close()

	fmt.Printf("wrote %s\n", certificateFilename)
}

func generatePrivateKeyFile(privateKey *rsa.PrivateKey) {
	keyOut, err := os.OpenFile("key.pem", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Printf("failed to create %s: %s\n", privateKeyFilename, err)
		return
	}

	keyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	err = pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: keyBytes})
	if err != nil {
		fmt.Printf("failed to create %s: %s\n", privateKeyFilename, err)
		return
	}
	keyOut.Close()

	fmt.Printf("wrote %s\n", privateKeyFilename)
}
