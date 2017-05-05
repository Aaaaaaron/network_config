package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/julienschmidt/httprouter"
	"github.com/vishvananda/netlink"
)

var (
	ErrNameUsed = errors.New("Interface name alerady exists")
	ErrDevsUsed = errors.New("Devs has alerady been occupied")
)

func init() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)
}

type ResponseMessage struct {
	Result  interface{}    `json:"result,omitempty"`
	Status  bool           `json:"status"`
	Message string         `json:"message"`
	Code    int            `json:"code"`
}

func main() {
	router := httprouter.New()
	router.GET("/init", initNetwork)

	http.HandleFunc("/config", config)
	http.HandleFunc("/dsconfig", dsconfig)
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

func initNetwork(resp http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	var rm ResponseMessage
	if err := breakNetwork(); err != nil {
		rm = ResponseMessage{Status: false, Message: "初始化网络配置失败" + err.Error(), Code: http.StatusInternalServerError}
	}
	rm = ResponseMessage{Status: true, Message: "初始化网络配置成功", Code: http.StatusOK}

	ret, _ := json.MarshalIndent(rm, "", "\t")
	resp.Write(ret)
}

func dsconfig(resp http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	resp.Header().Set("Content-Type", "application/json")
	userConfig, err := GetConfigFromDs()
	if err != nil {
		log.WithError(err).Error("Get config from database failed")
	}

	r, _ := json.MarshalIndent(userConfig, "", "\t")
	resp.Write(r)
}

func config(resp http.ResponseWriter, req *http.Request, ) {
	req.ParseForm()
	resp.Header().Set("Content-Type", "application/json")
	userConfig, err := GetConfigFromDs()
	if err != nil {
		log.WithError(err).Error("Get config from database failed")
	}

	r, _ := json.MarshalIndent(userConfig, "", "\t")
	resp.Write(r)
}

func apply(resp http.ResponseWriter, req *http.Request, ) {
	req.ParseForm()
	userConfig, err := GetConfigFromDs()
	if err != nil {
		log.WithError(err).Error("Get config from database failed")
	}

	if err := Apply(userConfig); err != nil {
		http.Error(resp, err.Error(), http.StatusInternalServerError)
	}

	resp.Header().Set("Content-Type", "application/json")
	sysConfig, _ := GetConfigFromSys()
	r, _ := json.MarshalIndent(sysConfig, "", "\t")
	resp.Write(r)
}

func bondAdd(resp http.ResponseWriter, req *http.Request, ) {
	req.ParseForm()
	name := req.FormValue("name")
	mode, _ := strconv.Atoi(req.FormValue("mode"))
	devs := strings.Split(req.FormValue("dev"), ",")

	if err := BondAdd(name, mode, devs); err != nil {
		http.Error(resp, err.Error(), http.StatusInternalServerError)
	}
}

func bondDel(resp http.ResponseWriter, req *http.Request, ) {
	req.ParseForm()
	name := req.FormValue("name")
	BondDel(name)
}

func briAdd(resp http.ResponseWriter, req *http.Request, ) {
	req.ParseForm()
	name := req.FormValue("name")
	devs := strings.Split(req.FormValue("dev"), ",")
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
	ips := strings.Split(req.FormValue("ips"), ",")

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

////////////////////////////////////////////////////////////////////
// get config form system
func GetConfigFromSys() (Config, error) {
	var config Config
	links, err := netlink.LinkList()
	if err != nil {
		log.WithError(err).Error("Get link list fail")
		return Config{}, err
	}

	devMap := getSlaveList(links)
	for _, link := range links {
		if err = grantConfig(link, devMap, &config); err != nil {
			log.WithError(err).Error("Grant config fail")
			return Config{}, err
		}
	}
	return config, nil
}

//get config from database
func GetConfigFromDs() (Config, error) {
	var config Config
	err := json.Unmarshal(([]byte)(DataSource["network"]), &config)
	if err != nil {
		log.WithError(err).Error("Json unmarshall fail")
		return Config{}, err
	}
	return config, nil
}

// below manipulate database's data
func BridgeAdd(name string, dev []string, mtu int) error {
	// 要根据数据源里存的配置的进行校验 而不是从系统中取到的配置
	userConfig, err := GetConfigFromDs()
	if err != nil {
		log.WithError(err).Error("Get config from database failed")
		return err
	}

	if err := validate(name, dev, userConfig); err != nil {
		log.WithError(err).Error("Validate fail")
		return err
	}

	userConfig.Bridges = append(userConfig.Bridges, Bridge{Name: name, Devs: dev, Mtu: mtu})

	if err := PutToDataSource(userConfig); err != nil {
		log.WithError(err).Error("Put data to database fail")
		return err
	}
	return nil
}

func BridgeUpdate(name string, dev []string, mtu int) error { // can not modify name
	BridgeDel(name)
	return BridgeAdd(name, dev, mtu)
}

func BridgeDel(name string) error {
	userConfig, err := GetConfigFromDs()
	if err != nil {
		log.WithError(err).Error("Get config from database failed")
		return err
	}

	for i, bri := range userConfig.Bridges {
		if bri.Name == name {
			userConfig.Bridges = append(userConfig.Bridges[:i], userConfig.Bridges[i+1:]...)
		}
	}

	if err := PutToDataSource(userConfig); err != nil {
		log.WithError(err).Error("Put data to database fail")
		return err
	}
	return nil
}

func BondAdd(name string, mode int, dev []string) error {
	userConfig, err := GetConfigFromDs()
	if err != nil {
		log.WithError(err).Error("Get config from database failed")
		return err
	}

	if err := validate(name, dev, userConfig); err != nil {
		log.WithError(err).Error("Validate fail")
		return err
	}

	userConfig.Bonds = append(userConfig.Bonds, Bond{Name: name, Mode: netlink.BondMode(mode), Devs: dev})

	if err := PutToDataSource(userConfig); err != nil {
		log.WithError(err).Error("Put data to database fail")
		return err
	}
	return nil
}

func BondDel(name string) error {
	userConfig, err := GetConfigFromDs()
	if err != nil {
		log.WithError(err).Error("Get config from database failed")
		return err
	}

	for i, bri := range userConfig.Bonds {
		if bri.Name == name {
			userConfig.Bonds = append(userConfig.Bonds[:i], userConfig.Bonds[i+1:]...)
		}
	}

	if err := PutToDataSource(userConfig); err != nil {
		log.WithError(err).Error("Put data to database fail")
		return err
	}
	return nil
}

func BondUpdate(name string, mode int, dev []string) error { // can not modify name
	if err := BondDel(name); err != nil {
		log.WithError(err).Error("Bond " + name + " del fail")
		return err
	}
	if err := BondAdd(name, mode, dev); err != nil {
		log.WithError(err).Error("Bond " + name + " add fail")
		return err
	}
	return nil
}

func VlanAdd(name string, tag int, parent string) error {
	userConfig, err := GetConfigFromDs()
	if err != nil {
		log.WithError(err).Error("Get config from database failed")
		return err
	}

	if isLinkAlreadyExists(name, userConfig) {
		log.WithError(ErrNameUsed).Error("Name:" + name)
		return ErrNameUsed
	}

	userConfig.Vlans = append(userConfig.Vlans, Vlan{Name: name, Tag: tag, Parent: parent})

	if err := PutToDataSource(userConfig); err != nil {
		log.WithError(err).Error("Put data to database fail")
		return err
	}
	return nil
}

func VlanUpdate(name string, tag int, parent string) error { // can not modify name
	if err := VlanDel(name); err != nil {
		log.WithError(err).Error("Vlan " + name + " del fail")
		return err
	}
	if err := VlanAdd(name, tag, parent); err != nil {
		log.WithError(err).Error("Vlan " + name + " add fail")
		return err
	}
	return nil
}

func VlanDel(name string) error {
	userConfig, err := GetConfigFromDs()
	if err != nil {
		log.WithError(err).Error("Get config from database failed")
		return err
	}

	for i, v := range userConfig.Vlans {
		if v.Name == name {
			userConfig.Vlans = append(userConfig.Vlans[:i], userConfig.Vlans[i+1:]...)
		}
	}

	if err := PutToDataSource(userConfig); err != nil {
		log.WithError(err).Error("Put data to database fail")
		return err
	}
	return nil
}

func AssignIP(name string, ipNet []string) error {
	for _, ips := range ipNet {
		_, err := netlink.ParseAddr(ips)
		if err != nil {
			log.WithError(err).Error("Parse IP failed,please check the input IP's format")
			return err
		}
	}

	userConfig, err := GetConfigFromDs()
	if err != nil {
		log.WithError(err).Error("Get config from database failed")
		return err
	}

	// assign IP to devices
	for i, d := range userConfig.Devices {
		if d.Name == name {
			userConfig.Devices[i].IpNets = append(userConfig.Devices[i].IpNets, ipNet...)
		}
	}

	// assign IP to bonds
	for i, b := range userConfig.Bonds {
		if b.Name == name {
			userConfig.Bonds[i].IpNets = append(userConfig.Bonds[i].IpNets, ipNet...)
		}
	}

	if err := PutToDataSource(userConfig); err != nil {
		log.WithError(err).Error("Put data to database fail")
		return err
	}
	return nil
}

func DelIP(name string, ipNet string) error {
	userConfig, err := GetConfigFromDs()
	if err != nil {
		log.WithError(err).Error("Get config from database failed")
		return err
	}

	//del devices's IP
	for i, d := range userConfig.Devices {
		if d.Name == name {
			for j, ipnet := range userConfig.Devices[i].IpNets {
				if ipnet == ipNet {
					userConfig.Devices[i].IpNets = append(userConfig.Devices[i].IpNets[:j], userConfig.Devices[i].IpNets[j+1:]...)
				}
			}
		}
	}

	//del bonds's IP
	for i, b := range userConfig.Bonds {
		if b.Name == name {
			for j, ipnet := range userConfig.Bonds[i].IpNets {
				if ipnet == ipNet {
					userConfig.Bonds[i].IpNets = append(userConfig.Bonds[i].IpNets[:j], userConfig.Bonds[i].IpNets[j+1:]...)
				}
			}
		}
	}

	if err := PutToDataSource(userConfig); err != nil {
		log.WithError(err).Error("Put data to database fail")
		return err
	}
	return nil
}

func validate(name string, dev []string, userConfig Config) error {
	if isLinkAlreadyExists(name, userConfig) {
		log.WithError(ErrNameUsed).Error("Name:" + name)
		return ErrNameUsed
	}

	if isDevsAlreadyUsed(dev, userConfig) {
		log.WithError(ErrDevsUsed)
		return ErrDevsUsed
	}
	return nil
}

func isDevsAlreadyUsed(devs []string, config Config) bool {
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
