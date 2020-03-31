/*
* Generates a self-signed X.509 certificate (valid for 5 years) and
* corresponding key for TLSv1 authentication between a ship and central
* shipgate. Both files should be placed in the ship's configuration
* directory and the cert.pem file distributed to any ships that need
* to connect to the server.
*
* Some code borrowed from the go standard library:
* src/crypto/tls/generate_cert.go
 */
package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"flag"
	"log"
	"math/big"
	"net"
	"os"
	"strings"
	"time"
)

var (
	host = flag.String("host", "", "Comma-separated hostnames and IPs to generate a certificate for")
)

func main() {
	flag.Parse()

	if len(*host) == 0 {
		log.Fatalf("Missing required --host parameter")
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)

	notBefore := time.Now()
	notAfter := notBefore.Add(time.Hour * 24 * 235 * 5)

	var hostIPs []net.IP
	for _, s := range strings.Split(*host, ",") {
		ip := net.ParseIP(s)
		if ip == nil {
			log.Fatalf("Invalid hostname: %s\n", s)
		}
		hostIPs = append(hostIPs, ip)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Archon PSO Server"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:        true,
		IPAddresses: hostIPs,
	}

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatalf("Error generating RSA key: %s\n", err.Error())
	}
	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		log.Fatalf(err.Error())
	}

	certOut, err := os.Create("certificate.pem")
	if err != nil {
		log.Fatalf("failed to open certitifcate.pem for writing: %s", err)
	}
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	certOut.Close()
	log.Print("written certificate.pem\n")

	keyOut, err := os.OpenFile("key.pem", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Print("failed to open key.pem for writing:", err)
		return
	}
	pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	keyOut.Close()
	log.Print("written key.pem\n")

	log.Printf("Place the cert and key in the config folder of the shipgate\n" +
		"and distrubute the certificate to connecting ships\n")
}
