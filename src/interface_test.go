package main

import (
	"fmt"
	"net"
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

func TestSetIP(t *testing.T) {
	ipNet1 := IPNet{IP: net.ParseIP("1.1.1.1"), mask: net.IPMask(net.ParseIP("255.255.255.0"))}
	ipNet2 := IPNet{IP: net.ParseIP("3.3.3.3"), mask: net.IPMask(net.ParseIP("255.255.255.0"))}
	breakNetwork()
	addBond("bond0", []string{"eth0", "eth1"})
	setIP("eth2", ipNet1)
	setIP("bond0", ipNet2)
	fmt.Println(GetConfigFromSys())
	//breakNetwork()
}
