package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"fmt"
)

func TestGetDevMap(t *testing.T) {
	breakNetwork()
	addBridge("br1", []string{"eth0", "eth1"})
	m := getDevMap(getLinkList())
	assert.Equal(t, []string{"eth0", "eth1"}, m[getIndexByName("br1")], "they should be equal")
	//assert.NotNil(t, err, "error should not be nil")
	breakNetwork()
}

func TestAddBond(t *testing.T) {
	breakNetwork()
	addBond("bond0", []string{"eth1"})
	bond := GetSysConfig().Bonds[0]
	assert.Equal(t, "bond0", bond.Name)
	assert.Equal(t, []string{"eth1"}, bond.Dev)
	breakNetwork()
}

func TestAddBridge(t *testing.T) {
	breakNetwork()
	addBridge("br1", []string{"eth0", "eth1"})
	bridge := GetSysConfig().Bridges[0]
	assert.Equal(t, "br1", bridge.Name)
	assert.Equal(t, []string{"eth0", "eth1"}, bridge.Dev)
	breakNetwork()
}

func TestAddVlan(t *testing.T) {
	breakNetwork()
	addVlan("vlan0", "eth2", 300)
	vlan := GetSysConfig().Vlans[0]
	assert.Equal(t, "vlan0", vlan.Name)
	assert.Equal(t, "eth2", vlan.Parent)
	breakNetwork()
}

func TestGetSysConfig(t *testing.T) {
	breakNetwork()
	printLinks(GetSysConfig())

	fmt.Println("---down all device---")
	downDevice()
	printLinks(GetSysConfig())

	fmt.Println("---del interface---")
	delInterfaces()
	printLinks(GetSysConfig())

	fmt.Println("---add bridge---")
	addBridge("testbr", []string{"eth0"})
	printLinks(GetSysConfig())

	fmt.Println("---add bond---")
	addBond("testbd", []string{"eth1"})
	printLinks(GetSysConfig())

	fmt.Println("---add vlan---")
	addVlan("testvlan", "eth2", 900)
	printLinks(GetSysConfig())
	breakNetwork()
}