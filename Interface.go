package main

import (
	"fmt"
	"net"
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
	Bridges []Bridge
	Bonds   []Bond
	Vlans   []Vlan
	//后期想到上面新的配置项可以加在这里
}

type Bridge struct {
	Name string
	Dev  string
	Addr []net.IPNet
	Mtu  int
	Stp  string
}

type Bond struct {
	Name string
	Mode string
	Dev  string
	Addr []net.IPNet
}

type Vlan struct {
	Name   string
	Tag    string
	Parent string
	Addr   []net.IPNet
}

func main() {
	GetSysConfig()
}

//func GetSysConfig() Config {
func GetSysConfig() Config {
	links := getLinkList()
	for _, link := range links {
		//link.Type()
		fmt.Println(link)
	}
}

func getLinkList() []netlink.Link {
	linkList, err := netlink.LinkList()
	if err != nil {
		log.Fatalf("get link list from netlink failed: %s", err)
	}
	return linkList
}

func getHostId() string {
	return "1"
}
