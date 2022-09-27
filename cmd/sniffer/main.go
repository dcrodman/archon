package main

import (
	"flag"
	"fmt"
	"math"
	"os"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
)

var (
	list   = flag.Bool("l", false, "List devices")
	device = flag.String("d", "en0", "Device on which to listen for packets")
)

func main() {
	flag.Parse()

	// Get a handle to the network interface, or just list them out.
	var deviceIP string
	devs, err := pcap.FindAllDevs()
	if err != nil {
		exit("error finding devices: %v", err)
	}
	for _, dev := range devs {
		if dev.Name == *device {
			for _, address := range dev.Addresses {
				deviceIP = address.IP.String()
				break
			}
		}
		if *list {
			fmt.Println(dev.Name)
		}
	}
	if *list {
		return
	} else if deviceIP == "" {
		exit("unrecognized device: %s", *device)
	}

	// Open the interface for capture.
	handle, err := pcap.OpenLive(*device, math.MaxInt32, false, pcap.BlockForever)
	if err != nil {
		exit("error opening handle: %v", err)
	}
	if err := handle.SetBPFFilter("tcp and not port 443 and not port 80"); err != nil {
		exit("error setting BPF filter: %s", err)
	}

	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	for packet := range packetSource.Packets() {
		flow := packet.TransportLayer().TransportFlow()

		fmt.Printf("source: %v, destination: %v\n", flow.Src(), flow.Dst())
		fmt.Println(packet)
	}
}

func exit(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args)
	os.Exit(1)
}
