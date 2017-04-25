package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func init() {
	initNetwork()
}

func initNetwork() {
	links := getLinkList()
	downDevice(links)
	delInterface(links)
	//addBridge("br0", []string{"eth1"})
	//addBond("bond0", []string{"eth0"})
	//addVlan("eth2.100", "eht2", 100)
}

func TestGetDevMap(t *testing.T) {
	addBridge("br1", []string{"eth0", "eth1"})
	m := getDevMap(getLinkList())
	assert.Equal(t, []string{"eth0", "eth1"}, m[getIndexByName("br1")], "they should be equal")
	//assert.NotNil(t, err, "error should not be nil")
}
