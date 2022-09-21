package dht

import (
	"errors"
	"net"
	"strings"
)

type NodeInfo struct {
	ID   string
	addr *net.UDPAddr
}

//只支持ipv4，ipv6是隔离的
// DHT IPV6 格式
// http://www.bittorrent.org/beps/bep_0032.html

func CompactNodesInfo(nodes []*NodeInfo) string {
	infos := make([]string, len(nodes))
	for i, node := range nodes {
		infos[i] = CompactNodeInfo(node)
	}
	return strings.Join(infos, "")
}

func CompactNodeInfo(node *NodeInfo) string {
	// logx.Infof("node.addr=%+v", node.addr)
	info, _ := encodeCompactIPPortInfo(node.addr.IP, node.addr.Port)
	return node.ID + info
}

func DecodeCompactNodesInfo(nodes string) []*NodeInfo {
	var nodesInfo []*NodeInfo
	for i := 0; i < len(nodes)/26; i++ {
		node, _ := DecodeCompactNodeInfo(string(nodes[i*26 : (i+1)*26]))
		nodesInfo = append(nodesInfo, node)
	}
	return nodesInfo
}

func DecodeCompactNodeInfo(compactNodeInfo string) (*NodeInfo, error) {
	if len(compactNodeInfo) != 26 {
		return nil, errors.New("compactNodeInfo should be a 26-length string")
	}
	id := compactNodeInfo[:20]
	ip, port, _ := decodeCompactIPPortInfo(compactNodeInfo[20:])

	addr, err := net.ResolveUDPAddr("udp4", genAddress(ip.String(), port))
	if err != nil {
		return nil, err
	}
	return &NodeInfo{ID: id, addr: addr}, nil
}
