package dht

import (
	"net"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

func (client *Client) send(sendTable *NodeTable) {
	ticker := time.NewTicker(time.Second * 4)
	for {
		total := 0
		//bug: 避免短时间突发包
		for i := sendTable.curSendBucket; i < 160; i++ {
			buck := sendTable.sendBuckets[i]
			total += len(buck)
			for _, node := range buck {
				client.sendFindNode(client.ID(), node.addr)
			}
			if total > 100 {
				sendTable.curSendBucket = i + 1
				if sendTable.curSendBucket >= 160 {
					sendTable.curSendBucket = 1
				}
				break
			}
		}
		if total == 0 || client.IsTableOld(sendTable) {
			client.sendPrime()
		} else {
			logx.Infof("client sendFindNode total=%v", total)
		}
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
	logx.Infof("send host addr %v", resAddr)
	addr, err := net.ResolveUDPAddr(client.network, resAddr)
	if err != nil {
		logx.Infof("ResolveUDPAddr targetAddr[%v] err:%v", resAddr, err)
		return
	}
	// client.sendPing(addr)
	client.sendFindNode(client.ID(), addr)
	// client.sendGetPeer(client.ID(), addr)
	// client.sendAnnouncePeer(addr)
	// client.sendError(addr)
}
