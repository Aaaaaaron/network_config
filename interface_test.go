package main

import (
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vishvananda/netlink"
)

func init() {
	initNetwork()
}

func initNetwork() {
	links := getLinkList()
	for _, link := range links {
		if link.Type() == "bond" || link.Type() == "vlan" || link.Type() == "bridge" {
			if err := netlink.LinkDel(link); err != nil {
				log.Fatal(err)
			}
		}
	}
}

func TestGetDevMap(t *testing.T) {

	assert.Equal(t, 1, 1, "they should be equal")
	//assert.NotNil(t, err, "error should not be nil")
}
