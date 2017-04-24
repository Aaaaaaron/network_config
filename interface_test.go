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
	downDevice(links)
	delInterface(links)
	upDevice(links)
	addBridge("br0", []string{"eth1"})
	addBond("bond0", []string{"eth0"})
	addVlan("eth2.100", "eht2", 100)
}

func delInterface(links []netlink.Link) {
	for _, link := range links {
		if link.Type() == "bond" || link.Type() == "vlan" || link.Type() == "bridge" {
			if err := netlink.LinkDel(link); err != nil {
				log.Fatal(err)
			}
		}
	}
}

func upDevice(links []netlink.Link) error {
	for _, link := range links {
		if link.Type() == "device" {
			if err := netlink.LinkSetUp(link); err != nil {
				return err
			}
		}
	}
	return nil
}

func downDevice(links []netlink.Link) error {
	for _, link := range links {
		if link.Type() == "device" && link.Attrs().Name != getAdminInterface() {
			if err := netlink.LinkSetDown(link); err != nil {
				return err
			}
		}
	}
	return nil
}

func getAdminInterface() string {
	return "eth3"
}

func addBond(name string, dev []string) error {
	link := netlink.NewLinkBond(netlink.LinkAttrs{Name: name})
	if err := netlink.LinkAdd(link); err != nil {
		log.Fatal(err)
		return err
	}
	for _, devName := range dev {
		link, _ := netlink.LinkByName(devName)
		master, _ := netlink.LinkByName(name)
		if err := netlink.LinkSetMasterByIndex(&link, master.Attrs().Index); err != nil {
			log.Fatal(err)
			return err
		}
	}
	return nil
}

func addBridge(name string, dev []string) error {
	link := &netlink.Bridge{netlink.LinkAttrs{Name: name, MTU: 1400}}
	if err := netlink.LinkAdd(link); err != nil {
		log.Fatal(err)
		return err
	}
	for _, devName := range dev {
		link, _ := netlink.LinkByName(devName)
		master, _ := netlink.LinkByName(name)
		if err := netlink.LinkSetMasterByIndex(&link, master.Attrs().Index); err != nil {
			log.Fatal(err)
			return err
		}
	}
	return nil
}

func addVlan(name string, parent string, id int) error {
	par, _ := netlink.LinkByName(parent)
	link := &netlink.Vlan{netlink.LinkAttrs{Name: name, ParentIndex: par.Attrs().Index}, id}
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
