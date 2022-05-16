// starSerial
// (c) 2018 CloverTwo, Inc.

package main

import (
	"flag"
	"fmt"
	"log"

	"gitlab.clovertwo.com/clovertwo/starUSB/star"
)

const version = "1.0"

func main() {

	fmt.Printf("*** goStar v%s (c) 2022 Impero, LLC.\n", version)

	fUsbID := flag.String("usbID", "AB123456", "A string with length 8, only capitals or numeric values")
	randomID := flag.Bool("random", false, "Set a random ID")
	simulate := flag.Bool("simulate", false, "Simulate actions")
	list := flag.Bool("list", false, "List Star USB printers")
	address := flag.Int("address", -1, "Target device USB address")
	bus := flag.Int("bus", -1, "Target device bus ID")
	port := flag.Int("port", -1, "Target device port")
	printRebootMsg := flag.Bool("printreboot", false, "Print reboot message on printer")

	flag.Parse()

	fmt.Println("Scanning the USB bus for Star devices..")

	targetSet := *address > -1 && *bus > -1 && *port > -1

	usbID := *fUsbID

	pl := star.PrinterList{}
	devicesFound, err := pl.Discover()

	if err != nil {
		fmt.Printf("Could not discover devices: %s - fatal", err)
		return
	}

	if devicesFound > 0 {
		fmt.Printf("%d Star device(s) found:\n", devicesFound)
		for idx, p := range pl {
			fmt.Printf("[%d]: %s address:%d bus:%d port:%d",
				idx, p.ProductName, p.Address, p.Bus, p.Port)
			if p.SerialNumber == "" {
				fmt.Println(" USB-ID: not set")
			} else {
				fmt.Printf(" USB-ID: [%s]\n", p.SerialNumber)
			}

			if !*list {
				if *randomID {
					usbID = star.MakeRandomID()
					fmt.Println("Commandline request for random ID..")
				}

				fmt.Printf("Setting USB-ID to [%s]\n", usbID)
				if *simulate {
					fmt.Println("--- Simulation mode ---")
				} else {
					// If there is a target selection, only change the target
					if targetSet && (p.Address == *address &&
						p.Bus == *bus &&
						p.Port == *port) || !targetSet {
						if err := p.SetUsbID(usbID, *printRebootMsg); err != nil {
							log.Fatalf("Failed to set USB-ID for %d: %s", idx, err)
						} else {
							fmt.Println("Reboot printer for changes to take effect")
							fmt.Println("Note that rebooting the printer might change its USB address")
						}
					}
				}
			}
		}

	} else {
		fmt.Println("No Star devices found")
	}
}
