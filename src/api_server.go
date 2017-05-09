package main

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	log "github.com/Sirupsen/logrus"
	"github.com/julienschmidt/httprouter"
	"github.com/vishvananda/netlink"
)

var (
	ErrNameUsed = errors.New("Interface Name alerady exists")
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
	router.GET("/network/init", initNetwork)
	router.GET("/network/config", config)
	router.GET("/network/apply", apply)

	router.POST("/network/bond/", bondAdd) // slave只可以从有的里面去
	router.DELETE("/network/bond/:Name", bondDel) // todo 没有的name del 显示 失败
	router.PUT("/network/bond", bondUpdate) // todo 同上

	router.POST("/network/bridge", briAdd)
	router.DELETE("/network/bridge/:Name", briDel)
	router.PUT("/network/bridge", briUpdate)

	router.POST("/network/vlan", vlanAdd)
	router.DELETE("/network/vlan/:Name", vlanDel)
	router.PUT("/network/vlan", vlanUpdate)

	router.POST("/network/Ip", ipAdd)
	router.DELETE("/network/Ip", ipDel)

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
		rm = ResponseMessage{Status: false, Message: "获取数据库网络配置配置失败." + err.Error(), Code: http.StatusInternalServerError}
	} else {
		rm = ResponseMessage{Result: userConfig, Status: true, Message: "获取数据库网络配置成功", Code: http.StatusOK}
	}
	ret, _ := json.MarshalIndent(rm, "", "\t")
	resp.Write(ret)
}

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
		sysConfig, _ := GetConfigFromSys()
		rm = ResponseMessage{Result: sysConfig, Status: true, Message: "应用网络配置成功", Code: http.StatusOK}
	}

	ret, _ := json.MarshalIndent(rm, "", "\t")
	resp.Write(ret)
}

func bondAdd(resp http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	var rm ResponseMessage
	resp.Header().Set("Content-Type", "application/json")
	bond, err := getBondJSONParam(req)
	if err != nil {
		rm = ResponseMessage{Status: false, Message: "Bond添加失败." + err.Error(), Code: http.StatusInternalServerError}
	} else if err := BondAdd(bond.Name, bond.Mode, bond.Devs); err != nil {
		rm = ResponseMessage{Status: false, Message: "Bond添加失败." + err.Error(), Code: http.StatusInternalServerError}
	} else {
		log.WithField("Bond", Bond{Name: bond.Name, Mode: bond.Mode, Devs: bond.Devs}).Info("添加Bond")
		rm = ResponseMessage{Status: true, Message: "Bond添加成功", Code: http.StatusCreated}
	}
	ret, _ := json.MarshalIndent(rm, "", "\t")
	resp.Write(ret)
}

func bondUpdate(resp http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	var rm ResponseMessage
	resp.Header().Set("Content-Type", "application/json")
	bond, err := getBondJSONParam(req)
	if err != nil {
		rm = ResponseMessage{Status: false, Message: "Bond更新失败." + err.Error(), Code: http.StatusInternalServerError}
	} else if err := BondUpdate(bond.Name, int(bond.Mode), bond.Devs); err != nil {
		rm = ResponseMessage{Status: false, Message: "Bond更新失败." + err.Error(), Code: http.StatusInternalServerError}
	} else {
		log.WithField("Bond", Bond{Name: bond.Name, Mode: bond.Mode, Devs: bond.Devs}).Info("更新Bond")
		rm = ResponseMessage{Status: true, Message: "Bond更新成功", Code: http.StatusOK}
	}
	ret, _ := json.MarshalIndent(rm, "", "\t")
	resp.Write(ret)
}

func bondDel(resp http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	var rm ResponseMessage
	resp.Header().Set("Content-Type", "application/json")
	name := ps.ByName("Name")
	if name == "" {
		rm = ResponseMessage{Status: false, Message: "Bond删除失败.Bond's Name can not be empty", Code: http.StatusInternalServerError}
	} else if err := BondDel(name); err != nil {
		rm = ResponseMessage{Status: false, Message: "Bond删除失败." + err.Error(), Code: http.StatusInternalServerError}
	} else {
		log.Info("删除Bond:" + name)
		rm = ResponseMessage{Status: true, Message: "Bond删除成功", Code: http.StatusOK}
	}
	ret, _ := json.MarshalIndent(rm, "", "\t")
	resp.Write(ret)
}

func briAdd(resp http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	var rm ResponseMessage
	resp.Header().Set("Content-Type", "application/json")
	bri, err := getBridgeJSONParam(req)
	if err != nil {
		rm = ResponseMessage{Status: false, Message: "Bridge添加失败." + err.Error(), Code: http.StatusInternalServerError}
	} else if err := BridgeAdd(bri.Name, bri.Devs, bri.Mtu); err != nil {
		rm = ResponseMessage{Status: false, Message: "Bridge添加失败." + err.Error(), Code: http.StatusInternalServerError}
	} else {
		log.WithField("Bridge", Bridge{Name: bri.Name, Devs: bri.Devs, Mtu: bri.Mtu}).Info("添加Bridge")
		rm = ResponseMessage{Status: true, Message: "Bridge添加成功", Code: http.StatusCreated}
	}
	ret, _ := json.MarshalIndent(rm, "", "\t")
	resp.Write(ret)
}

func briUpdate(resp http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	var rm ResponseMessage
	resp.Header().Set("Content-Type", "application/json")
	bri, err := getBridgeJSONParam(req)
	if err != nil {
		rm = ResponseMessage{Status: false, Message: "Bridge更新失败." + err.Error(), Code: http.StatusInternalServerError}
	} else if err := BridgeUpdate(bri.Name, bri.Devs, bri.Mtu); err != nil {
		rm = ResponseMessage{Status: false, Message: "Bridge更新失败." + err.Error(), Code: http.StatusInternalServerError}
	} else {
		log.WithField("Bridge", Bridge{Name: bri.Name, Devs: bri.Devs, Mtu: bri.Mtu}).Info("更新Bridge")
		rm = ResponseMessage{Status: true, Message: "Bridge更新成功", Code: http.StatusOK}
	}
	ret, _ := json.MarshalIndent(rm, "", "\t")
	resp.Write(ret)
}

func briDel(resp http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	var rm ResponseMessage
	resp.Header().Set("Content-Type", "application/json")
	name := ps.ByName("Name")
	if name == "" {
		rm = ResponseMessage{Status: false, Message: "Bridge删除失败.Bridge's Name can not be empty", Code: http.StatusInternalServerError}

	} else if err := BridgeDel(name); err != nil {
		rm = ResponseMessage{Status: false, Message: "Bridge删除失败." + err.Error(), Code: http.StatusInternalServerError}
	} else {
		log.Info("删除Bridge:" + name)
		rm = ResponseMessage{Status: true, Message: "Bridge删除成功", Code: http.StatusOK}
	}
	ret, _ := json.MarshalIndent(rm, "", "\t")
	resp.Write(ret)
}

func vlanAdd(resp http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	var rm ResponseMessage
	resp.Header().Set("Content-Type", "application/json")
	v, err := getVlanJSONParam(req)
	if err != nil {
		rm = ResponseMessage{Status: false, Message: "Vlan添加失败." + err.Error(), Code: http.StatusInternalServerError}
	} else if err := VlanAdd(v.Name, v.Tag, v.Parent); err != nil {
		rm = ResponseMessage{Status: false, Message: "Vlan添加失败." + err.Error(), Code: http.StatusInternalServerError}
	} else {
		log.WithField("Vlan", Vlan{Name: v.Name, Parent: v.Parent, Tag: v.Tag}).Info("添加Vlan")
		rm = ResponseMessage{Status: true, Message: "Vlan添加成功", Code: http.StatusCreated}
	}
	ret, _ := json.MarshalIndent(rm, "", "\t")
	resp.Write(ret)
}

func vlanUpdate(resp http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	var rm ResponseMessage
	resp.Header().Set("Content-Type", "application/json")
	v, err := getVlanJSONParam(req)
	if err != nil {
		rm = ResponseMessage{Status: false, Message: "Vlan更新失败." + err.Error(), Code: http.StatusInternalServerError}
	} else if err := VlanUpdate(v.Name, v.Tag, v.Parent); err != nil {
		rm = ResponseMessage{Status: false, Message: "Vlan更新失败." + err.Error(), Code: http.StatusInternalServerError}
	} else {
		log.WithField("Vlan", Vlan{Name: v.Name, Parent: v.Parent, Tag: v.Tag}).Info("更新Vlan")
		rm = ResponseMessage{Status: true, Message: "Vlan更新成功", Code: http.StatusOK}
	}
	ret, _ := json.MarshalIndent(rm, "", "\t")
	resp.Write(ret)
}

func vlanDel(resp http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	var rm ResponseMessage
	resp.Header().Set("Content-Type", "application/json")
	name := ps.ByName("Name")
	if name == "" {
		rm = ResponseMessage{Status: false, Message: "Vlan删除失败. Vlan's Name can not be empty", Code: http.StatusInternalServerError}
	} else if err := BondDel(name); err != nil || name == "" {
		rm = ResponseMessage{Status: false, Message: "Vlan删除失败." + err.Error(), Code: http.StatusInternalServerError}
	} else {
		log.Info("删除Vlan:" + name)
		rm = ResponseMessage{Status: true, Message: "Vlan删除成功", Code: http.StatusOK}
	}
	ret, _ := json.MarshalIndent(rm, "", "\t")
	resp.Write(ret)
}

func ipAdd(resp http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	var rm ResponseMessage
	resp.Header().Set("Content-Type", "application/json")
	i, err := getIPJSONParam(req)
	if err != nil {
		rm = ResponseMessage{Status: false, Message: "IP添加失败." + err.Error(), Code: http.StatusInternalServerError}
	} else if err := AssignIP(i.Name, i.Ip); err != nil {
		rm = ResponseMessage{Status: false, Message: "IP添加失败." + err.Error(), Code: http.StatusInternalServerError}
	} else {
		log.WithField("IP", i.Ip).Info(i.Name + "添加IP")
		rm = ResponseMessage{Status: true, Message: "IP添加成功", Code: http.StatusCreated}
	}
	ret, _ := json.MarshalIndent(rm, "", "\t")
	resp.Write(ret)
}

func ipDel(resp http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	var rm ResponseMessage
	resp.Header().Set("Content-Type", "application/json")
	i, err := getIPJSONParam(req)
	if err != nil {
		rm = ResponseMessage{Status: false, Message: "IP删除失败." + err.Error(), Code: http.StatusInternalServerError}
	} else if err := DelIP(i.Name, i.Ip[0]); err != nil {
		rm = ResponseMessage{Status: false, Message: "IPk删除失败." + err.Error(), Code: http.StatusInternalServerError}
	} else {
		log.Info(i.Name + "删除IP " + i.Ip[0])
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

func BridgeUpdate(name string, dev []string, mtu int) error { // can not modify Name
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

/*
BOND_MODE_BALANCE_RR     = iota(0)
BOND_MODE_ACTIVE_BACKUP
BOND_MODE_BALANCE_XOR
BOND_MODE_BROADCAST
BOND_MODE_802_3AD
BOND_MODE_BALANCE_TLB
BOND_MODE_BALANCE_ALB
BOND_MODE_UNKNOWN
*/
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

	userConfig.Bonds = append(userConfig.Bonds, Bond{Name: name, Mode: mode, Devs: dev})

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

func BondUpdate(name string, mode int, dev []string) error { // can not modify Name
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

func VlanUpdate(name string, tag int, parent string) error { // can not modify Name
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

func getBondJSONParam(req *http.Request) (Bond, error) {
	req.ParseForm()
	var bond Bond
	body, _ := ioutil.ReadAll(io.LimitReader(req.Body, 1048576))
	if err := json.Unmarshal(body, &bond); err != nil {
		return Bond{}, errors.New("用户输入参数格式有误")
	}
	if bond.Name == "" {
		return Bond{}, errors.New("Bond's Name can not be empty")
	}
	return bond, nil
}

func getBridgeJSONParam(req *http.Request) (Bridge, error) {
	req.ParseForm()
	var bri Bridge
	body, _ := ioutil.ReadAll(io.LimitReader(req.Body, 1048576))
	if err := json.Unmarshal(body, &bri); err != nil {
		return Bridge{}, errors.New("用户输入参数格式有误")
	}

	if bri.Name == "" {
		return Bridge{}, errors.New("Bridge's Name can not be empty")
	}

	if bri.Mtu == 0 {
		bri.Mtu = 1500
	}
	return bri, nil
}

func getVlanJSONParam(req *http.Request) (Vlan, error) {
	req.ParseForm()
	var v Vlan
	body, _ := ioutil.ReadAll(io.LimitReader(req.Body, 1048576))
	if err := json.Unmarshal(body, &v); err != nil {
		return Vlan{}, errors.New("用户输入参数格式有误")
	}

	if v.Name == "" {
		return Vlan{}, errors.New("Vlan's Name can not be empty")
	}
	if v.Parent == "" {
		return Vlan{}, errors.New("Vlan's parent can not be empty")
	}
	return v, nil
}

// used to unmarshal req
type ipParam struct {
	Name string
	Ip   []string
}

func getIPJSONParam(req *http.Request) (ipParam, error) {
	req.ParseForm()
	var i ipParam
	body, _ := ioutil.ReadAll(io.LimitReader(req.Body, 1048576))
	if err := json.Unmarshal(body, &i); err != nil {
		return ipParam{}, errors.New("用户输入参数格式有误")
	}

	if i.Name == "" {
		return ipParam{}, errors.New("Device Name can not be empty when setting IP")
	}
	return i, nil
}
