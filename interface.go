package main

import (
	"fmt"
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

func init() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)
}

type Config struct {
	HostId  string
	Devices []Device
	Bonds   []Bond
	Bridges []Bridge
	Vlans   []Vlan
	//后期想到上面新的配置项可以加在这里
}

type Device struct {
	Index int
	Name  string
	Addr  []netlink.Addr
}

type Bond struct {
	Index int
	Name  string
	Mode  netlink.BondMode
	Dev   []string
	Addr  []netlink.Addr
}

type Bridge struct {
	Index int
	Name  string
	Dev   []string
	Addr  []netlink.Addr
	Mtu   int
	Stp   string
}

type Vlan struct {
	Index  int
	Name   string
	Tag    int
	Parent string
	Addr   []netlink.Addr
}

func main() {
	config := GetSysConfig()
	fmt.Println(config)
}

func apply() {
	if err := breakNetwork(); err != nil {

	}
}

func breakNetwork() error {
	//admin := getAdminInteface()
	return nil
}

//func getAdminInteface() string {
//	return "eth3"
//}

func GetSysConfig() Config {
	var config Config
	links := getLinkList()
	devMap := getDevMap(links)
	for _, link := range links {
		grantConfig(link, devMap, &config)
	}
	return config
}

func grantConfig(link netlink.Link, devMap map[int][]string, config *Config) {
	addr, _ := netlink.AddrList(link, netlink.FAMILY_ALL)
	switch link.Type() {
	case "device":
		if deviceLink, ok := link.(*netlink.Device); ok {
			config.Devices = append(config.Devices, Device{deviceLink.Index, deviceLink.Name, addr})
		}
	case "bond":
		if bondLink, ok := link.(*netlink.Bond); ok {
			config.Bonds = append(config.Bonds, Bond{bondLink.Index, bondLink.Name, bondLink.Mode, devMap[link.Attrs().Index], addr})
		}
	case "vlan":
		if vlanLink, ok := link.(*netlink.Vlan); ok {
			parent, _ := netlink.LinkByIndex(link.Attrs().ParentIndex)
			config.Vlans = append(config.Vlans, Vlan{vlanLink.Index, vlanLink.Name, vlanLink.VlanId, parent.Attrs().Name, addr})
		}
	case "bridge":
		if bridgeLink, ok := link.(*netlink.Bridge); ok {
			config.Bridges = append(config.Bridges, Bridge{bridgeLink.Index, bridgeLink.Name, devMap[link.Attrs().Index], addr, bridgeLink.MTU, ""})
		}
	}
}

// get the interface's dev,eg: 5:eth0 eth1,5 is the bond0's index
func getDevMap(links []netlink.Link) map[int][]string {
	m := make(map[int][]string)
	for _, link := range links {
		if masterIndex := link.Attrs().MasterIndex; masterIndex != 0 {
			m[masterIndex] = append(m[masterIndex], link.Attrs().Name)
		}
	}
	return m
}

func getLinkList() []netlink.Link { // link represent all network interface
	linkList, err := netlink.LinkList()
	if err != nil {
		log.Fatalf("get link list from netlink failed: %s", err)
	}
	return linkList
}

func getHostId() string {
	return "1"
}
