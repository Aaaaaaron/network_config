package main

//mock data source
func init() {
	DataSource = make(map[string]string)
	sysConfig, _ := GetConfigFromSys()
	PutToDataSource(sysConfig)
}

var DataSource map[string]string
