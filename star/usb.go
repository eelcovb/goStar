package star

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/google/gousb"
)

// Allowed ID characters
const (
	idChars     = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789" // Numbers double defined to optimize distribution
	idMaxLength = 8
	starFullCut = "\x1B\x64\x32"
	starBR      = "\x1B\x61\x01"
)

// Printer defines the printer type
type Printer struct {
	Vendor       uint16
	Product      uint16
	Manufacturer string
	ProductName  string
	SerialNumber string
	Description  string
	Address      int
	Speed        string
	Bus          int
	Port         int
}

// PrinterList is an array of printers
type PrinterList []Printer

// Discover scans the USB for Star devices
func (pl *PrinterList) Discover() (int, error) {

	// Only check for Star devices
	starDeviceID := gousb.ID(0x519)

	// Initiate a context
	ctx := gousb.NewContext()
	defer ctx.Close()

	// Get the star devices
	devs, err := ctx.OpenDevices(func(desc *gousb.DeviceDesc) bool {
		// Only return true when a device has the starDeviceID
		return desc.Vendor == starDeviceID
	})

	if err != nil {
		return 0, fmt.Errorf("Failed to open USB devices: %s", err)
	}

	// Register the devices to the local data structure
	for _, d := range devs {

		p := Printer{}
		p.Vendor = uint16(d.Desc.Vendor)
		p.Product = uint16(d.Desc.Product)
		p.Manufacturer, _ = d.Manufacturer()
		p.ProductName, _ = d.Product()
		p.SerialNumber, _ = d.SerialNumber()
		p.Description = d.String()
		p.Bus = d.Desc.Bus
		p.Address = d.Desc.Address
		p.Speed = d.Desc.Speed.String()
		p.Port = d.Desc.Port

		// Add this printer to the list
		*pl = append(*pl, p)
	}
	return len(*pl), nil
}

// MakeRandomID generates a random ID based
// on the specifications set for a Star USB ID
func MakeRandomID() string {

	// Seed the randomizer
	rand.Seed(time.Now().UnixNano())

	newID := ""
	idSourceLen := len(idChars)

	for i := 0; i < idMaxLength; i++ {
		newID += string(idChars[rand.Intn(idSourceLen)])
	}
	return newID
}

func validateUsbID(usbID *string) bool {
	// Serialnumbers should only contain numbers or capitals
	l := len(*usbID)
	if l == 0 {
		*usbID = "????????"
		return true
	}
	if l > 8 {
		return false
	}

	for _, r := range *usbID {
		if r < 48 || (r > 57 && r < 65) || r > 90 {
			return false
		}
	}
	return true
}

func getInOutEndpoints(intf *gousb.Interface) (in, out int) {

	endpoints := intf.Setting.Endpoints
	for _, endpoint := range endpoints {
		if endpoint.Direction == gousb.EndpointDirectionIn {
			in = endpoint.Number
		} else {
			out = endpoint.Number
		}
	}

	return in, out
}

// SendUsbIDToPrinter sends the serial information to a certain
// USB endpoint
// If s is set to true, a message is printed on the printer to reboot
func SendUsbIDToPrinter(usbID string, address, bus, port int, printRebootMsg bool) error {

	// Commands for printer
	sCmdPrefix := "\x1b\x23\x23\x57\x38\x2c"
	sCmdPostfix := "\x0a\x00"
	sResetCmdP1 := "\x1b\x1d\x23\x2b\x43\x30\x30\x30\x32\x0a\x00\x1b\x1d\x23\x54\x30"
	sResetCmdP2 := "\x30\x30\x30\x30\x0A\x00"

	path := fmt.Sprintf("%d/%d/%d", address, bus, port)

	ctx := gousb.NewContext()
	defer ctx.Close()

	devs, err := ctx.OpenDevices(func(desc *gousb.DeviceDesc) bool {
		return desc.Address == address &&
			desc.Bus == bus &&
			desc.Port == port
	})

	if err != nil {
		return fmt.Errorf("SendUsbIDToPrinter: USB error: %s", err)
	}

	// Set defer closes
	for _, dev := range devs {
		defer dev.Close()
	}

	lenDevs := len(devs)

	if lenDevs == 0 {
		return fmt.Errorf("SendUsbIDToPrinter: device %s has gone away", path)
	}
	if lenDevs > 1 {
		return fmt.Errorf("SendUsbIDToPrinter: multiple (%d) devices found that match path %s", lenDevs, path)
	}

	// There should be only one device at this point
	device := devs[0]

	// Claim the default interface
	intf, done, err := device.DefaultInterface()
	if err != nil {
		return fmt.Errorf("%s.DefaultInterface(): %v", device, err)
	}

	defer done()
	defer intf.Close()

	// Get the enpoints for the interface
	_, out := getInOutEndpoints(intf)

	// Open the out endpoint
	epOut, err := intf.OutEndpoint(out)
	if err != nil {
		return fmt.Errorf("%s.OutEndpoint(%d): %s", intf, out, err)
	}

	// Send the serialID data
	// Writing is slowed with 1 sec delays
	// to give the printer time to process
	epOut.Write([]byte(sCmdPrefix + usbID + sCmdPostfix))
	time.Sleep(1 * time.Second)
	// Write store payload
	epOut.Write([]byte(sResetCmdP1))
	time.Sleep(1 * time.Second)
	epOut.Write([]byte(sResetCmdP2))

	// Determine if a reboot message needs to be send
	if printRebootMsg {
		fmt.Printf("Printing reboot message to printer on address=%d,bus=%d,port=%d",
			address, bus, port)
		// Delay for a bit in order for the serialID writing process to finish
		time.Sleep(5 * time.Second)
		// Write reboot message
		epOut.Write([]byte("Please reboot me, my new USBID = " + usbID))
		epOut.Write([]byte(starBR + starBR + starBR + starBR + starBR + starFullCut))
	}

	return nil
}

// SetUsbID sets the USB-ID for printer
func (p *Printer) SetUsbID(usbID string, printRebootMsg bool) error {
	if p.SerialNumber == usbID {
		return fmt.Errorf("Device already had USB-ID [%s] set", usbID)
	}
	if !validateUsbID(&usbID) {
		return fmt.Errorf("USB-ID [%s] is invalid", usbID)
	}

	return SendUsbIDToPrinter(usbID, p.Address, p.Bus, p.Port, printRebootMsg)
}

// SetGenericIDForAll set a generic ID for all printers without an ID
// Error is set if something goes wrong
func (pl *PrinterList) SetGenericIDForAll() (int, error) {
	for _, p := range *pl {
		if p.SerialNumber == "" {
			err := SendUsbIDToPrinter(MakeRandomID(), p.Address, p.Bus, p.Port, true)
			if err != nil {
				return 0, err
			}
		}
	}
	return 0, nil
}
