package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

func init() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)
}

func main() {
	http.HandleFunc("/config", config)
	http.HandleFunc("/apply", apply)
	http.HandleFunc("/bondadd", bondAdd)
	http.HandleFunc("/bonddel", bondDel)
	http.HandleFunc("/briadd", briAdd)
	http.HandleFunc("/bridel", briDel)
	http.HandleFunc("/vlanadd", vlanAdd)
	http.HandleFunc("/vlandel", vlanDel)
	http.HandleFunc("/ipadd", ipAdd)
	http.HandleFunc("/ipdel", ipDel)
	err := http.ListenAndServe(":9090", nil) //设置监听的端口
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func config(resp http.ResponseWriter, req *http.Request, ) {
	req.ParseForm()
	resp.Header().Set("Content-Type", "application/json")
	r, _ := json.MarshalIndent(GetConfigFromDs(), "", "\t")
	resp.Write(r)
}

func apply(resp http.ResponseWriter, req *http.Request, ) {
	req.ParseForm()
	if err := Apply(GetConfigFromDs()); err != nil {
		http.Error(resp, err.Error(), http.StatusInternalServerError)
	}

	resp.Header().Set("Content-Type", "application/json")
	r, _ := json.MarshalIndent(GetConfigFromSys(), "", "\t")
	resp.Write(r)
}

func bondAdd(resp http.ResponseWriter, req *http.Request, ) {
	req.ParseForm()
	name := req.FormValue("name")
	mode, _ := strconv.Atoi(req.FormValue("mode"))
	devs := strings.Split(req.FormValue("dev"), " ")

	if err := BondAdd(name, mode, devs); err != nil {
		http.Error(resp, err.Error(), http.StatusInternalServerError)
	}

	resp.Header().Set("Content-Type", "application/json")
	jData, _ := json.MarshalIndent(GetConfigFromDs(), "", "\t")
	fmt.Println(GetConfigFromDs())
	resp.Write(jData)
}

func bondDel(resp http.ResponseWriter, req *http.Request, ) {
	req.ParseForm()
	name := req.FormValue("name")
	BondDel(name)
}

func briAdd(resp http.ResponseWriter, req *http.Request, ) {
	req.ParseForm()
	name := req.FormValue("name")
	devs := strings.Split(req.FormValue("dev"), " ")
	mtu, _ := strconv.Atoi(req.FormValue("mtu"))

	if err := BridgeAdd(name, devs, mtu); err != nil {
		http.Error(resp, err.Error(), http.StatusInternalServerError)
	}
}

func briDel(resp http.ResponseWriter, req *http.Request, ) {
	req.ParseForm()
	name := req.FormValue("name")
	BondDel(name)
}

func vlanAdd(resp http.ResponseWriter, req *http.Request, ) {
	req.ParseForm()
	name := req.FormValue("name")
	parent := req.FormValue("parent")
	tag, _ := strconv.Atoi(req.FormValue("tage"))

	if err := VlanAdd(name, tag, parent); err != nil {
		http.Error(resp, err.Error(), http.StatusInternalServerError)
	}
}

func vlanDel(resp http.ResponseWriter, req *http.Request, ) {
	req.ParseForm()
	name := req.FormValue("name")
	BondDel(name)
	resp.Header().Set("Content-Type", "application/json")
}

func ipAdd(resp http.ResponseWriter, req *http.Request, ) {
	req.ParseForm()
	name := req.FormValue("name")
	ips := strings.Split(req.FormValue("ips"), " ")

	if err := AssignIP(name, ips); err != nil {
		http.Error(resp, err.Error(), http.StatusInternalServerError)
	}
}

func ipDel(resp http.ResponseWriter, req *http.Request, ) {
	req.ParseForm()
	name := req.FormValue("name")
	ip := req.FormValue("ip")
	DelIP(name, ip)
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

func AssignIP(name string, ipNet []string) error {
	for _, ips := range ipNet {
		_, err := netlink.ParseAddr(ips)
		if err != nil {
			//log.WithError(err).Error("ip net format error, parse failed")
			return errors.New("ip net :" + ips + "format error, parse failed")
		}
	}

	userConfig := GetConfigFromDs()
	for i, d := range userConfig.Devices {
		if d.Name == name {
			userConfig.Devices[i].IpNets = append(userConfig.Devices[i].IpNets, ipNet...)
		}
	}
	for i, b := range userConfig.Bonds {
		if b.Name == name {
			userConfig.Bonds[i].IpNets = append(userConfig.Bonds[i].IpNets, ipNet...)
		}
	}
	PutToDataSource(userConfig)
	return nil
}

func DelIP(name string, ipNet string) {
	userConfig := GetConfigFromDs()

	for i, d := range userConfig.Devices {
		if d.Name == name {
			for j, ipnet := range userConfig.Devices[i].IpNets {
				if ipnet == ipNet {
					userConfig.Devices[i].IpNets = append(userConfig.Devices[i].IpNets[:j], userConfig.Devices[i].IpNets[j+1:]...)
				}
			}
		}
	}

	for i, b := range userConfig.Bonds {
		if b.Name == name {
			for j, ipnet := range userConfig.Bonds[i].IpNets {
				if ipnet == ipNet {
					userConfig.Bonds[i].IpNets = append(userConfig.Bonds[i].IpNets[:j], userConfig.Bonds[i].IpNets[j+1:]...)
				}
			}
		}
	}

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
		if device.Name == getAdminInterface() || device.Name == "lo"{
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
