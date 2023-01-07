package main

import (
	"bufio"
	"flag"
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
)

const (
	// Default BPF filter set on pcap.
	defaultFilter = "tcp portrange 11000-11002 or tcp portrange 12001-12003 or tcp portrange 15000-15003 or tcp portrange 5278-5290"
	// Threshold after which packets will be truncated.
	truncatePacketLimit = 0x1000
)

var (
	list     = flag.Bool("l", false, "List devices")
	device   = flag.String("d", "en0", "Device on which to listen for packets")
	filter   = flag.String("f", defaultFilter, "BPF packet filter to apply")
	output   = flag.String("o", "", "File to which to output logs (default stdout)")
	truncate = flag.Bool("truncate", false, fmt.Sprintf("Truncate packets over %d bytes long", truncatePacketLimit))
)

func main() {
	flag.Parse()

	if *list {
		printNetworkDevices()
	} else {
		startSniffer()
	}
}

func exit(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args)
	os.Exit(1)
}

func printNetworkDevices() {
	devs, err := pcap.FindAllDevs()
	if err != nil {
		exit("error finding devices: %v", err)
	}

	for _, dev := range devs {
		var ipv4Addresses []string
		for _, address := range dev.Addresses {
			ipv4Addr := address.IP.To4()
			if ipv4Addr != nil {
				ipv4Addresses = append(ipv4Addresses, ipv4Addr.String())
			}
		}
		fmt.Printf("%v (%v)\n", dev.Name, strings.Join(ipv4Addresses, ","))
	}
}

func startSniffer() {
	// Open the interface for capture.
	handle, err := pcap.OpenLive(*device, math.MaxInt32, false, pcap.BlockForever)
	if err != nil {
		exit("error opening handle: %v", err)
	}
	if err := handle.SetBPFFilter(*filter); err != nil {
		exit("error setting BPF filter: %s", err)
	}

	outputFile := bufio.NewWriter(os.Stdout)
	if *output != "" {
		f, err := os.OpenFile(*output, os.O_CREATE, 0666)
		if err != nil {
			exit("error opening file for output: %v", err)
		}
		outputFile = bufio.NewWriter(f)
	}

	sniffer := &sniffer{Writer: outputFile}
	packetChan := make(chan gopacket.Packet)
	go sniffer.startReading(packetChan)

	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	for packet := range packetSource.Packets() {
		if err != nil {
			exit("error reading packet: %v", err)
		}

		// We don't care about the TCP handshake packets or anything lower level than layer 4.
		if packet.ApplicationLayer() != nil {
			packetChan <- packet
		}
	}
}
