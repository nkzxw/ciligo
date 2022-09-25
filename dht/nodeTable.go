package dht

import (
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

func (client *Client) IsTableOld() bool {
	return time.Since(client.sendTableUpdateTs) > time.Duration(client.updateSeconds)*time.Second
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
		client.sendTableUpdateTs = time.Now()
	}
}

func (client *Client) UpdateRecvTable(node *NodeInfo) {
	dis := calcDistance(client.peerInfo.ID, node.ID)
	if dis == 0 {
		logx.Infof("client.sendTable dis=%v, node=%v, ID=%v", dis, node.addr.String(), node.ID)
	}
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
