package dht

import (
	"bytes"
	"log"
	"net"

	bencode "github.com/jackpal/bencode-go"
)

// bencode有4种数据类型:string,integer,list和dictionary。
// 1. string：字符是以这种方式编码的: <字符串长度>:<字符串>。
// 如，"hello"：5:hello

// 2. integer：整数是以这种方式编码的: i<整数>e。
// 如，1234：i1234e

// 3. list：列表是以这种方式编码的: l[数据1][数据2][数据3][…]e。
// 如，["hello","world",1234]
// 1). "hello"编码：5:hello
// 2). "world"编码：5:world
// 3). 1234编码：i1234e
// 4). 最终编码：l5:hello5:worldi1234ee

// 4. dictionary：字典是以这种方式编码的: d[key1][value1][key2][value2][…]e，
// 其中key必须是string而且按照字母顺序排序。
// 如，{"name":"jisen","coin":"btc","balance":1000}
// 1). "name":"jisen"编码：4:name5:jisen
// 2). "coin":"btc"编码：4:coin3:btc
// 3). "balance":1000编码：7:balancei1000e
// 4). 最终编码，按key的字母排序：d7:balancei1000e4:coin3:btc4:name5:jisene

// https://zhuanlan.zhihu.com/p/34377702

// ping: A向B发送请求,测试对方节点是否存活. 如果B存活,需要响应对应报文

// find_node: A向B查询某个nodeId. B需要从自己的路由表中找到对应的nodeId返回,或者返回离该nodeId最近的8个node信息.
// 然后A节点可以再向B节点继续发送find_node请求

// get_peers: A向B查询某个infoHash(可以理解为一个Torrent的id,也是由20个字节组成.
// 该20个字节并非随机,是由Torrent文件中的metadata字段(该字段包含了文件的主要信息,
// 也就是上文提到的名字/长度/子文件目录/子文件长度等信息,实际上一个磁力搜索网站提供的也就是这些信息).进行SH1编码生成的).
// 如果B拥有该infoHash的信息,则返回该infoHash 的peers(也就是可以从这些peers处下载到该种子和文件).
// 如果没有,则返回离该infoHash最近的8个node信息. 然后A节点可继续向这些node发送请求.

// announce_peer: A通知B(以及其他若干节点)自己拥有某个infoHash的资源
// (也就是A成为该infoHash的peer,可提供文件或种子的下载),并给B发送下载的端口.
// 其实是A在收到最终地址结果后，扩散通知给其他节点上的节点

// 目前,大部分info_hash都是通过announce_peer获取到的
// 如何通过info_hash获取到torrent的metadata信息.网上的普遍说法是两种方式:
// 1、从迅雷种子库(以及其他一些磁力网站的接口)获取.大多数的实现方式,
// 都是拼接URL + infoHash + ".torrent".但是大多数能查到的接口都已经失效.
// 2、通过bep-009协议获取.但是我看了官网的该协议,仍是一头雾水.
// 对于第二种方式,直到我找到了这篇

// DHT 基准
// Tracker服务器会存在单点故障问题。所以在BT技术的基础上，后来又衍生出DHT网络和磁力链接技术
// http://www.bittorrent.org/beps/bep_0005.html

// DHT IPV6
// http://www.bittorrent.org/beps/bep_0032.html

// BitTorrent 协议
// BitTorrent's peer protocol operates over TCP or uTP.
// https://wiki.theory.org/BitTorrentSpecification
// v1
// http://www.bittorrent.org/beps/bep_0003.html
// v2
// http://www.bittorrent.org/beps/bep_0052.html

// bt 获取元数据（种子） ut_metadata, 使用bt的扩展协议
// http://www.bittorrent.org/beps/bep_0009.html
// http://www.bittorrent.org/beps/bep_0010.html
// https://www.aneasystone.com/archives/2015/05/analyze-magnet-protocol-using-wireshark.html
// https://blog.csdn.net/qq_41910048/article/details/105615275

// bt utp协议-实现拥塞控制
// http://www.bittorrent.org/beps/bep_0029.html

// 大型实现
// http://libtorrent.org/

// bt 全部协议
// http://www.bittorrent.org/beps/bep_0000.html

// 抓包文件
// https://wiki.wireshark.org/BitTorrent

// bt 传输算法
// http://bittorrent.org/bittorrentecon.pdf

// put、get
// http://www.libtorrent.org/dht_store.html

type structNested struct {
	//https://www.cnblogs.com/bymax/p/4973639.html
	T string `bencode:"t,omitempty"`
	//每一个消息都包含t关键字，它是一个代表了transactionID的字符串类型。
	//transactionID由请求node产生，并且回复中要包含回显该字段，所以回复可能对应一个节点的多个请求。
	Y string `bencode:"y,omitempty"`
	// y必带，对应的值有三种情况：q表示请求，r表示回复，e表示错误。
	Q string `bencode:"q,omitempty"`
	// 如果y参数是"q", 则附加"q"和"a"。"q"参数指定查询类型：ping,find_node,get_peers,announce_peer
	A map[string]string `bencode:"a,omitempty"`
	// 关键字"a"一个字典类型包含了q请求所附加的参数。
	// 请求都包含一个关键字id，它包含了请求节点的nodeID
	// 其他数据字段key：
	// ping: 无
	// find_node: target
	// get_peers: info_hash
	// announce_peer: token、info_hash、port、implied_port

	// 1、token是一个短的二进制字符串。在get_peers回复包中产生。
	// 收到announce_peer请求的node必须检查这个token与之前我们回复给这个节点get_peers的token是否相同。
	// 如果相同，那么被请求的节点将记录发送announce_peer节点的IP和请求中包含的port端口号在peer联系信息中对应的infohash。
	// 记录用于下次get_peers
	// 2、info_hash，代表torrent文件的infohash。本质是文件名，文件长度，子文件信息
	// 3、target，包含了请求者正在查找的node的nodeID

	R map[string]string `bencode:"r,omitempty"`
	// 如果"y"关键字的值是“r”，则包含了一个附加的关键字r，r的值是一个字典类型。
	// 回复包也含关键字id，它包含了回复节点的nodeID。
	// 其他数据字段key：
	// ping: 无
	// find_node: nodes
	// get_peers: token、nodes、values
	// announce_peer: 无

	E []interface{} `bencode:"e,omitempty"`
	//错误信息包含一个附加的关键字e。关键字“e”是一个列表类型。
	//当一个请求不能解析或出错时，错误包将被发送。
	//第一个元素是一个数字类型，表明了错误码。
	//第二个元素是一个字符串类型，表明了错误信息。
}

type Client struct {
	peerInfo   *NodeInfo
	connection *net.UDPConn
	// mutex        sync.RWMutex
	// disconnected bool
	// ToFindAddrs  map[string]int
	port       string
	targetAddr string
}

func NewClient(port string, targetAddr string) *Client {
	myAddr, _ := getRemoteIP()
	addr, err := net.ResolveUDPAddr("udp4", myAddr+":"+port)
	if err != nil {
		log.Print(err)
		return nil
	}
	log.Printf("NewClient port=%v, addr=%v", port, addr)
	id := newId(getMacAddrs()[0] + port)
	log.Printf("newId len: %v, newId data: %x", len(id), id)
	return &Client{
		// disconnected: false,
		&NodeInfo{
			ID:   string(id),
			addr: addr,
		},
		nil,
		port,
		targetAddr,
		// ToFindAddrs:  map[string]int{},
	}
}
func (client *Client) ID() string {
	return client.peerInfo.ID
}

func (client *Client) Start() error {
	err := client.ListenUDP()
	if err != nil {
		log.Print(err)
		return err
	}
	go client.recv()
	go client.sendTimer()
	return err
}

func (client *Client) ListenUDP() error {
	s, err := net.ResolveUDPAddr("udp4", ":"+client.port)
	if err != nil {
		log.Print(err)
		return err
	}
	connection, err := net.ListenUDP("udp4", s)
	if err != nil {
		log.Print(err)
		return err
	}
	client.connection = connection
	return err
}

//1、解码，2、响应请求，3、保存收包的地址，用于find，4、保存infohash
func (client *Client) recv() {
	buffer := make([]byte, 4096)
	for {
		// if client.disconnected {
		// 	return
		// }
		n, addr, err := client.connection.ReadFromUDP(buffer)
		if err != nil {
			log.Print(err)
			continue
		}
		log.Printf("recv from %v, data: %v", addr.String(), string(buffer[:n]))
		buff := bytes.NewBuffer(buffer[:n])
		var recvmsg structNested
		err = bencode.Unmarshal(buff, &recvmsg)
		if err != nil {
			log.Print(err)
			continue
		}
		log.Printf("Unmarshal ok %+v", recvmsg)
		//如果是来自本机则不回复
		// if client.ToFindAddrs[addr.String()] == 1 {
		// 	log.Printf("recv sendFindNode addr, do not echo back")
		// 	continue
		// }
		client.processMsg(recvmsg, addr)
	}
}
func (client *Client) processMsg(recvmsg structNested, addr *net.UDPAddr) error {
	log.Printf("processMsg msg:%v resp:%v query:%v", recvmsg.Y, recvmsg.R, recvmsg.Q)
	switch recvmsg.Y {
	case "q":
		{
			log.Printf("processMsg request arg:%v", recvmsg.A)
			if recvmsg.A["id"] != "" {
				//记录node_id + addr
				log.Printf("processMsg request Id:=%v, addr=%v", recvmsg.A["id"], addr.String())
			}
			resp := structNested{
				T: recvmsg.T,
				Y: "r",
				Q: "",
				A: nil,
				E: nil,
			}
			switch recvmsg.Q {
			case "ping":
				resp.R = map[string]string{
					"id": client.ID(),
				}
				client.sendMsg(resp, addr)
			case "find_node":
				log.Printf("find_node peerInfo: %+v", *client.peerInfo)
				nodes := CompactNodeInfo(client.peerInfo)
				log.Printf("request find_node nodes: %+v %+v", len(nodes), len(client.ID()))
				resp.R = map[string]string{
					"id":    client.ID(),
					"nodes": nodes,
				}
				client.sendMsg(resp, addr)
			case "get_peers":
				log.Printf("get_peers peerInfo: %+v", *client.peerInfo)
				nodes := CompactNodeInfo(client.peerInfo)
				log.Printf("request find_node nodes: %+v %+v", len(nodes), len(client.ID()))
				resp.R = map[string]string{
					"id":    client.ID(),
					"nodes": nodes,
				}
				client.sendMsg(resp, addr)
			case "announce_peer":
				resp.R = map[string]string{
					"id": client.ID(),
				}
				client.sendMsg(resp, addr)
			}
		}
	case "r":
		// 发送来的是响应，ping 或 find_node
		{
			log.Printf("processMsg response arg:%v", recvmsg.R)
			if recvmsg.R["id"] != "" {
				//记录node_id + addr
				log.Printf("processMsg response Id=%v, addr=%v, nodes=%v", recvmsg.R["id"], addr.String(), recvmsg.R["nodes"])
			}
			nodesMsg := recvmsg.R["nodes"]
			if len(nodesMsg)%26 != 0 || len(nodesMsg) == 0 {
				return nil
			}
			log.Printf("response msg nodes=%+x", nodesMsg)
			nodes := DecodeCompactNodesInfo(nodesMsg)
			if len(nodes) > 0 {
				log.Printf("response parsed, Id=%+v, addr=%+v", nodes[0].ID, nodes[0].addr)
			}
		}
	case "e":
		{
			log.Printf("processMsg error %v", recvmsg.E)
		}
	}

	return nil
}
