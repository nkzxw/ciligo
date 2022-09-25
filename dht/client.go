package dht

import (
	"bytes"
	"encoding/hex"
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
	//[distance] map[id][ip+port]
	buckets map[int][]*NodeInfo
}
type Client struct {
	peerInfo   *NodeInfo
	connection *net.UDPConn
	// mutex        sync.RWMutex
	// disconnected bool
	port string

	resolve6 string
	want     []string

	targetAddr string

	//两个路由表用于异步更新。
	//一个表发，一个表收
	sendTable *NodeTable
	recvTable *NodeTable

	//更新发包路由表条件
	sendTableUpdateTs time.Time
	updateSeconds     int
}

func NewClient(port string, targetAddr string, ipType string) *Client {
	myIP := "[::1]" + ":" + port
	resolve := "udp6"
	ipWant := []string{"n6"}
	if ipType == "4" {
		resolve = "udp4"
		ip, err := getRemoteIP()
		logx.Infof("remove ip:%v,err:%v", ip, err)
		myIP = getLocalIPs()[0] + ":" + port
		ipWant = []string{"n4"}
	}
	myAddr, err := net.ResolveUDPAddr(resolve, myIP)
	if err != nil {
		logx.Infof("err:%v", err)
		return nil
	}
	logx.Infof("NewClient port=%v, addr=%v", port, myAddr)
	id := newId(getMacAddrs()[0] + port)
	logx.Infof("newId len: %v, newId data: %x", len(id), id)
	return &Client{
		// disconnected: false,
		peerInfo: &NodeInfo{
			ID:   string(id),
			addr: myAddr,
		},
		connection:    nil,
		port:          port,
		targetAddr:    targetAddr,
		resolve6:      resolve,
		want:          ipWant,
		sendTable:     &NodeTable{buckets: make(map[int][]*NodeInfo, 160)},
		recvTable:     &NodeTable{buckets: make(map[int][]*NodeInfo, 160)},
		updateSeconds: 8,
	}
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
	go client.send()
	return err
}

func (client *Client) SearchFileInfo(infoHash string) error {
	data, err := hex.DecodeString(infoHash)
	if err != nil {
		return err
	}
	infoHash = string(data)
	for _, node := range client.GetClosest(infoHash) {
		client.sendGetPeer(infoHash, node.addr)
	}
	return nil
}
func (client *Client) ListenUDP() error {
	connection, err := net.ListenUDP(client.resolve6, client.peerInfo.addr)
	if err != nil {
		logx.Infof("err:%v", err)
		return err
	}
	client.connection = connection
	return err
}

func (client *Client) send() {
	ticker := time.NewTicker(time.Second * 4)
	for {
		total := 0
		//bug: 160*8 可能短时间突发包
		for _, buck := range client.sendTable.buckets {
			total += len(buck)
			for _, node := range buck {
				client.sendFindNode(client.ID(), node.addr)
			}
		}
		if total == 0 || client.IsTableOld() {
			client.sendPrime()
		} else {
			//bug：如果boot节点返回的节点不可用，会入网失败。
			//--长期没有更新发送表，使用一次boot节点
			logx.Infof("client.send() total=%v", total)
		}
		// ubuntu-14.04.2-desktop-amd64.iso
		client.SearchFileInfo("546cf15f724d19c4319cc17b179d7e035f89c1f4")
		// some movie
		// client.SearchFileInfo("32D9A70EB9E1AD7609C5A6913E8216CFFE95998E")
		<-ticker.C
	}
}

func (client *Client) sendPrime() {
	if client.targetAddr == "" {
		for _, resAddr := range PrimeNodes {
			client.sendCmd(resAddr)
		}
	} else {
		client.sendCmd(client.targetAddr)
	}
}

func (client *Client) sendCmd(resAddr string) {
	logx.Infof("sendCmd targetAddr %v", resAddr)
	addr, err := net.ResolveUDPAddr(client.resolve6, resAddr)
	if err != nil {
		logx.Infof("ResolveUDPAddr targetAddr[%v] err:%v", resAddr, err)
		return
	}
	// client.sendPing(addr)
	client.sendFindNode(client.ID(), addr)
	// client.sendAnnouncePeer(addr)
	// client.sendError(addr)
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
		// logx.Infof("Unmarshal ok %+v", recvmsg)
		client.processMsg(&recvmsg, addr)
	}
}
func (client *Client) processMsg(recvmsg *structNested, addr *net.UDPAddr) error {
	// 如果是来自本机则不回复
	// if client.ToFindAddrs[addr.String()] == 1 {
	// 	logx.Infof("recv sendFindNode addr, do not echo back")
	// 	return
	// }
	switch recvmsg.Y {
	case "q":
		// 发来的是请求
		{
			// logx.Infof("processMsg request arg:%+v", recvmsg.A)
			// if recvmsg.A.Id != "" {
			// 记录node_id + addr
			// logx.Infof("processMsg request Id:=%v, addr=%v", recvmsg.A.Id, addr.String())
			// }
			resp := &structNested{
				T: recvmsg.T,
				Y: "r",
			}
			switch recvmsg.Q {
			case "ping":
				client.sendPingResp(resp, addr)
			case "find_node":
				client.sendFindNodeResp(resp, addr)
			case "get_peers":
				//Token原样返回
				resp.R.Token = recvmsg.A.Token
				client.sendGetPeerResp(resp, addr)
			case "announce_peer":
				logx.Infof("announce_peer Implied_port: %+v", recvmsg.A.Implied_port)
				client.sendAnnouncePeerResp(resp, addr)
			}
		}
	case "r":
		// 发来的是响应
		{
			logx.Infof("response from:%v,t:%+v", addr.String(), recvmsg.T)
			// logx.Infof("response: %+v", recvmsg)
			// if recvmsg.R.Id != "" {
			// 记录node_id -> addr
			// logx.Infof("processMsg store Id=%v, addr=%v, nodes=%v", recvmsg.R.Id, addr.String(), recvmsg.R.Nodes)
			// }
			if len(recvmsg.R.Nodes) > 0 {
				nodes := DecodeCompactNodesInfo(recvmsg.R.Nodes)
				total := len(nodes)
				for i, node := range nodes {
					logx.Infof("response NodeInfo(%v/%v):%v", i+1, total, node.addr.String())
					client.UpdateRecvTable(node)
				}
			}
			if len(recvmsg.R.Nodes6) > 0 {
				logx.Infof("response Nodes6 len:%v", len(recvmsg.R.Nodes6))
				nodes := DecodeCompactNodesInfo(recvmsg.R.Nodes6)
				total := len(nodes)
				for i, node := range nodes {
					logx.Infof("response NodeInfo6(%v/%v):%v", i+1, total, node.addr.String())
				}
			}
			valuesMsg := recvmsg.R.Values
			logx.Infof("response values len:%v", len(valuesMsg))
		}
	case "e":
		{
			logx.Infof("processMsg error %v", recvmsg.E)
		}
	}
	return nil
}
