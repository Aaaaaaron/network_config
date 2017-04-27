package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetDevMap(t *testing.T) {
	breakNetwork()
	addBridge("br1", []string{"eth0", "eth1"}, 1600)
	m := getDevMap(getLinkList())
	assert.Equal(t, []string{"eth0", "eth1"}, m[getIndexByName("br1")], "they should be equal")
	//assert.NotNil(t, err, "error should not be nil")
	breakNetwork()
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
	assert.Equal(t, 300, sysConfig.Vlans[0].Tag )
}

func TestAddBond(t *testing.T) {
	breakNetwork()
	addBond("bond0", []string{"eth1"})
	bond := GetConfigFromSys().Bonds[0]
	assert.Equal(t, "bond0", bond.Name)
	assert.Equal(t, []string{"eth1"}, bond.Devs)
	breakNetwork()
}

func TestAddBridge(t *testing.T) {
	breakNetwork()
	addBridge("br1", []string{"eth0", "eth1"}, 1600)
	bridge := GetConfigFromSys().Bridges[0]
	assert.Equal(t, "br1", bridge.Name)
	assert.Equal(t, []string{"eth0", "eth1"}, bridge.Devs)
	assert.Equal(t, 1500, bridge.Mtu)
	breakNetwork()
}

func TestAddVlan(t *testing.T) {
	breakNetwork()
	addVlan("vlan0", "eth2", 300)
	vlan := GetConfigFromSys().Vlans[0]
	assert.Equal(t, "vlan0", vlan.Name)
	assert.Equal(t, "eth2", vlan.Parent)
	assert.Equal(t, 300, vlan.Tag)
	breakNetwork()
}

func TestGetSysConfig(t *testing.T) {
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
