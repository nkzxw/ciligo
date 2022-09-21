package dht

import (
	"bytes"
	"log"
	"net"
	"time"

	bencode "github.com/jackpal/bencode-go"
	"github.com/zeromicro/go-zero/core/logx"
)

type Client struct {
	peerInfo   *NodeInfo
	connection *net.UDPConn
	// mutex        sync.RWMutex
	// disconnected bool
	// ToFindAddrs  map[string]int
	port       string
	targetAddr string
	resolve6   string
}

func NewClient(port string, targetAddr string, ipType string) *Client {
	myIP, err := getRemoteIP()
	if err != nil {
		log.Print(err)
		return nil
	}
	resolve := "udp4"
	if ipType == "6" {
		resolve = "udp6"
	}
	myAddr, err := net.ResolveUDPAddr(resolve, myIP+":"+port)
	if err != nil {
		log.Print(err)
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
		connection: nil,
		port:       port,
		targetAddr: targetAddr,
		resolve6:   resolve,
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
	s, err := net.ResolveUDPAddr(client.resolve6, ":"+client.port)
	if err != nil {
		log.Print(err)
		return err
	}
	connection, err := net.ListenUDP(client.resolve6, s)
	if err != nil {
		log.Print(err)
		return err
	}
	client.connection = connection
	return err
}

// main 工作
func (client *Client) sendTimer() {
	ticker := time.NewTicker(time.Second * 3)
	i := 0
	for {
		<-ticker.C
		resAddr := client.targetAddr
		if resAddr == "" {
			resAddr = PrimeNodes[i%3]
			logx.Infof("targetAddr use %v", resAddr)
			i++
		}
		addr, err := net.ResolveUDPAddr(client.resolve6, resAddr)
		if err != nil {
			logx.Infof("ResolveUDPAddr targetAddr[%v] err:%v", resAddr, err)
			return
		}
		client.sendPing(addr)
		client.sendFindNode(addr)
		// client.sendAnnouncePeer(addr)
		// client.sendError(addr)
	}
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
			if recvmsg.A.Id != "" {
				//记录node_id + addr
				// logx.Infof("processMsg request Id:=%v, addr=%v", recvmsg.A.Id, addr.String())
			}
			resp := &structNested{
				T: recvmsg.T,
				Y: "r",
				R: map[string]string{},
			}
			switch recvmsg.Q {
			case "ping":
				client.sendPingResp(resp, addr)
			case "find_node":
				client.sendFindNodeResp(resp, addr)
			case "get_peers":
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
			logx.Infof("response: %+v", recvmsg)
			if recvmsg.R["id"] != "" {
				// 记录node_id -> addr
				// logx.Infof("processMsg store Id=%v, addr=%v, nodes=%v", recvmsg.R["id"], addr.String(), recvmsg.R["nodes"])
			}
			nodesMsg := recvmsg.R["nodes"]
			if len(nodesMsg)%26 == 0 && len(nodesMsg) != 0 {
				nodes := DecodeCompactNodesInfo(nodesMsg)
				total := len(nodes)
				for i, node := range nodes {
					logx.Infof("response NodeInfo(%v/%v):%v", i+1, total, node.addr.String())
				}
			}
			valuesMsg := recvmsg.R["values"]
			logx.Infof("response values:%v", valuesMsg)
		}
	case "e":
		{
			logx.Infof("processMsg error %v", recvmsg.E)
		}
	}
	return nil
}
