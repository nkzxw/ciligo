package dht

import (
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

func (client *Client) IsTableOld(table *NodeTable) bool {
	return time.Since(table.sendTableTs) > time.Duration(client.updateSeconds)*time.Second
}

func (client *Client) UpdateSendTable(table *NodeTable) {
	if client.IsTableOld(table) {
		sendBuckets := make(map[int][]*NodeInfo, 160)
		total := 0
		for dis, buck := range table.recvBuckets {
			sendBuckets[dis] = make([]*NodeInfo, len(buck))
			total += len(buck)
			copy(sendBuckets[dis], buck)
		}
		logx.Infof("UpdateSendTable nodes total=%v", total)
		table.sendBuckets = sendBuckets
		table.sendTableTs = time.Now()
	}
}

func (client *Client) UpdateRecvTable(node *NodeInfo) {
	for id, buck := range client.nodeTables {
		dis := calcDistance(id, node.ID)
		if dis == 0 {
			logx.Infof("not Update myID addr=%v,ID=%x", node.addr.String(), node.ID)
			return
		}
		for _, nod := range buck.recvBuckets[dis] {
			if nod.ID == node.ID {
				return
			}
		}
		buck.recvBuckets[dis] = append(buck.recvBuckets[dis], node)
		bucketLen := len(buck.recvBuckets[dis])
		if bucketLen > 8 {
			buck.recvBuckets[dis] = buck.recvBuckets[dis][bucketLen-8 : bucketLen]
		}
		client.UpdateSendTable(buck)
	}
}

func (client *Client) GetClosest(hashInfo string) []*NodeInfo {
	dis := calcDistance(client.peerInfo.ID, hashInfo)
	return client.nodeTables[client.peerInfo.ID].recvBuckets[dis]
}
