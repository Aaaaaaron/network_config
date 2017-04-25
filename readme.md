# 已完成
1. 从系统返回当前设置 
2. 拆除bond/bridge/vlan
3. 创建bond/bridge/vlan
4. 上面三点的单元测试

主要代码在src/interface.go

测试代码在src/interface_test.go

项目目录:/root/work/network_config

运行测试:sh /root/work/network_config/bin/test.sh
***
# 一.用户正常修改
1. 用户修改配置
2. gateway校验配置
3. 校验通过修改到etcd
4. 用户点立即应用,gateway从etcd中取当前waf的配置,传给exec函数
5. exec函数根据这份配置重建系统配置.直接返回给用户成功与否

# 二.配置应用到系统流程
0. 关闭接口
1. 删除现有网桥
2. 删除现有VLAN虚拟接口
3. 删除现有链路聚合设备
4. 创建链路聚合设备(若需要)
    1. 初始化bonding模块(bond设备数量及模式)
    2. 依次组建各bonding设备
5. 创建VLAN虚拟接口(若需要)
6. 创建网桥(若需要)
7. 删除现有接口绑定的所有IP
8. 为每个接口绑定IP地址和子网掩码

# 三.说明
1. 用户就是简单的修改MTU,也会走一遍上面的流程
2. 若是切换etcd,系统就默认执行一次立即应用,从新的etcd实例中取出数据应用到系统
3. 一旦系统收到 "立即应用信号",不管配置有无改动,都把当前配置应用到系统(走一遍上面的流程)

# 四.API
ModifyConifg()

ApplyModify()

# 五.相关设置json
etcd中存储的配置k-v:hostId-config
```
{
    "hostId":"",//集群中的哪台waf
    "config": [
        {
            "bridge": [
                {"name":"br1", "dev":"eth0 eth1", "mtu":1500, "stp":"off",[addr{"IP":"1.1.1.1", "Mask":"255.255.255.0"}]},
                {"name":"br2", "dev":"eth2 eth3", "mtu":1500, "stp":"off",[addr{"IP":"1.1.1.2", "Mask":"255.255.255.0"}]}        
            ],
            "bond": [
                {"name":"bond1", "dev":"eth4", "mode":"1",[addr{"IP":"1.1.1.3", "Mask":"255.255.255.0"}]}
            ],
            "vlan" :[
                {"name":"eth0.100", "parent":"eth5", "tag":"110",[addr{"IP":"1.1.1.4", "Mask":"255.255.255.0"}]}
            ]
        }
    ]
}
```

## 结构体
```
type Config struct {
	HostId  string
	Devices []Device
	Bonds   []Bond
	Bridges []Bridge
	Vlans   []Vlan
	//后期想到上面新的配置项可以加在这里
}

type Device struct {
	Index int
	Name  string
	Addr  []net.IPNet
}

type Bond struct {
	Index int
	Name  string
	Mode  netlink.BondMode
	Dev   string
	Addr  []net.IPNet
}

type Bridge struct {
	Index int
	Name  string
	Dev   string
	Addr  []net.IPNet
	Mtu   int
	Stp   string
}

type Vlan struct {
	Index  int
	Name   string
	Tag    string
	Parent string
	Addr   []net.IPNet
}
```


# 六.伪代码
```
@POST
@Path("/config)
func ModifyConifg(req Request) (resp Response) {
  userModifiedConfig := req.getConfig()
  if !validate(modifyConfig) {
    resp.Status = 403
    log.Fatal("validate fail")
  }
  putEtcd(groupId,config)
  resp.Status = 200
}
```

```
@GET
@Path("/apply/{groupId}")
func ApplyModify(req Request) (resp Response) {
  groupId := req.Body.get("groupId")
  config := getEtcd(groupId)
  if err := exec(config) != nil {
    resp.Status = 403
    log.Fatal("execute failed")
  }
  resp.Status = 200
}
```

```
func exec(config string) error {
  if err := breakNetwork() != nil {
    return err
  }
  if err := bulidNetwork(config) != nil {
    return err
  }
}
```

## Questions
1. validate哪些东西,需要确认
2. 什么才算是不能再拆的状态?
3. 是否回滚(目前先不回滚) 回滚的一种方案,把etcd回退到上一版本(就是应用之前的那个版本),然后再对这个版本应用更改.但是由于一般执行失败可能是由于硬件原因,所以这样还是有很大可能性执行失败.
4. 直接返回执行是否成功给用户,系统不再记录状态
5. 拆的粒度是不是要分的更细?
