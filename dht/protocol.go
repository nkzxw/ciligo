package dht

import (
	"bytes"
	"net"

	bencode "github.com/jackpal/bencode-go"
	"github.com/zeromicro/go-zero/core/logx"
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

// utp协议-实现拥塞控制
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

type RequestArg struct {
	// 请求都包含一个关键字id，为请求节点的nodeID
	// 其他数据字段key：
	// ping: 无
	// find_node: target
	// get_peers: info_hash
	// announce_peer: token + info_hash + port + implied_port
	Id    string `bencode:"id,omitempty"`
	Token string `bencode:"token,omitempty"`
	// 1、token是一个短的二进制字符串。在get_peers回复包中产生。
	// 收到announce_peer请求的node必须检查这个token与之前我们回复给这个节点get_peers的token是否相同。
	// 如果相同，那么被请求的节点将记录发送announce_peer节点的IP和请求中包含的port端口号在peer联系信息中对应的infohash。
	Info_hash string `bencode:"info_hash,omitempty"`
	// 2、info_hash，代表torrent文件的infohash。本质是文件名，文件长度，子文件信息
	Port   uint64 `bencode:"port,omitempty"`
	Target string `bencode:"target,omitempty"`
	// 3、target，包含了请求者正在查找的node的nodeID。
	Implied_port uint64 `bencode:"implied_port,omitempty"`
	// 4、implied_port 如果不为0，使用收包socket端口作为tcp连接端口。否则使用port字段
	Want []string `bencode:"want,omitempty"`
	//The "want" parameter is allowed in the find_nodes and get_peers requests,
	// and governs the presence or absence of the "nodes" and "nodes6" parameters in the requested reply.
	// Its value is a list of one or more strings, which may include
	// "n4": the node requests the presence of a "nodes" key;
	// "n6": the node requests the presence of a "nodes6" key.
}

type ResponseInfo struct {
	// 回复包含关键字id，它包含了回复节点的nodeID。其他：
	// ping: 无
	// find_node: nodes
	// get_peers: token、nodes、values
	// announce_peer: 无
	Id     string `bencode:"id,omitempty"`
	Token  string `bencode:"token,omitempty"`
	Nodes  string `bencode:"nodes,omitempty"`
	Nodes6 string `bencode:"nodes6,omitempty"`
	// nodes是string，n个26个字节拼接，每个代表nodeID+ip+port
	Values []string `bencode:"values,omitempty"`
	// values是list，n个6字节，每个代表ip+port
}
type structNested struct {
	//https://www.cnblogs.com/bymax/p/4973639.html
	T string `bencode:"t,omitempty"`
	//每一个消息都包含t关键字，它是一个代表了transactionID的字符串类型。
	//transactionID由请求node产生，并且回复中要包含回显该字段，所以回复可能对应一个节点的多个请求。
	Y string `bencode:"y,omitempty"`
	// y必带，对应的值有三种情况：q表示请求，r表示回复，e表示错误。
	Q string `bencode:"q,omitempty"`
	// 如果y参数是"q", 则附加"a"和"q"。
	// "q"参数指定查询类型：ping,find_node,get_peers,announce_peer
	A RequestArg `bencode:"a,omitempty"`
	// 关键字"a"一个字典类型，包含了q请求所附加的参数。见RequestArg结构说明

	R ResponseInfo  `bencode:"r,omitempty"`
	E []interface{} `bencode:"e,omitempty"`
	//当一个请求不能解析或出错时，错误包将被发送。
}

// ping Query = {"t":"aa", "y":"q", "q":"ping", "a":{"id":"abcdefghij0123456789"}}
func (client *Client) sendPing(addr *net.UDPAddr) error {
	msg := &structNested{
		T: randomTranssionId(),
		Y: "q",
		Q: "ping",
		A: RequestArg{
			Id: client.ID(),
		},
	}
	logx.Infof("sendPing :%v, t:%v", addr, msg.T)
	return client.sendMsg(msg, addr)
}

// Response = {"t":"aa", "y":"r", "r": {"id":"mnopqrstuvwxyz123456"}}
func (client *Client) sendPingResp(resp *structNested, addr *net.UDPAddr) error {
	resp.R.Id = client.ID()
	return client.sendMsg(resp, addr)
}

// find_node Query = {"t":"aa", "y":"q", "q":"find_node",
// "a": {"id":"abcdefghij0123456789", "target":"mnopqrstuvwxyz123456"}}
func (client *Client) sendFindNode(Target string, addr *net.UDPAddr) error {
	msg := &structNested{
		T: randomTranssionId(),
		Y: "q",
		Q: "find_node",
		A: RequestArg{
			Id:     client.ID(),
			Target: Target,
			Want:   client.want,
		},
	}
	logx.Infof("sendFindNode:%v, t:%v", addr, msg.T)
	return client.sendMsg(msg, addr)
}

// Response = {"t":"aa", "y":"r", "r": {"id":"0123456789abcdefghij", "nodes": "def456..."}}
func (client *Client) sendFindNodeResp(resp *structNested, addr *net.UDPAddr) error {
	resp.R.Id = client.ID()
	return client.sendMsg(resp, addr)
}

// get_peers Query = {"t":"aa", "y":"q", "q":"get_peers", "a": {"id":"abcdefghij0123456789", "info_hash":"mnopqrstuvwxyz123456"}}
func (client *Client) sendGetPeer(Info_hash string, addr *net.UDPAddr) error {
	msg := &structNested{
		T: randomTranssionId(),
		Y: "q",
		Q: "get_peers",
		A: RequestArg{
			Id:        client.ID(),
			Info_hash: Info_hash,
			Want:      client.want,
		},
	}
	logx.Infof("sendGetPeer:%v, t:%v", addr, msg.T)
	return client.sendMsg(msg, addr)
}

// Response with peers = {"t":"aa", "y":"r", "r": {"id":"abcdefghij0123456789", "token":"aoeusnth", "values": ["axje.u", "idhtnm"]}}
// Response with nodes = {"t":"aa", "y":"r", "r": {"id":"abcdefghij0123456789", "token":"aoeusnth", "nodes": "def456..."}}
func (client *Client) sendGetPeerResp(resp *structNested, addr *net.UDPAddr) error {
	logx.Infof("get_peers reply my addr: %+v", client.peerInfo.addr.String())
	resp.R.Id = client.ID()
	return client.sendMsg(resp, addr)
}

// announce_peers Query = {"t":"aa", "y":"q", "q":"announce_peer", "a": {"id":"abcdefghij0123456789", "implied_port": 1, "info_hash":"mnopqrstuvwxyz123456", "port": 6881, "token": "aoeusnth"}}
func (client *Client) sendAnnouncePeer(addr *net.UDPAddr) error {
	msg := &structNested{
		T: randomTranssionId(),
		Y: "q",
		Q: "announce_peer",
		A: RequestArg{
			Id:           client.ID(),
			Token:        randomString(20),
			Info_hash:    randomString(20),
			Port:         8080,
			Implied_port: 8082,
		},
	}
	logx.Infof("sendAnnouncePeer :%v, t:%v", addr, msg.T)
	return client.sendMsg(msg, addr)
}

// Response = {"t":"aa", "y":"r", "r": {"id":"mnopqrstuvwxyz123456"}}
func (client *Client) sendAnnouncePeerResp(resp *structNested, addr *net.UDPAddr) error {
	resp.R.Id = client.ID()
	return client.sendMsg(resp, addr)
}

// generic error = {"t":"aa", "y":"e", "e":[201, "A Generic Error Ocurred"]}
func (client *Client) sendError(addr *net.UDPAddr) error {
	msg := &structNested{
		T: randomTranssionId(),
		Y: "e",
		E: []interface{}{201, "A Generic Error Ocurred"},
	}
	logx.Infof("sendError :%v, t:%v", addr, msg.T)
	return client.sendMsg(msg, addr)
}

//1.
// map的tag变成了大写--fixed
// 空map也进行了编码 -- fixed
// 都是因为结构体tag格式问题
// 2.
// 192.168.0.101不能向 127发包
func (client *Client) sendMsg(msg *structNested, addr *net.UDPAddr) error {
	// rmsg := reflect.ValueOf(msg)
	// typ := rmsg.Type()
	// field := typ.Field(1)
	// tag := field.Tag
	// key := tag.Get("bencode")
	// logx.Infof("key: %v", key)
	// logx.Infof("tag: %v", tag)
	buf := new(bytes.Buffer)
	err := bencode.Marshal(buf, *msg)
	if err != nil {
		logx.Infof("Marshal err:%v", err)
		return err
	}
	n, err := client.connection.WriteToUDP(buf.Bytes(), addr)
	if err != nil {
		logx.Infof("WriteToUDP n:%v,err:%v", n, err)
	}
	return err
}
