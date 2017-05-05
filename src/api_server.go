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
	router.GET("/config", config)
	router.GET("/apply", apply)

	router.POST("/bond/", bondAdd)
	router.DELETE("/bond/:name", bondDel)
	router.PUT("/bond", bondUpdate)

	router.POST("/bridge", briAdd)
	router.DELETE("/bridge/:name", briDel)
	router.PUT("/bridge", briUpdate)

	router.POST("/vlan", vlanAdd)
	router.DELETE("/vlan/:name", vlanDel)
	router.PUT("/vlan", vlanUpdate)

	router.POST("/ip", ipAdd)
	router.DELETE("/ip", ipDel)

	log.Info("服务启动")
	err := http.ListenAndServe(":9090", router) //设置监听的端口
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func initNetwork(resp http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	log.Info("初始化网络")
	var rm ResponseMessage
	resp.Header().Set("Content-Type", "application/json")
	if err := breakNetwork(); err != nil {
		rm = ResponseMessage{Status: false, Message: "初始化网络配置失败." + err.Error(), Code: http.StatusInternalServerError}
	} else {
		rm = ResponseMessage{Status: true, Message: "初始化网络配置成功", Code: http.StatusOK}
	}
	ret, _ := json.MarshalIndent(rm, "", "\t")
	resp.Write(ret)
}

func config(resp http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	log.Info("从数据库获取网络配置")
	var rm ResponseMessage
	resp.Header().Set("Content-Type", "application/json")
	userConfig, err := GetConfigFromDs()
	if err != nil {
		rm = ResponseMessage{Status: false, Message: "获取数据库配置失败." + err.Error(), Code: http.StatusInternalServerError}
	} else {
		rm = ResponseMessage{Result: userConfig, Status: true, Message: "初始化网络配置成功", Code: http.StatusOK}
	}
	ret, _ := json.MarshalIndent(rm, "", "\t")
	resp.Write(ret)
}

//func dsconfig(resp http.ResponseWriter, req *http.Request) {
//	req.ParseForm()
//	resp.Header().Set("Content-Type", "application/json")
//	userConfig, err := GetConfigFromDs()
//	if err != nil {
//		log.WithError(err).Error("Get config from database failed")
//	}
//
//	r, _ := json.MarshalIndent(userConfig, "", "\t")
//	resp.Write(r)
//}

func apply(resp http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	log.Info("应用网络配置")
	var rm ResponseMessage
	resp.Header().Set("Content-Type", "application/json")
	userConfig, err := GetConfigFromDs()
	if err != nil {
		rm = ResponseMessage{Status: false, Message: "获取数据库配置失败." + err.Error(), Code: http.StatusInternalServerError}
	} else if err := Apply(userConfig); err != nil {
		rm = ResponseMessage{Status: false, Message: "应用网络配置失败." + err.Error(), Code: http.StatusInternalServerError}
	} else {
		//sysConfig, _ := GetConfigFromSys()
		//r, _ := json.MarshalIndent(sysConfig, "", "\t")
		rm = ResponseMessage{Result: userConfig, Status: true, Message: "应用网络配置成功", Code: http.StatusOK}
	}

	ret, _ := json.MarshalIndent(rm, "", "\t")
	resp.Write(ret)
}

func bondAdd(resp http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	req.ParseForm()
	var rm ResponseMessage
	resp.Header().Set("Content-Type", "application/json")
	name := req.FormValue("name")
	mode, _ := strconv.Atoi(req.FormValue("mode"))
	devs := strings.Split(req.FormValue("dev"), ",")
	log.WithField("Bond", Bond{Name: name, Mode: netlink.BondMode(mode), Devs: devs}).Info("添加Bond")

	if err := BondAdd(name, mode, devs); err != nil {
		rm = ResponseMessage{Status: false, Message: "Bond添加失败." + err.Error(), Code: http.StatusInternalServerError}
	} else {
		rm = ResponseMessage{Status: true, Message: "Bond添加成功", Code: http.StatusOK}
	}
	ret, _ := json.MarshalIndent(rm, "", "\t")
	resp.Write(ret)
}

func bondUpdate(resp http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	req.ParseForm()
	var rm ResponseMessage
	resp.Header().Set("Content-Type", "application/json")
	name := req.FormValue("name")
	mode, _ := strconv.Atoi(req.FormValue("mode"))
	devs := strings.Split(req.FormValue("dev"), ",")
	log.WithField("Bond", Bond{Name: name, Mode: netlink.BondMode(mode), Devs: devs}).Info("更新Bond")

	if err := BondUpdate(name, mode, devs); err != nil {
		rm = ResponseMessage{Status: false, Message: "Bond更新失败." + err.Error(), Code: http.StatusInternalServerError}
	} else {
		rm = ResponseMessage{Status: true, Message: "Bond更新成功", Code: http.StatusOK}
	}
	ret, _ := json.MarshalIndent(rm, "", "\t")
	resp.Write(ret)
}

func bondDel(resp http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	var rm ResponseMessage
	resp.Header().Set("Content-Type", "application/json")
	name := ps.ByName("name")
	log.Info("删除Bond:" + name)

	if err := BondDel(name); err != nil {
		rm = ResponseMessage{Status: false, Message: "Bond删除失败." + err.Error(), Code: http.StatusInternalServerError}
	} else {
		rm = ResponseMessage{Status: true, Message: "Bond删除成功", Code: http.StatusOK}
	}
	ret, _ := json.MarshalIndent(rm, "", "\t")
	resp.Write(ret)
}

func briAdd(resp http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	req.ParseForm()
	var rm ResponseMessage
	resp.Header().Set("Content-Type", "application/json")
	name := req.FormValue("name")
	devs := strings.Split(req.FormValue("dev"), ",")
	mtu, _ := strconv.Atoi(req.FormValue("mtu"))
	log.WithField("Bridge", Bridge{Name: name, Devs: devs, Mtu: mtu}).Info("添加Bridge")

	if err := BridgeAdd(name, devs, mtu); err != nil {
		rm = ResponseMessage{Status: false, Message: "Bridge添加失败." + err.Error(), Code: http.StatusInternalServerError}
	} else {
		rm = ResponseMessage{Status: true, Message: "Bridge添加成功", Code: http.StatusOK}
	}
	ret, _ := json.MarshalIndent(rm, "", "\t")
	resp.Write(ret)
}

func briUpdate(resp http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	req.ParseForm()
	var rm ResponseMessage
	resp.Header().Set("Content-Type", "application/json")
	name := req.FormValue("name")
	devs := strings.Split(req.FormValue("dev"), ",")
	mtu, _ := strconv.Atoi(req.FormValue("mtu"))
	log.WithField("Bridge", Bridge{Name: name, Devs: devs, Mtu: mtu}).Info("更新Bridge")

	if err := BridgeUpdate(name, devs, mtu); err != nil {
		rm = ResponseMessage{Status: false, Message: "Bridge更新失败." + err.Error(), Code: http.StatusInternalServerError}
	} else {
		rm = ResponseMessage{Status: true, Message: "Bridge更新成功", Code: http.StatusOK}
	}
	ret, _ := json.MarshalIndent(rm, "", "\t")
	resp.Write(ret)
}

func briDel(resp http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	var rm ResponseMessage
	resp.Header().Set("Content-Type", "application/json")
	name := ps.ByName("name")
	log.Info("删除Bridge:" + name)

	if err := BridgeDel(name); err != nil {
		rm = ResponseMessage{Status: false, Message: "Bridge删除失败." + err.Error(), Code: http.StatusInternalServerError}
	} else {
		rm = ResponseMessage{Status: true, Message: "Bridge删除成功", Code: http.StatusOK}
	}
	ret, _ := json.MarshalIndent(rm, "", "\t")
	resp.Write(ret)
}

func vlanAdd(resp http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	req.ParseForm()
	var rm ResponseMessage
	resp.Header().Set("Content-Type", "application/json")
	name := req.FormValue("name")
	parent := req.FormValue("parent")
	tag, _ := strconv.Atoi(req.FormValue("tag"))
	log.WithField("Vlan", Vlan{Name: name, Parent: parent, Tag: tag}).Info("添加Vlan")

	if err := VlanAdd(name, tag, parent); err != nil {
		rm = ResponseMessage{Status: false, Message: "Vlan添加失败." + err.Error(), Code: http.StatusInternalServerError}
	} else {
		rm = ResponseMessage{Status: true, Message: "Vlan添加成功", Code: http.StatusOK}
	}
	ret, _ := json.MarshalIndent(rm, "", "\t")
	resp.Write(ret)
}

func vlanUpdate(resp http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	req.ParseForm()
	var rm ResponseMessage
	resp.Header().Set("Content-Type", "application/json")
	name := req.FormValue("name")
	parent := req.FormValue("parent")
	tag, _ := strconv.Atoi(req.FormValue("tag"))
	log.WithField("Vlan", Vlan{Name: name, Parent: parent, Tag: tag}).Info("更新Vlan")

	if err := VlanAdd(name, tag, parent); err != nil {
		rm = ResponseMessage{Status: false, Message: "Vlan更新失败." + err.Error(), Code: http.StatusInternalServerError}
	} else {
		rm = ResponseMessage{Status: true, Message: "Vlan更新成功", Code: http.StatusOK}
	}
	ret, _ := json.MarshalIndent(rm, "", "\t")
	resp.Write(ret)
}

func vlanDel(resp http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	var rm ResponseMessage
	resp.Header().Set("Content-Type", "application/json")
	name := ps.ByName("name")
	log.Info("删除Vlan:" + name)

	if err := BondDel(name); err != nil {
		rm = ResponseMessage{Status: false, Message: "Vlan删除失败." + err.Error(), Code: http.StatusInternalServerError}
	} else {
		rm = ResponseMessage{Status: true, Message: "Vlan删除成功", Code: http.StatusOK}
	}
	ret, _ := json.MarshalIndent(rm, "", "\t")
	resp.Write(ret)
}

func ipAdd(resp http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	req.ParseForm()
	var rm ResponseMessage
	resp.Header().Set("Content-Type", "application/json")
	name := req.FormValue("name")
	ips := strings.Split(req.FormValue("ips"), ",")
	log.WithField("IP", ips).Info(name + "添加IP")

	if err := AssignIP(name, ips); err != nil {
		rm = ResponseMessage{Status: false, Message: "IP添加失败." + err.Error(), Code: http.StatusInternalServerError}
	} else {
		rm = ResponseMessage{Status: true, Message: "IP添加成功", Code: http.StatusOK}
	}
	ret, _ := json.MarshalIndent(rm, "", "\t")
	resp.Write(ret)
}

func ipDel(resp http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	req.ParseForm()
	var rm ResponseMessage
	resp.Header().Set("Content-Type", "application/json")
	name := req.FormValue("name")
	ip := req.FormValue("ip")
	log.Info(name + "删除IP " + ip)

	if err := DelIP(name, ip); err != nil {
		rm = ResponseMessage{Status: false, Message: "IP删除失败." + err.Error(), Code: http.StatusInternalServerError}
	} else {
		rm = ResponseMessage{Status: true, Message: "IP删除成功", Code: http.StatusOK}
	}
	ret, _ := json.MarshalIndent(rm, "", "\t")
	resp.Write(ret)
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
	if err := BridgeDel(name); err != nil {
		log.WithError(err).Error("Bond " + name + " del fail")
		return err
	}
	if err := BridgeAdd(name, dev, mtu); err != nil {
		log.WithError(err).Error("Bond " + name + " add fail")
		return err
	}
	return nil
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
