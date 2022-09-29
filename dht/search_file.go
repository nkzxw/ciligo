package dht

import (
	"encoding/hex"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

func (client *Client) SearchFileInfo(infoHashs []string) {
	for _, search := range infoHashs {
		data, _ := hex.DecodeString(search)
		client.nodeTables[string(data)] = &NodeTable{
			recvBuckets: make(map[int][]*NodeInfo, 160),
			sendBuckets: make(map[int][]*NodeInfo, 160),
		}
		client.infoHashs = append(client.infoHashs, string(data))
	}
	go client.Search()
}
func (client *Client) Search() {
	ticker := time.NewTicker(time.Second * 4)
	for {
		for info, infoHash := range client.infoHashs {
			total := 0
			for i := 0; i < 160; i++ {
				buck := client.nodeTables[infoHash].sendBuckets[i]
				total += len(buck)
				for _, node := range buck {
					client.sendGetPeer(infoHash, node.addr)
				}
				if total > 8 {
					break
				}
			}
			logx.Infof("Search info:%v,total:%v,infoHash:%x", info, total, infoHash)
		}
		<-ticker.C
	}
}

// // ubuntu-14.04.2-desktop-amd64.iso
// client.SearchFileInfo("546cf15f724d19c4319cc17b179d7e035f89c1f4")
// // some movie
// // client.SearchFileInfo("32D9A70EB9E1AD7609C5A6913E8216CFFE95998E")
