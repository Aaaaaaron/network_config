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
	Mode   int
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

func PutToDataSource(config Config) error {
	data, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		log.WithError(err).Error("Put to database failed cuz convert json failed")
		return err
	}
	DataSource["network"] = string(data)
	return nil
}

//not thread safe
func Apply(config Config) error {
	if err := breakNetwork(); err != nil {
		log.WithError(err).Error("Break network failed")
		return err
	}

	if err := setDevice(config.Devices); err != nil {
		log.WithError(err).Error("Set device fail")
		return err
	}

	if err := buildBond(config.Bonds); err != nil {
		log.WithError(err).Error("Build bond fail")
		return err
	}

	if err := buildVlan(config.Vlans); err != nil {
		log.WithError(err).Error("Build vlan fail")
		return err
	}

	if err := buildBridge(config.Bridges); err != nil {
		log.WithError(err).Error("Build bridge fail")
		return err
	}

	return nil
}

func setDevice(devices []Device) error {
	// assign device's Ip,eg assign Ip:192.168.3.3 ,mask:255.255.255.0 to eth0
	for _, device := range devices {
		// ignore admin interface and lo
		if device.Name == getAdminInterface() || device.Name == "lo" {
			continue
		}

		// set device's IP
		if ipNets := device.IpNets; len(ipNets) > 0 {
			for _, ipNet := range ipNets {
				if err := setIP(device.Name, ipNet); err != nil {
					log.WithError(err).Error("Device " + device.Name + "add Ip failed")
					return err
				}
			}
		}
	}
	return nil
}

func buildBond(bonds []Bond) error {
	for _, bond := range bonds {
		if err := addBond(bond.Name, bond.Mode, bond.Devs); err != nil {
			log.WithError(err).Error("add bond failed")
			return err
		}
		// assign bond's Ip,eg assign Ip:192.168.3.3 ,mask:255.255.255.0 to bond0
		if ipNets := bond.IpNets; len(ipNets) > 0 {
			for _, ipNet := range ipNets {
				if err := setIP(bond.Name, ipNet); err != nil {
					log.WithError(err).Error("bond add Ip failed")
					return err
				}
			}
		}
	}
	return nil
}

func buildVlan(vlans []Vlan) error {
	for _, vlan := range vlans {
		if err := addVlan(vlan.Name, vlan.Parent, vlan.Tag); err != nil {
			log.WithError(err).Error("add vlan failed")
			return err
		}
	}
	return nil
}

func buildBridge(bridges []Bridge) error {
	for _, bridge := range bridges {
		if err := addBridge(bridge.Name, bridge.Devs, 1600); err != nil {
			log.WithError(err).Error("add bridge failed")
			return err
		}
	}
	return nil
}

func breakNetwork() error {
	if err := downDevice(); err != nil {
		log.WithError(err).Error("Break network failed, down device fail")
		return err
	}

	if err := delInterfaces(); err != nil {
		log.WithError(err).Error("Break network failed, del bond/bridge/vlan fail")
		return err
	}

	if err := setNoIP(); err != nil {
		log.WithError(err).Error("Break network failed, clear Ip failed")
		return err
	}
	return nil
}

func grantConfig(link netlink.Link, devMap map[int][]string, config *Config) error {
	var ipNets []string
	addrs, err := netlink.AddrList(link, netlink.FAMILY_ALL)
	if err != nil {
		log.WithError(err).Error("Get config from database failed")
		return err
	}
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
			config.Bonds = append(config.Bonds, Bond{bondLink.Index, bondLink.Name, int(bondLink.Mode), devMap[link.Attrs().Index], ipNets})
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
	return nil
}

// get the interface's dev,eg: 5:eth0 eth1,5 is the bond0's index
func getSlaveList(links []netlink.Link) map[int][]string {
	m := make(map[int][]string)
	for _, link := range links {
		if masterIndex := link.Attrs().MasterIndex; masterIndex != 0 {
			m[masterIndex] = append(m[masterIndex], link.Attrs().Name)
		}
	}
	return m
}

func getHostId() string {
	return "1"
}

// del bond, vlan, bridge, if exists
func delInterfaces() error {
	links, err := netlink.LinkList()
	if err != nil {
		log.WithError(err).Error(" Get link list failed")
		return err
	}

	for _, link := range links {
		if link.Type() == BOND || link.Type() == VLAN || link.Type() == BRIDGE {
			if err := netlink.LinkDel(link); err != nil {
				log.WithError(err).Error(" Del " + link.Attrs().Name + " link failed")
				return err
			}
		}
	}
	return nil
}

// down devices like eth0,eth1 etc.
func downDevice() error {
	links, err := netlink.LinkList()
	if err != nil {
		log.WithError(err).Error("Get link list failed")
		return err
	}

	for _, link := range links {
		if link.Type() == DEVICE && link.Attrs().Name != getAdminInterface() && link.Attrs().Name != "lo" {
			if err := netlink.LinkSetDown(link); err != nil {
				log.WithError(err).Error("Down " + link.Attrs().Name + " link failed")
				return err
			}
		}
	}
	return nil
}

func upAllLinks() error {
	links, err := netlink.LinkList()
	if err != nil {
		log.WithError(err).Error("Get link list failed")
		return err
	}

	for _, link := range links {
		if err := netlink.LinkSetUp(link); err != nil {
			log.WithError(err).Error("Up " + link.Attrs().Name + " link failed")
			return err
		}
	}
	return nil
}

func getAdminInterface() string {
	return "eth3"
}

func addBond(masterName string, mode int, dev []string) error {
	bond := netlink.NewLinkBond(netlink.LinkAttrs{Name: masterName})
	bond.Mode = netlink.BondMode(mode)
	if err := netlink.LinkAdd(bond); err != nil {
		log.WithError(err).Error("Add bond " + masterName + " fail ")
		return err
	}
	if err := addSlave(masterName, dev); err != nil {
		log.WithError(err).Error("Bond " + masterName + " add slave fail")
		return err
	}
	return nil
}

func addBridge(masterName string, dev []string, mtu int) error {
	bri := &netlink.Bridge{LinkAttrs: netlink.LinkAttrs{Name: masterName, MTU: mtu}}
	if err := netlink.LinkAdd(bri); err != nil {
		log.WithError(err).Error("Add bridge " + masterName + " fail ")
		return err
	}
	if err := addSlave(masterName, dev); err != nil {
		log.WithError(err).Error("Bridge " + masterName + " add slave fail")
		return err
	}
	return nil
}

func addVlan(name string, parent string, id int) error {
	parentIndex, err := getIndexByName(parent)
	if err != nil {
		log.WithError(err).Error("get parent device " + parent + "'s index fail ")
		return err
	}

	vlan := &netlink.Vlan{LinkAttrs: netlink.LinkAttrs{Name: name, ParentIndex: parentIndex}, VlanId: id}
	if err := netlink.LinkAdd(vlan); err != nil {
		log.WithError(err).Error("Add vlan " + name + " fail ")
		return err
	}
	return nil
}

func getIndexByName(name string) (int, error) {
	link, err := netlink.LinkByName(name)
	if err != nil {
		log.WithError(err).Error("link set master failed.")
		return -1, err
	}
	return link.Attrs().Index, nil
}

func addSlave(masterName string, dev []string) error {
	for _, devName := range dev {
		slave, err := netlink.LinkByName(devName)
		if err != nil {
			log.WithError(err).Error("Get slave link " + slave.Attrs().Name + " failed")
			return err
		}

		masterID, err := getIndexByName(masterName)
		if err != nil {
			log.WithError(err).Error("get master " + masterName + "'s index fail ")
			return err
		}

		if err := netlink.LinkSetMasterByIndex(slave, masterID); err != nil {
			log.WithError(err).Error("link set master failed.")
			return err
		}
	}
	return nil
}

// only can assign Ip to devices and bonds
func setIP(name string, ipNet string) error {
	addr, err := netlink.ParseAddr(ipNet)
	if err != nil {
		log.WithError(err).Error("parse addr " + ipNet + " failed")
		return err
	}

	link, err := netlink.LinkByName(name)
	if err != nil {
		log.WithError(err).Error("Get link " + name + " failed")
		return err
	}

	if err := netlink.AddrAdd(link, addr); err != nil {
		log.WithError(err).Error("link " + name + " set Ip" + ipNet + " failed.")
		return err
	}
	return nil
}

func setNoIP() error {
	links, err := netlink.LinkList()
	if err != nil {
		log.WithError(err).Error("Get link list failed")
		return err
	}

	for _, link := range links {
		if link.Attrs().Name == getAdminInterface() || link.Attrs().Name == "lo" {
			continue
		}

		addrs, err := netlink.AddrList(link, netlink.FAMILY_ALL)
		if err != nil {
			log.WithError(err).Error("Get link " + link.Attrs().Name + "'s address  failed")
			return err
		}

		for _, addr := range addrs {
			err := netlink.AddrDel(link, &addr)
			if err != nil {
				log.WithError(err).Error(" Link " + link.Attrs().Name + "clear Ip failed.")
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
