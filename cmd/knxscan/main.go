package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/vapourismo/knx-go/knx"
	"github.com/vapourismo/knx-go/knx/cemi"
	"github.com/vapourismo/knx-go/knx/util"
)

func main() {
	tunIpP := flag.String("gw", "127.0.0.1:3671", "knx ip interface address")
	lineP := flag.String("line", "1.1", "knx line to scan")
	startP := flag.Uint("start", 1, "start address of device")
	stopP := flag.Uint("stop", 255, "last device to scan")

	flag.Parse()

	if *startP > *stopP {
		panic("start > stop")
	}

	logger := log.New(os.Stdout, "", log.LstdFlags)
	util.Logger = logger

	tun, err := knx.NewTransportTunnel(*tunIpP,
		knx.TunnelConfig{})

	if err != nil {
		panic(err)
	}

	// Loop for ever. Failures don't matter, we'll always retry.
	for i := *startP; i <= *stopP; i += 1 {
		addr, err := cemi.NewIndividualAddrString(fmt.Sprintf("%s.%d", *lineP, i))
		if err != nil {
			panic(err) // should never happen
		}

		cc, err := tun.Dial(addr)
		if err != nil {
			continue
		}
		fmt.Printf("created connection to %s.%d\n", *lineP, i)
		// TODO: receive info using APDU
		cc.Close()
	}
	tun.Close()
}
