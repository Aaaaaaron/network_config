package main

import (
	"encoding/json"
	"fmt"
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

const (
	DEVICE = "device" // eg eth0,eth1 etc...
	BOND   = "bond"
	VLAN   = "vlan"
	BRIDGE = "bridge"
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
	Index  int
	Name   string
	IpNets []string
}

type Bond struct {
	Index  int
	Name   string
	Mode   netlink.BondMode
	Devs   []string
	IpNets []string
}

type Bridge struct {
	Index  int
	Name   string
	Devs   []string
	IpNets []string
	Mtu    int
	Stp    string
}

type Vlan struct {
	Index  int
	Name   string
	Tag    int
	Parent string
	IpNets []string
}

func PutToDataSource(config Config) {
	data, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		log.Fatalf("JSON marshaling failed: %s", err)
	}
	//fmt.Printf("%s\n", data)
	DataSource["network"] = string(data)
}
func main() {
	a, _ := netlink.ParseIPNet("1.1.1.1/24")
	fmt.Println(a)
	fmt.Println(a.IP)
	fmt.Println(a.Mask)

}

//func GetFromDataSource() Config {
//	get json
//	convert to object
//return Config{}
//}

func breakNetwork() error {
	if err := downDevice(); err != nil {
		log.WithError(err).Error("down device fail")
		return err
	}

	if err := delInterfaces(); err != nil {
		log.WithError(err).Error("del bond/bridge/vlan fail")
		return err
	}

	if err := setNoIP(); err != nil {
		log.WithError(err).Error("clear ip failed")
		return err
	}
	return nil
}

func grantConfig(link netlink.Link, devMap map[int][]string, config *Config) {
	addrs, _ := netlink.AddrList(link, netlink.FAMILY_ALL)
	var ipNets []string
	for _, addr := range addrs {
		ipNets = append(ipNets, addr.IPNet.String())
	}
	switch link.Type() {
	case DEVICE:
		if deviceLink, ok := link.(*netlink.Device); ok {
			config.Devices = append(config.Devices, Device{deviceLink.Index, deviceLink.Name, ipNets})
		}
	case BOND:
		if bondLink, ok := link.(*netlink.Bond); ok {
			config.Bonds = append(config.Bonds, Bond{bondLink.Index, bondLink.Name, bondLink.Mode, devMap[link.Attrs().Index], ipNets})
		}
	case VLAN:
		if vlanLink, ok := link.(*netlink.Vlan); ok {
			parent, _ := netlink.LinkByIndex(link.Attrs().ParentIndex)
			config.Vlans = append(config.Vlans, Vlan{vlanLink.Index, vlanLink.Name, vlanLink.VlanId, parent.Attrs().Name, ipNets})
		}
	case BRIDGE:
		if bridgeLink, ok := link.(*netlink.Bridge); ok {
			config.Bridges = append(config.Bridges, Bridge{bridgeLink.Index, bridgeLink.Name, devMap[link.Attrs().Index], ipNets, bridgeLink.MTU, ""})
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
		log.WithError(err).Error("get link list from netlink failed")
	}
	return linkList
}

func getHostId() string {
	return "1"
}

// del bond, vlan, bridge, if exists
func delInterfaces() error {
	links := getLinkList()
	for _, link := range links {
		if link.Type() == BOND || link.Type() == VLAN || link.Type() == BRIDGE {
			if err := netlink.LinkDel(link); err != nil {
				return err
			}
		}
	}
	return nil
}

func upAllInterfaces() error {
	links := getLinkList()
	for _, link := range links {
		if err := netlink.LinkSetUp(link); err != nil {
			return err
		}
	}
	return nil
}

// down eth0,eth1 etc.
func downDevice() error {
	links := getLinkList()
	for _, link := range links {
		if link.Type() == DEVICE && link.Attrs().Name != getAdminInterface() {
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

func addBridge(masterName string, dev []string, mtu int) error {
	bri := &netlink.Bridge{netlink.LinkAttrs{Name: masterName, MTU: mtu}}
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
			log.WithError(err).Error("link set master failed.")
			return err
		}
	}
	return nil
}

func setIP(name string, ipNet string) error {
	addr, _ := netlink.ParseAddr(ipNet)
	link, _ := netlink.LinkByName(name)
	if err := netlink.AddrAdd(link, addr); err != nil {
		log.WithError(err).Error("link set ip failed.")
		return err
	}
	return nil
}

func setNoIP() error {
	links := getLinkList()
	for _, link := range links {
		if link.Attrs().Name == getAdminInterface() {
			continue
		}

		addrs, _ := netlink.AddrList(link, netlink.FAMILY_ALL)
		for _, addr := range addrs {
			err := netlink.AddrDel(link, &addr)
			if err != nil {
				//log.WithError(err).Error("link clear ip failed.")
				return err
			}
		}
	}
	return nil
}

func printLinks(config Config) {
	fmt.Println("Host ID:", config.HostId)
	for _, device := range config.Devices {
		fmt.Println(device)
	}
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
