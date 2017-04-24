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
	addBond("bond0", []string{"eth0"})
	addBridge("br0", []string{"eth1"})
	addVlan("eth2.100","eht2",100)
}

func addBond(name string, dev []string) error {
	link := netlink.NewLinkBond(netlink.LinkAttrs{Name: name})
	if err := netlink.LinkAdd(link); err != nil {
		log.Fatal(err)
		return err
	}
	for _, name := range dev {
		dev, _ := netlink.LinkByName(name)
		if err := netlink.LinkSetMasterByIndex(link, dev.Attrs().Index); err != nil {
			log.Fatal(err)
			return err
		}
	}
	return nil
}

func addBridge(name string, dev []string) error {
	link := &netlink.Bridge{netlink.LinkAttrs{Name: "foo", MTU: 1400}}
	if err := netlink.LinkAdd(link); err != nil {
		log.Fatal(err)
		return err
	}
	for _, name := range dev {
		dev, _ := netlink.LinkByName(name)
		if err := netlink.LinkSetMasterByIndex(link, dev.Attrs().Index); err != nil {
			log.Fatal(err)
			return err
		}
	}
	return nil
}

func addVlan(name string, parent string, id int) error {
	par, _ := netlink.LinkByName(parent)
	link := &netlink.Vlan{netlink.LinkAttrs{Name: "bar", ParentIndex: par.Attrs().Index}, id}
	if err := netlink.LinkAdd(link); err != nil {
		log.Fatal(err)
		return err
	}
	return nil
}

func TestGetDevMap(t *testing.T) {

	assert.Equal(t, 1, 1, "they should be equal")
	//assert.NotNil(t, err, "error should not be nil")
}
