package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vishvananda/netlink"
)

func TestGetDevMap(t *testing.T) {
	breakNetwork()
	addBridge("br1", []string{"eth0", "eth1"}, 1600)
	links, _ := netlink.LinkList()
	m := getSlaveList(links)
	masterIndex, _ := getIndexByName("br1")
	assert.Equal(t, []string{"eth0", "eth1"}, m[masterIndex], "they should be equal")
	//assert.NotNil(t, err, "error should not be nil")
	breakNetwork()
}

func TestAddBond(t *testing.T) {
	breakNetwork()
	addBond("bond0", 4, []string{"eth1"})
	sysConfig, _ := GetConfigFromSys()
	bond := sysConfig.Bonds[0]
	assert.Equal(t, "bond0", bond.Name)
	assert.Equal(t, []string{"eth1"}, bond.Devs)
	assert.Equal(t, 4, int(bond.Mode))

	breakNetwork()
}

func TestAddBridge(t *testing.T) {
	breakNetwork()
	addBridge("br1", []string{"eth0", "eth1"}, 1600)
	sysConfig, _ := GetConfigFromSys()
	bridge := sysConfig.Bridges[0]
	assert.Equal(t, "br1", bridge.Name)
	assert.Equal(t, []string{"eth0", "eth1"}, bridge.Devs)
	assert.Equal(t, 1500, bridge.Mtu)
	breakNetwork()
}

func TestAddVlan(t *testing.T) {
	breakNetwork()
	addVlan("vlan0", "eth2", 300)
	sysConfig, _ := GetConfigFromSys()
	vlan := sysConfig.Vlans[0]
	assert.Equal(t, "vlan0", vlan.Name)
	assert.Equal(t, "eth2", vlan.Parent)
	assert.Equal(t, 300, vlan.Tag)
	breakNetwork()
}

func TestSetIP(t *testing.T) {
	ip1 := "1.1.1.1/24"
	ip2 := "3.3.3.3/24"
	breakNetwork()
	addBond("bond0", 3, []string{"eth0", "eth1"})
	setIP("eth2", ip1)
	setIP("bond0", ip2)
	sysConfig, _ := GetConfigFromSys()
	fmt.Println(sysConfig)
	breakNetwork()
}

func TestApply(t *testing.T) {
	breakNetwork()
	config, _ := GetConfigFromSys()
	bonds := []Bond{{Name: "bond00", Devs: []string{"eth0"}}}
	vlan := []Vlan{{Name: "eth2.300", Parent: "eth2", Tag: 300}}
	bridge := []Bridge{{Name: "br00", Mtu: 1800, Devs: []string{"eth1", "bond00"}}}
	config.Bonds = bonds
	config.Vlans = vlan
	config.Bridges = bridge

	Apply(config)
	sysConfig, _ := GetConfigFromSys()
	assert.Equal(t, "bond00", sysConfig.Bonds[0].Name)
	assert.Equal(t, []string{"eth0"}, sysConfig.Bonds[0].Devs)

	assert.Equal(t, "br00", sysConfig.Bridges[0].Name)
	assert.Equal(t, []string{"eth1", "bond00"}, sysConfig.Bridges[0].Devs)
	assert.Equal(t, 1500, sysConfig.Bridges[0].Mtu)

	assert.Equal(t, "eth2.300", sysConfig.Vlans[0].Name)
	assert.Equal(t, "eth2", sysConfig.Vlans[0].Parent)
	assert.Equal(t, 300, sysConfig.Vlans[0].Tag)
	breakNetwork()
}
