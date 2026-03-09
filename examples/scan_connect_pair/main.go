package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	bluetooth "github.com/SnapITHD/go_bluetooth"
)

var adapter = bluetooth.DefaultAdapter

func main() {
	// Enable BLE interface
	must("enable BLE stack", adapter.Enable())
	agent := adapter.DefaultAgent()
	// Set capability type
	agent.SetCapability(bluetooth.AgentCapabilityKeyboardOnly)
	agent.Register()
	scanner := bufio.NewScanner(os.Stdin)

	// Get device name filter from user
	fmt.Print("Enter device name filter (or press Enter for all devices): ")
	scanner.Scan()
	nameFilter := strings.TrimSpace(scanner.Text())

	// Scan for devices
	foundDevices := scanForDevices(nameFilter)
	if len(foundDevices) == 0 {
		fmt.Println("No devices found!")
		return
	}

	// Let user select device
	selectedDevice := selectDevice(foundDevices, scanner)

	// Connect to selected device
	fmt.Printf("Connecting to %s...\n", selectedDevice.Address.String())
	device, err := adapter.Connect(selectedDevice.Address, bluetooth.ConnectionParams{})
	must("connect to device", err)
	defer device.Disconnect()

	fmt.Printf("Connected to %s\n", device.Address.MAC.String())

	// Check if already paired
	paired, err := device.IsPaired()
	if err != nil {
		fmt.Printf("Warning: Could not check pairing status: %v\n", err)
	}

	if !paired {
		fmt.Printf("Pairing...")

		fmt.Print("Enter code (leave empty for no code): ")
		scanner.Scan()
		codeResponse := strings.ToLower(strings.TrimSpace(scanner.Text()))

		if codeResponse == "" {
			err = device.Pair()
		} else {
			err = device.PairWithCode(codeResponse)
		}

		if err != nil {
			fmt.Printf("Pairing failed: %v\n", err)
			paired = false
		} else {
			fmt.Printf("Device paired successfully\n")
			paired = true
		}

	} else {
		fmt.Printf("Device already paired\n")
	}

	err = device.TrustDevice()
	if err != nil {
		fmt.Printf("Failed to trust device: %v\n", err)
	} else {
		fmt.Printf("Device trusted\n")
	}

	// Print device info
	printDeviceInfo(device)

	// Ask if user wants to remove/unpair the device
	fmt.Print("Remove/unpair this device? (y/N): ")
	scanner.Scan()
	removeResponse := strings.ToLower(strings.TrimSpace(scanner.Text()))

	if removeResponse == "y" || removeResponse == "yes" {
		fmt.Printf("Removing device %s...\n", device.Address.MAC.String())
		device.Remove()

	} else {
		// Disconnect
		fmt.Printf("Disconnecting from %s...\n", device.Address.MAC.String())
		err = device.Disconnect()
		if err != nil {
			fmt.Printf("Disconnect error: %v\n", err)
		} else {
			fmt.Printf("Disconnected successfully\n")
		}
	}

}

func scanForDevices(nameFilter string) []bluetooth.ScanResult {
	fmt.Printf("📡 Scanning for devices")
	if nameFilter != "" {
		fmt.Printf(" (filter: %s)", nameFilter)
	}
	fmt.Println("...")

	var foundDevices []bluetooth.ScanResult
	deviceSeen := make(map[string]bool)

	// Start scanning
	go adapter.Scan(func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
		// Avoid duplicates
		addr := result.Address.String()
		if deviceSeen[addr] {
			return
		}

		// Apply name filter if specified
		deviceName := result.LocalName()
		if nameFilter != "" && !strings.Contains(strings.ToLower(deviceName), strings.ToLower(nameFilter)) {
			return
		}

		deviceSeen[addr] = true
		foundDevices = append(foundDevices, result)

		fmt.Printf("Found: %s (RSSI: %d) - %s\n",
			addr, result.RSSI, deviceName)
	})

	// Scan for 10 seconds
	time.Sleep(5 * time.Second)
	adapter.StopScan()

	fmt.Printf("\nFound %d device(s)\n\n", len(foundDevices))
	return foundDevices
}

func selectDevice(devices []bluetooth.ScanResult, scanner *bufio.Scanner) bluetooth.ScanResult {
	fmt.Println("Select a device:")
	for i, device := range devices {
		fmt.Printf("%d. %s - %s (RSSI: %d)\n",
			i+1, device.Address.String(), device.LocalName(), device.RSSI)
	}

	for {
		fmt.Print("Enter device number: ")
		scanner.Scan()
		input := strings.TrimSpace(scanner.Text())

		selection, err := strconv.Atoi(input)
		if err != nil || selection < 1 || selection > len(devices) {
			fmt.Printf("Invalid selection. Please enter 1-%d\n", len(devices))
			continue
		}

		return devices[selection-1]
	}
}

func printDeviceInfo(device bluetooth.Device) {
	fmt.Println("\nDevice Information:")
	fmt.Printf("Address: %s\n", device.Address.MAC.String())

	// Check connection status
	connected, err := device.IsConnected()
	if err != nil {
		fmt.Printf("Connected: Unknown (%v)\n", err)
	} else {
		fmt.Printf("Connected: %t\n", connected)
	}

	// Check pairing status
	paired, err := device.IsPaired()
	if err != nil {
		fmt.Printf("Paired: Unknown (%v)\n", err)
	} else {
		fmt.Printf("Paired: %t\n", paired)
	}

	// Check trust status
	trusted, err := device.IsTrusted()
	if err != nil {
		fmt.Printf("Trusted: Unknown (%v)\n", err)
	} else {
		fmt.Printf("Trusted: %t\n", trusted)
	}

	// Try to discover services
	fmt.Println("\n🔍 Discovering services...")
	services, err := device.DiscoverServices(nil)
	if err != nil {
		fmt.Printf("Failed to discover services: %v\n", err)
	} else {
		fmt.Printf("Found %d service(s):\n", len(services))
		for i, service := range services {
			fmt.Printf("  %d. %s\n", i+1, service.UUID().String())
		}
	}
	fmt.Println()
}

func must(action string, err error) {
	if err != nil {
		panic("failed to " + action + ": " + err.Error())
	}
}
