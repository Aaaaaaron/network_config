package main

import (
	"encoding/json"
	"errors"
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

func main() {
	var gconfig Config
	gconfig.Devices = append(gconfig.Devices, Device{Name: "eth0"})
	gconfig.Devices = append(gconfig.Devices, Device{Name: "eth1"})
	gconfig.Devices = append(gconfig.Devices, Device{Name: "eth2"})
	gconfig.Devices = append(gconfig.Devices, Device{Name: "eth3"})
	gconfig.Devices = append(gconfig.Devices, Device{Name: "eth4"})
	gconfig.Devices = append(gconfig.Devices, Device{Name: "eth5"})
	gconfig.Bonds = append(gconfig.Bonds, Bond{Name: "bond0", Devs: []string{"eth0", "eth1"}})
	gconfig.Bridges = append(gconfig.Bridges, Bridge{Name: "bridge0", Devs: []string{"eth2", "eth3"}, Mtu: 1300})
	gconfig.Vlans = append(gconfig.Vlans, Vlan{Name: "vlan0", Tag: 100, Parent: "eth0"})
	PutToDataSource(gconfig)
	AssignIP("eth0", "1.1.1.1/24")
	AssignIP("eth0", "2.2.2.2/24")
	AssignIP("eth0", "3.3.3.3/24")
	BondAdd("bond9", 0, []string{})
	addBond("bond9", []string{})
	AssignIP("bond9", "33.33.33.33/24")
	config := GetConfigFromDs()
	fmt.Println(config)

	//ip1 := "1.1.1.1/24"
	//ip2 := "3.3.3.3/24"
	//breakNetwork()
	//addBond("bond0", []string{"eth0", "eth1"})
	//setIP("eth2", ip1)
	//setIP("bond0", ip2)
	//fmt.Println(GetConfigFromSys())
	//breakNetwork()

	//var gconfig Config
	//gconfig.Devices = append(gconfig.Devices, Device{Name: "eth0"})
	//gconfig.Devices = append(gconfig.Devices, Device{Name: "eth1"})
	//gconfig.Devices = append(gconfig.Devices, Device{Name: "eth2"})
	//gconfig.Devices = append(gconfig.Devices, Device{Name: "eth3"})
	//gconfig.Devices = append(gconfig.Devices, Device{Name: "eth4"})
	//gconfig.Devices = append(gconfig.Devices, Device{Name: "eth5"})
	//gconfig.Bonds = append(gconfig.Bonds, Bond{Name: "bond0", Devs: []string{"eth0", "eth1"}})
	//gconfig.Bridges = append(gconfig.Bridges, Bridge{Name: "bridge0", Devs: []string{"eth2", "eth3"}, Mtu: 1300})
	//gconfig.Vlans = append(gconfig.Vlans, Vlan{Name: "vlan0", Tag: 100, Parent: "eth0"})
	//PutToDataSource(gconfig)
	//BondAdd("adsf", 0, nil)
	//fmt.Println(GetConfigFromDs())
	//
	//PutToDataSource(GetConfigFromSys())
	//fmt.Println(DataSource["network"])
	//fmt.Println(isLinkAlreadyExists("bond0", gconfig))
	//hasDevBeenOccupied([]string{"eth0"}, gconfig)
}

func GetConfigFromSys() Config {
	var config Config
	links := getLinkList()
	devMap := getDevMap(links)
	for _, link := range links {
		grantConfig(link, devMap, &config)
	}
	return config
}

func GetConfigFromDs() Config {
	var config Config
	json.Unmarshal(([]byte)(DataSource["network"]), &config)
	return config
}

func BridgeAdd(name string, dev []string, mtu int) error {
	// 要根据数据源里存的配置的进行校验 而不是从系统中取到的配置
	userConfig := GetConfigFromDs()
	if isLinkAlreadyExists(name, userConfig) {
		//log.Error("interface named " + name + " alerady exists")
		return errors.New("interface named " + name + " alerady exists")
	}
	if hasDevBeenOccupied(dev, userConfig) {
		//log.Error("dev has alerady been occupied")
		return errors.New("dev has alerady been occupied")
	}
	bridges := GetConfigFromDs().Bridges
	bridges = append(bridges, Bridge{Name: name, Devs: dev, Mtu: mtu})
	userConfig.Bridges = bridges
	PutToDataSource(userConfig)
	return nil
}

func BridgeDel(name string) {
	userConfig := GetConfigFromDs()
	bridges := userConfig.Bridges
	for i, bri := range bridges {
		if bri.Name == name {
			bridges = append(bridges[:i], bridges[i+1:]...)
		}
	}
	userConfig.Bridges = bridges
	PutToDataSource(userConfig)
}

func BridgeUpdate(name string, dev []string, mtu int) error { // can not modify name
	BridgeDel(name)
	return BridgeAdd(name, dev, mtu)
}

func BondAdd(name string, mode int, dev []string) error {
	userConfig := GetConfigFromDs()
	if isLinkAlreadyExists(name, userConfig) {
		//log.Error("interface named " + name + " alerady exists")
		return errors.New("interface named " + name + " alerady exists")
	}
	if hasDevBeenOccupied(dev, userConfig) {
		//log.Error("dev has alerady been occupied")
		return errors.New("dev has alerady been occupied")
	}

	bonds := GetConfigFromDs().Bonds
	bonds = append(bonds, Bond{Name: name, Mode: netlink.BondMode(mode), Devs: dev})
	userConfig.Bonds = bonds
	PutToDataSource(userConfig)
	return nil
}

func BondDel(name string) {
	userConfig := GetConfigFromDs()
	bonds := userConfig.Bonds
	for i, bri := range bonds {
		if bri.Name == name {
			bonds = append(bonds[:i], bonds[i+1:]...)
		}
	}
	userConfig.Bonds = bonds
	PutToDataSource(userConfig)
}

func BondUpdate(name string, mode int, dev []string) error { // can not modify name
	BondDel(name)
	return BondAdd(name, mode, dev)
}

func VlanAdd(name string, tag int, parent string) error {
	userConfig := GetConfigFromDs()
	if isLinkAlreadyExists(name, userConfig) {
		//log.Error("interface named " + name + " alerady exists")
		return errors.New("interface named " + name + " alerady exists")
	}
	vlans := GetConfigFromDs().Vlans
	vlans = append(vlans, Vlan{Name: name, Tag: tag, Parent: parent})
	userConfig.Vlans = vlans
	PutToDataSource(userConfig)
	return nil
}

func VlanDel(name string) {
	userConfig := GetConfigFromDs()
	vlans := userConfig.Vlans
	for i, v := range vlans {
		if v.Name == name {
			vlans = append(vlans[:i], vlans[i+1:]...)
		}
	}
	userConfig.Vlans = vlans
	PutToDataSource(userConfig)
}

func VlanUpdate(name string, tag int, parent string) error { // can not modify name
	VlanDel(name)
	return VlanAdd(name, tag, parent)
}

func AssignIP(name string, ipNet string) error {
	_, err := netlink.ParseAddr(ipNet)
	if err != nil {
		//log.WithError(err).Error("ip net format error, parse failed")
		return errors.New("ip net :" + ipNet + "format error, parse failed")
	}
	userConfig := GetConfigFromDs()
	device := userConfig.Devices
	bonds := userConfig.Bonds
	for _, d := range device {
		ipNets := d.IpNets
		if d.Name == name {
			ipNets = append(ipNets, ipNet)
		}
		d.IpNets = ipNets
	}
	userConfig.Devices = device
	for _, b := range bonds {
		ipNets := b.IpNets
		if b.Name == name {
			ipNets = append(ipNets, ipNet)
		}
		b.IpNets = ipNets
	}
	userConfig.Bonds = bonds
	PutToDataSource(userConfig)
	return nil
}

func DelIP(name string, ipNet string) {
	userConfig := GetConfigFromDs()
	device := userConfig.Devices
	bonds := userConfig.Bonds
	for _, d := range device {
		ipNets := d.IpNets
		if d.Name == name {
			for i, ipnet := range ipNets {
				if ipnet == ipNet {
					ipNets = append(ipNets[:i], ipNets[i+1:]...)
				}
			}
		}
		d.IpNets = ipNets
	}
	userConfig.Devices = device
	for _, b := range bonds {
		ipNets := b.IpNets
		if b.Name == name {
			for i, ipnet := range ipNets {
				if ipnet == ipNet {
					ipNets = append(ipNets[:i], ipNets[i+1:]...)
				}
			}
		}
		b.IpNets = ipNets
	}
	userConfig.Bonds = bonds
	PutToDataSource(userConfig)
}

//not thread safe
func Apply(config Config) error {
	if err := breakNetwork(); err != nil {
		log.WithError(err).Error("break network failed")
		return err
	}

	// assign device's ip,eg assign ip:192.168.3.3 ,mask:255.255.255.0 to eth0
	for _, device := range config.Devices {
		if device.Name == getAdminInterface() {
			continue
		}

		if ipNets := device.IpNets; len(ipNets) > 0 {
			for _, ipNet := range ipNets {
				if err := setIP(device.Name, ipNet); err != nil {
					log.WithError(err).Error("device add ip failed")
					return err
				}
			}
		}
	}

	for _, bond := range config.Bonds {
		if err := addBond(bond.Name, bond.Devs); err != nil {
			log.WithError(err).Error("add bond failed")
			return err
		}
		// assign bond's ip,eg assign ip:192.168.3.3 ,mask:255.255.255.0 to bond0
		if ipNets := bond.IpNets; len(ipNets) > 0 {
			for _, ipNet := range ipNets {
				if err := setIP(bond.Name, ipNet); err != nil {
					log.WithError(err).Error("bond add ip failed")
					return err
				}
			}
		}
	}

	for _, vlan := range config.Vlans {
		if err := addVlan(vlan.Name, vlan.Parent, vlan.Tag); err != nil {
			log.WithError(err).Error("add vlan failed")
			return err
		}
	}

	for _, bridge := range config.Bridges {
		if err := addBridge(bridge.Name, bridge.Devs, 1600); err != nil {
			log.WithError(err).Error("add bridge failed")
			return err
		}
	}
	return nil
}

func hasDevBeenOccupied(devs []string, config Config) bool {
	for _, dev := range devs {
		for _, b := range config.Bonds {
			for _, d := range b.Devs {
				if d == dev {
					return true
				}
			}
		}
		for _, br := range config.Bridges {
			for _, d := range br.Devs {
				if d == dev {
					return true
				}
			}
		}
	}
	return false
}

func isLinkAlreadyExists(name string, config Config) bool {
	for _, de := range config.Devices {
		if de.Name == name {
			return true
		}
	}
	for _, b := range config.Bonds {
		if b.Name == name {
			return true
		}
	}
	for _, v := range config.Vlans {
		if v.Name == name {
			return true
		}
	}
	for _, br := range config.Bridges {
		if br.Name == name {
			return true
		}
	}
	return false
}
