/*
* Archon PSOBB Server
* Copyright (C) 2014 Andrew Rodman
*
* This program is free software: you can redistribute it and/or modify
* it under the terms of the GNU General Public License as published by
* the Free Software Foundation, either version 3 of the License, or
* (at your option) any later version.
*
* This program is distributed in the hope that it will be useful,
* but WITHOUT ANY WARRANTY; without even the implied warranty of
* MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
* GNU General Public License for more details.
*
* You should have received a copy of the GNU General Public License
* along with this program.  If not, see <http://www.gnu.org/licenses/>.
* ---------------------------------------------------------------------
*
* Utility script for generating public and private RSA keypairs to
* be used for encrypted communications between the shipgate and ships.
 */
package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
)

func main() {
	filename := "key"
	numArgs := len(os.Args)
	if numArgs > 2 {
		fmt.Println("Generates a public/private RSA keypair for the shipgate or ship servers.")
		fmt.Println("Usage: keygen.go [base filename]")
		os.Exit(1)
	}
	if numArgs > 1 {
		filename = os.Args[1]
	}
	// Generate our private key.
	privateKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		fmt.Printf("Error generating RSA key: %s\n", err.Error())
		os.Exit(-1)
	}

	// Now write it to our pem file.
	bytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})
	ioutil.WriteFile(filename+".pem", bytes, 0644)

	// Compute the distributable public key and write that to our pub file.
	key, err := x509.MarshalPKIXPublicKey(privateKey.Public())
	if err != nil {
		fmt.Printf("Failed to generate public key: %s\n", err.Error())
	}
	bytes = pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: key,
	})
	ioutil.WriteFile(filename+".pub", bytes, 0644)

	fmt.Println("Generated private key: " + filename + ".pem" +
		"\nGenerated public key: " + filename + ".pub")
}
