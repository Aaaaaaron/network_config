package main

import "fmt"

func main() {
	PutToDataSource()
	fmt.Println(DataSource["network"])
}

func GetConfig() {

}

func BridgeAdd(name string, dev []string, mtu int) {

}

func BridgeUpdate(name string, dev []string, mtu int) { // can not modify name

}

func BridgeDel(name string) {

}

func BondAdd(name string, mode int, dev []string) {

}

func BondUpdate(name string, mode int, dev []string) { // can not modify name
}

func BondDel(name string) {

}

func VlanAdd(name string, tag int, parent string) {

}

func VlanUpdate(name string, tag int, parent string) { // can not modify name

}

func VlanDel(name string) {

}

func AssignIP(name string, ip string, mask string) {

}

func DelIP(name string) {

}
