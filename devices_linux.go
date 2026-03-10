//go:build !baremetal

package bluetooth

import (
	"strings"

	"github.com/godbus/dbus/v5"
)

// KnownDevice holds cached BlueZ information about a device that the adapter
// already knows about (paired, connected, or recently seen).
type KnownDevice struct {
	Address   Address
	Name      string
	RSSI      int16
	Paired    bool
	Bonded    bool
	Connected bool
	Trusted   bool
}

// GetKnownDevices returns all devices that BlueZ currently knows about for
// this adapter, regardless of whether they are in range.  This includes
// paired, bonded, trusted and recently-seen devices.
func (a *Adapter) GetKnownDevices() ([]KnownDevice, error) {
	var deviceList map[dbus.ObjectPath]map[string]map[string]dbus.Variant
	if err := a.bluez.Call("org.freedesktop.DBus.ObjectManager.GetManagedObjects", 0).Store(&deviceList); err != nil {
		return nil, err
	}

	adapterPrefix := string(a.adapter.Path())
	var out []KnownDevice
	for path, ifaces := range deviceList {
		props, ok := ifaces["org.bluez.Device1"]
		if !ok {
			continue
		}
		if !strings.HasPrefix(string(path), adapterPrefix) {
			continue
		}

		macStr, _ := props["Address"].Value().(string)
		mac, err := ParseMAC(macStr)
		if err != nil {
			continue
		}

		name, _ := props["Name"].Value().(string)
		if name == "" {
			name, _ = props["Alias"].Value().(string)
		}
		rssi, _ := props["RSSI"].Value().(int16)
		paired, _ := props["Paired"].Value().(bool)
		bonded, _ := props["Bonded"].Value().(bool)
		connected, _ := props["Connected"].Value().(bool)
		trusted, _ := props["Trusted"].Value().(bool)

		out = append(out, KnownDevice{
			Address:   Address{MACAddress: MACAddress{MAC: mac}},
			Name:      name,
			RSSI:      rssi,
			Paired:    paired,
			Bonded:    bonded,
			Connected: connected,
			Trusted:   trusted,
		})
	}
	return out, nil
}

// DeviceFromAddress constructs a Device handle for a known device by MAC
// address without initiating a connection.  This is useful for calling
// methods like Remove() on a device that is out of range.
func (a *Adapter) DeviceFromAddress(address Address) Device {
	devicePath := dbus.ObjectPath(string(a.adapter.Path()) + "/dev_" + strings.Replace(address.MAC.String(), ":", "_", -1))
	return Device{
		Address: address,
		device:  a.bus.Object("org.bluez", devicePath),
		adapter: a,
	}
}
