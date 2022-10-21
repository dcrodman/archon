package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
)

const defaultFilter = "tcp portrange 11000-11001"

var (
	list   = flag.Bool("l", false, "List devices")
	device = flag.String("d", "en0", "Device on which to listen for packets")
	filter = flag.String("f", defaultFilter, "BPF packet filter to apply")
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

	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	for packet := range packetSource.Packets() {
		if err != nil {
			exit("error reading packet: %v", err)
		}

		// We don't care about the TCP handshake packets or anything lower level than layer 4.
		if packet.ApplicationLayer() != nil {
			flow := packet.TransportLayer().TransportFlow()
			fmt.Printf("source: %v, destination: %v\n", flow.Src(), flow.Dst())
			fmt.Println(packet.Dump())
		}
	}
}
