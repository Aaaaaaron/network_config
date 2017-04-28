package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

var gconfig Config

func init() {
	gconfig.Devices = append(gconfig.Devices, Device{Name: "eth0"})
	gconfig.Devices = append(gconfig.Devices, Device{Name: "eth1"})
	gconfig.Devices = append(gconfig.Devices, Device{Name: "eth2"})
	gconfig.Devices = append(gconfig.Devices, Device{Name: "eth3"})
	gconfig.Devices = append(gconfig.Devices, Device{Name: "eth4"})
	gconfig.Devices = append(gconfig.Devices, Device{Name: "eth5"})
	gconfig.Bonds = append(gconfig.Bonds, Bond{Name: "bond0", Devs: []string{"eth0", "eth1"}})
	gconfig.Bridges = append(gconfig.Bridges, Bridge{Name: "bridge0", Devs: []string{"eth2", "eth3"}, Mtu: 1300})
	gconfig.Vlans = append(gconfig.Vlans, Vlan{Name: "vlan0", Tag: 100, Parent: "eth0"})
	PutToDataSource(gconfig)
}

func TestGetConfigFromDs(t *testing.T) {
	config := GetConfigFromDs()
	assert.Equal(t, Device{Name: "eth0"}, config.Devices[0])
	assert.Equal(t, Device{Name: "eth1"}, config.Devices[1])
	assert.Equal(t, Device{Name: "eth2"}, config.Devices[2])
	assert.Equal(t, Device{Name: "eth3"}, config.Devices[3])
	assert.Equal(t, Device{Name: "eth4"}, config.Devices[4])
	assert.Equal(t, Device{Name: "eth5"}, config.Devices[5])
	assert.Equal(t, Bond{Name: "bond0", Devs: []string{"eth0", "eth1"}}, config.Bonds[0])
	assert.Equal(t, Bridge{Name: "bridge0", Devs: []string{"eth2", "eth3"}, Mtu: 1300}, config.Bridges[0])
	assert.Equal(t, Vlan{Name: "vlan0", Tag: 100, Parent: "eth0"}, config.Vlans[0])
}

func TestBondAdd(t *testing.T) {
	err := BondAdd("bond1", 0, []string{"eth4"})
	config := GetConfigFromDs()
	assert.Equal(t, Bond{Name: "bond1", Devs: []string{"eth4"}}, config.Bonds[1])
	assert.Nil(t, err)

	err2 := BondAdd("bond0", 0, []string{})
	assert.Error(t, err2, "name alerady exists")

	err3 := BondAdd("bond2", 0, []string{"eth2"})
	assert.Error(t, err3, "dev has alerady been occupied")
}

func TestBondDel(t *testing.T) {
	BondDel("bond0")
	config := GetConfigFromDs()
	assert.NotEqual(t, Bond{Name: "bond0", Devs: []string{"eth0", "eth1"}}, config.Bonds[0])
	assert.Equal(t, Bond{Name: "bond1", Devs: []string{"eth4"}}, config.Bonds[0])
	assert.Equal(t, 1, len(config.Bonds))
}

func TestBridgeAdd(t *testing.T) {
	err := BridgeAdd("bridge1", []string{"eth5"}, 1333)
	config := GetConfigFromDs()
	assert.Equal(t, Bridge{Name: "bridge1", Devs: []string{"eth5"}, Mtu: 1333}, config.Bridges[1])
	assert.Nil(t, err)

	err2 := BridgeAdd("bridge0", []string{}, 1333)
	assert.Error(t, err2, "name alerady exists")

	err3 := BridgeAdd("bridge2", []string{"eth4"}, 1333)
	assert.Error(t, err3, "dev has alerady been occupied")
}

func TestBridgeDel(t *testing.T) {
	BridgeDel("bridge0")
	config := GetConfigFromDs()
	assert.NotEqual(t, Bridge{Name: "bridge0", Devs: []string{"eth2", "eth3"}, Mtu: 1300}, config.Bridges[0])
	assert.Equal(t, Bridge{Name: "bridge1", Devs: []string{"eth5"}, Mtu: 1333}, config.Bridges[0])
	assert.Equal(t, 1, len(config.Bridges))
}

func TestVlanAdd(t *testing.T) {
	err := VlanAdd("vlan1", 200, "eth1")
	config := GetConfigFromDs()
	assert.Equal(t, Vlan{Name: "vlan1", Tag: 200, Parent: "eth1"}, config.Vlans[1])
	assert.Nil(t, err)

	err2 := VlanAdd("vlan0", 0, "")
	assert.Error(t, err2, "name alerady exists")
}

func TestVlanDel(t *testing.T) {
	VlanDel("vlan0")
	config := GetConfigFromDs()
	assert.NotEqual(t, Vlan{Name: "vlan0", Tag: 100, Parent: "eth0"}, config.Vlans[0])
	assert.Equal(t, Vlan{Name: "vlan1", Tag: 200, Parent: "eth1"}, config.Bonds[0])
	assert.Equal(t, 1, len(config.Bonds))
}

func TestApply(t *testing.T) {
	breakNetwork()
	config := GetConfigFromSys()
	bonds := []Bond{{Name: "bond00", Devs: []string{"eth0"}}}
	vlan := []Vlan{{Name: "eth2.300", Parent: "eth2", Tag: 300}}
	bridge := []Bridge{{Name: "br00", Mtu: 1800, Devs: []string{"eth1", "bond00"}}}
	config.Bonds = bonds
	config.Vlans = vlan
	config.Bridges = bridge

	Apply(config)
	sysConfig := GetConfigFromSys()
	assert.Equal(t, "bond00", sysConfig.Bonds[0].Name)
	assert.Equal(t, []string{"eth0"}, sysConfig.Bonds[0].Devs)

	assert.Equal(t, "br00", sysConfig.Bridges[0].Name)
	assert.Equal(t, []string{"eth1", "bond00"}, sysConfig.Bridges[0].Devs)
	assert.Equal(t, 1500, sysConfig.Bridges[0].Mtu)

	assert.Equal(t, "eth2.300", sysConfig.Vlans[0].Name)
	assert.Equal(t, "eth2", sysConfig.Vlans[0].Parent)
	assert.Equal(t, 300, sysConfig.Vlans[0].Tag)
}

func TestGetConfigFromSys(t *testing.T) {
	breakNetwork()
	printLinks(GetConfigFromSys())

	fmt.Println("---down all device---")
	downDevice()
	printLinks(GetConfigFromSys())

	fmt.Println("---del interface---")
	delInterfaces()
	printLinks(GetConfigFromSys())

	fmt.Println("---add bridge---")
	addBridge("testbr", []string{"eth0"}, 1600)
	printLinks(GetConfigFromSys())

	fmt.Println("---add bond---")
	addBond("testbd", []string{"eth1"})
	printLinks(GetConfigFromSys())

	fmt.Println("---add vlan---")
	addVlan("testvlan", "eth2", 900)
	printLinks(GetConfigFromSys())
	breakNetwork()
}
