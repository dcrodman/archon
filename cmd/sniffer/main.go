package main

import (
	"flag"
	"fmt"
	"math"
	"os"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
)

var device = flag.String("d", "en0", "Device on which to listen for packets")

func main() {
	deviceIP := getDeviceIP()
	if deviceIP == "" {
		exit("invalid device: ", *device)
	}

	handle, err := pcap.OpenLive(*device, math.MaxInt32, false, pcap.BlockForever)
	if err != nil {
		exit("error opening handle: %v", err)
	}
	_ = handle.SetBPFFilter("tcp and not port 443 and not port 80")

	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	for packet := range packetSource.Packets() {
		flow := packet.TransportLayer().TransportFlow()

		fmt.Printf("source: %v, destination: %v\n", flow.Src(), flow.Dst())
		fmt.Println(packet)
	}
}

func exit(format string, args ...interface{}) {
	fmt.Printf(format, args, "\n")
	os.Exit(1)
}

func getDeviceIP() string {
	devs, _ := pcap.FindAllDevs()
	for _, dev := range devs {
		if dev.Name == *device {
			for _, address := range dev.Addresses {
				return address.IP.String()
			}
		}
	}
	return ""
}
