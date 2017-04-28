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
	PutToDataSource(GetConfigFromSys())
	fmt.Println(DataSource["network"])
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
	bridges := GetConfigFromSys().Bridges
	bridges = append(bridges, Bridge{Name: name, Devs: dev, Mtu: mtu})
	userConfig.Bridges = bridges
	PutToDataSource(userConfig)
	return nil
}

func BridgeUpdate(name string, dev []string, mtu int) { // can not modify name
	BridgeDel(name)
	BridgeAdd(name, dev, mtu)
}

func BridgeDel(name string) {
	userConfig := GetConfigFromDs()
	bridges := userConfig.Bridges
	for i, bri := range bridges {
		if bri.Name == name {
			bridges = append(bridges[:i], bridges[i+1:]...)
		}
	}
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

	bonds := GetConfigFromSys().Bonds
	bonds = append(bonds, Bond{Name: name, Mode: netlink.BondMode(mode), Devs: dev})
	userConfig.Bonds = bonds
	PutToDataSource(userConfig)
	return nil
}

func BondUpdate(name string, mode int, dev []string) { // can not modify name
	BondDel(name)
	BondAdd(name, mode, dev)
}

func BondDel(name string) {
	userConfig := GetConfigFromDs()
	bonds := userConfig.Bridges
	for i, bri := range bonds {
		if bri.Name == name {
			bonds = append(bonds[:i], bonds[i+1:]...)
		}
	}
}

func VlanAdd(name string, tag int, parent string) error {
	userConfig := GetConfigFromDs()
	if isLinkAlreadyExists(name, userConfig) {
		//log.Error("interface named " + name + " alerady exists")
		return errors.New("interface named " + name + " alerady exists")
	}
	vlans := GetConfigFromSys().Vlans
	vlans = append(vlans, Vlan{Name: name, Tag: tag, Parent: parent})
	userConfig.Vlans = vlans
	PutToDataSource(userConfig)
	return nil
}

func VlanUpdate(name string, tag int, parent string) { // can not modify name
	VlanDel(name)
	VlanAdd(name, tag, parent)
}

func VlanDel(name string) {
	userConfig := GetConfigFromDs()
	vlans := userConfig.Vlans
	for i, v := range vlans {
		if v.Name == name {
			vlans = append(vlans[:i], vlans[i+1:]...)
		}
	}
}

func AssignIP(name string, ip string, mask string) {

}

func DelIP(name string) {

}

//not thread safe
func Apply(config Config) error {
	if err := breakNetwork(); err != nil {
		log.WithError(err).Error("break network failed")
		return err
	}
	for _, bond := range config.Bonds {
		if err := addBond(bond.Name, bond.Devs); err != nil {
			log.WithError(err).Error("add bond failed")
			return err
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
