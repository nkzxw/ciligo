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
		buckets := make(map[int][]*NodeInfo, 160)
		for dis, buck := range table.buckets {
			buckets[dis] = make([]*NodeInfo, len(buck))
			copy(buckets[dis], buck)
			logx.Infof("client.sendTable dis=%v, len=%v", dis, len(buckets[dis]))
		}
		table.sendBuckets = buckets
		table.sendTableTs = time.Now()
	}
}

func (client *Client) UpdateRecvTable(node *NodeInfo) {
	for id, buck := range client.nodeTables {
		dis := calcDistance(id, node.ID)
		if dis == 0 {
			logx.Infof("client.sendTable dis=%v, node=%v, ID=%v", dis, node.addr.String(), node.ID)
			return
		}
		buck.buckets[dis] = append(buck.buckets[dis], node)
		bucketLen := len(buck.buckets[dis])
		if bucketLen > 8 {
			buck.buckets[dis] = buck.buckets[dis][bucketLen-8 : bucketLen]
		}
		client.UpdateSendTable(buck)
	}
}

// func (client *Client) GetClosest(hashInfo string) []*NodeInfo {
// 	dis := calcDistance(client.peerInfo.ID, hashInfo)
// 	return client.sendTable.buckets[dis]
// }
