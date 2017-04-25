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
	printLinks(GetSysConfig())
	fmt.Println("---down all device---")
	downDevice(getLinkList())
	printLinks(GetSysConfig())

	fmt.Println("---del interface---")
	delInterface(getLinkList())
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

//func testVlan() error {
//	parent := &netlink.Dummy{netlink.LinkAttrs{Name: "foo"}}
//	if err := netlink.LinkAdd(parent); err != nil {
//		fmt.Println("add parent error")
//		return err
//	}
//	link := &netlink.Vlan{netlink.LinkAttrs{Name: "bar", ParentIndex: parent.Attrs().Index}, 900}
//	if err := netlink.LinkAdd(link); err != nil {
//		fmt.Println("add parent error")
//		return err
//	}
//	return nil
//}

// del bond, vlan, bridge, if exists
func delInterface(links []netlink.Link) error {
	for _, link := range links {
		if link.Type() == "bond" || link.Type() == "vlan" || link.Type() == "bridge" {
			if err := netlink.LinkDel(link); err != nil {
				return err
			}
		}
	}
	return nil
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

func addBond(masterName string, dev []string) error {
	bond := netlink.NewLinkBond(netlink.LinkAttrs{Name: masterName})
	if err := netlink.LinkAdd(bond); err != nil {
		return err
	}
	if err := setMaster(masterName, dev); err != nil {
		return err
	}
	return nil
}

func addBridge(masterName string, dev []string) error {
	bri := &netlink.Bridge{netlink.LinkAttrs{Name: masterName, MTU: 1400}}
	if err := netlink.LinkAdd(bri); err != nil {
		return err
	}
	if err := setMaster(masterName, dev); err != nil {
		return err
	}
	return nil
}

func addVlan(name string, parent string, id int) error {
	parentIndex := getIndexByName(parent)
	vlan := &netlink.Vlan{netlink.LinkAttrs{Name: name, ParentIndex: parentIndex}, id}
	if err := netlink.LinkAdd(vlan); err != nil {
		return err
	}
	return nil
}

func getIndexByName(name string) int {
	link, err := netlink.LinkByName(name)
	if err != nil {
		log.Fatal(err)
	}
	return link.Attrs().Index
}

func setMaster(masterName string, dev []string) error {
	for _, devName := range dev {
		slave, err := netlink.LinkByName(devName)
		if err != nil {
			log.Fatal(err)
		}
		masterID := getIndexByName(masterName)
		if err := netlink.LinkSetMasterByIndex(slave, masterID); err != nil {
			return err
		}
	}
	return nil
}

func printLinks(config Config) {
	fmt.Println("Host ID:", config.HostId)
	for _, bond := range config.Bonds {
		fmt.Println(bond)
	}
	for _, bridge := range config.Bridges {
		fmt.Println(bridge)
	}
	for _, vlan := range config.Vlans {
		fmt.Println(vlan)
	}
}
