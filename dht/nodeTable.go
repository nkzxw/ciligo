package dht

import (
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

func (client *Client) IsTableOld() bool {
	if time.Since(client.lastUpdated) > time.Duration(client.updateSeconds)*time.Second {
		client.lastUpdated = time.Now()
		return true
	}
	return false
}

func (client *Client) UpdateSendTable() {
	if client.IsTableOld() {
		sendTable := &NodeTable{buckets: make(map[int][]*NodeInfo, 160)}
		for dis, buck := range client.recvTable.buckets {
			sendTable.buckets[dis] = make([]*NodeInfo, len(buck))
			copy(sendTable.buckets[dis], buck)
			logx.Infof("client.sendTable dis=%v, len=%v", dis, len(sendTable.buckets[dis]))
		}
		client.sendTable = sendTable
	}
}

func (client *Client) UpdateRecvTable(node *NodeInfo) {
	dis := calcDistance(client.peerInfo.ID, node.ID)
	client.recvTable.buckets[dis] = append(client.recvTable.buckets[dis], node)
	bucketLen := len(client.recvTable.buckets[dis])
	if bucketLen > 8 {
		client.recvTable.buckets[dis] = client.recvTable.buckets[dis][bucketLen-8 : bucketLen]
	}
	client.UpdateSendTable()
}

func (client *Client) GetClosest(hashInfo string) []*NodeInfo {
	dis := calcDistance(client.peerInfo.ID, hashInfo)
	return client.sendTable.buckets[dis]
}
