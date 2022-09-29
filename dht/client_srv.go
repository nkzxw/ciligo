package dht

import (
	"bytes"
	"net"
	"time"

	bencode "github.com/jackpal/bencode-go"
	"github.com/zeromicro/go-zero/core/logx"
)

var (
	PrimeNodes = []string{
		"router.bittorrent.com:6881",
		"router.utorrent.com:6881",
		"dht.transmissionbt.com:6881",
	}
)

type NodeTable struct {
	//两个路由表用于异步更新。 一发一收
	//[distance] map[id][ip+port]
	recvBuckets   map[int][]*NodeInfo
	sendBuckets   map[int][]*NodeInfo
	curSendBucket int
	//更新发包路由表的时间
	sendTableTs time.Time
}
type Client struct {
	peerInfo   *NodeInfo // 不作为find_node和get_peer的结果返回
	connection *net.UDPConn
	// mutex        sync.RWMutex
	// disconnected bool
	port          string
	network       string
	want          []string
	targetAddr    string
	nodeTables    map[string]*NodeTable
	updateSeconds int
	// 测试getpeers
	infoHashs []string
}

func NewClient(port string, targetAddr string, ipType string) *Client {
	myIP := "[::1]" + ":" + port
	resolve := "udp6"
	ipWant := []string{"n6"}
	if ipType == "4" {
		resolve = "udp4"
		ip, err := getRemoteIP()
		logx.Infof("remote ip:%v,err:%v", ip, err)
		logx.Infof("local ip:%+v", getLocalIPs())
		myIP = ":" + port
		ipWant = []string{"n4"}
	}
	myAddr, err := net.ResolveUDPAddr(resolve, myIP)
	if err != nil {
		logx.Infof("err:%v", err)
		return nil
	}
	logx.Infof("NewClient port=%v, addr=%+v", port, myAddr)
	id := newId(getMacAddrs()[0] + port)
	logx.Infof("newId len: %v, newId data: %x", len(id), id)
	cli := &Client{
		// disconnected: false,
		peerInfo: &NodeInfo{
			ID:   string(id),
			addr: myAddr,
		},
		connection:    nil,
		port:          port,
		targetAddr:    targetAddr,
		network:       resolve,
		want:          ipWant,
		nodeTables:    make(map[string]*NodeTable),
		updateSeconds: 8,
	}
	cli.nodeTables[cli.peerInfo.ID] = &NodeTable{
		recvBuckets: make(map[int][]*NodeInfo, 160),
		sendBuckets: make(map[int][]*NodeInfo, 160),
	}
	return cli
}
func (client *Client) ID() string {
	return client.peerInfo.ID
}

func (client *Client) Start() error {
	err := client.ListenUDP()
	if err != nil {
		logx.Infof("err:%v", err)
		return err
	}
	go client.recv()
	go client.send(client.nodeTables[client.peerInfo.ID])
	return err
}

func (client *Client) ListenUDP() error {
	logx.Infof("ListenUDP addr:%v ", client.peerInfo.addr.String())
	connection, err := net.ListenUDP(client.network, client.peerInfo.addr)
	if err != nil {
		logx.Infof("ListenUDP err:%v", err)
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
			logx.Infof("err:%v", err)
			continue
		}
		// logx.Infof("recv data: %v", string(buffer[:n]))
		buff := bytes.NewBuffer(buffer[:n])
		var recvmsg structNested
		err = bencode.Unmarshal(buff, &recvmsg)
		if err != nil {
			logx.Infof("recv from %v Unmarshal fail", addr.String())
			continue
		}
		client.processMsg(&recvmsg, addr)
	}
}
func (client *Client) processMsg(recvmsg *structNested, addr *net.UDPAddr) error {
	switch recvmsg.Y {
	// 发来的是请求
	case "q":
		{
			if recvmsg.A.Id != "" {
				client.UpdateRecvTable(&NodeInfo{ID: recvmsg.A.Id, addr: addr})
			}
			resp := &structNested{
				T: recvmsg.T,
				Y: "r",
			}
			switch recvmsg.Q {
			case "ping":
				client.sendPingResp(resp, addr)
			case "find_node":
				logx.Infof("find_node from: %+v", addr.String())
				resp.R.Nodes = CompactNodesInfo(client.GetClosest(recvmsg.A.Target))
				client.sendFindNodeResp(resp, addr)
			case "get_peers":
				logx.Infof("get_peers from: %+v, infoHash:%x", addr.String(), recvmsg.A.Info_hash)
				resp.R.Token = recvmsg.A.Token
				resp.R.Values = EncodeValues(client.GetClosest(recvmsg.A.Target))
				client.sendGetPeerResp(resp, addr)
			case "announce_peer":
				logx.Infof("announce_peer Implied_port: %+v", recvmsg.A.Implied_port)
				client.sendAnnouncePeerResp(resp, addr)
			}
		}
	// 发来的是响应
	case "r":
		{
			logx.Infof("response from:%v,t:%+v", addr.String(), recvmsg.T)
			if len(recvmsg.R.Nodes) > 0 {
				nodes := DecodeCompactNodesInfo(recvmsg.R.Nodes)
				logx.Infof("response NodeInfo len:%v", len(nodes))
				for _, node := range nodes {
					client.UpdateRecvTable(node)
				}
			}
			if len(recvmsg.R.Nodes6) > 0 {
				nodes6 := DecodeCompactNodesInfo(recvmsg.R.Nodes6)
				logx.Infof("response NodeInfo6 len:%v", len(nodes6))
				for _, node := range nodes6 {
					client.UpdateRecvTable(node)
				}
			}
			if len(recvmsg.R.Values) > 0 {
				for i, addr := range DecodeCompactValues(recvmsg.R.Values) {
					logx.Infof("response values(%v/%v):%v", i+1, len(recvmsg.R.Values), addr.String())
				}
			}
		}
	case "e":
		{
			logx.Infof("processMsg error %v", recvmsg.E)
		}
	}
	return nil
}
