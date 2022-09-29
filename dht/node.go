package dht

import (
	"net"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"
)

type NodeInfo struct {
	ID   string
	addr *net.UDPAddr
}

// DHT IPV6 格式
// http://www.bittorrent.org/beps/bep_0032.html

func CompactNodesInfo(nodes []*NodeInfo) string {
	var infos []string
	for _, node := range nodes {
		info, _ := encodeCompactIPPortInfo(node.addr.IP, node.addr.Port)
		infos = append(infos, node.ID+info)
	}
	return strings.Join(infos, "")
}

func DecodeCompactNodesInfo(nodes string) []*NodeInfo {
	var nodesInfo []*NodeInfo
	size := 0
	if len(nodes)%38 == 0 {
		size = 38
	} else if len(nodes)%26 == 0 {
		size = 26
	} else {
		return nodesInfo
	}
	for i := 0; i < len(nodes)/size; i++ {
		node, err := DecodeCompactNodeInfo(string(nodes[i*size : (i+1)*size]))
		if err != nil {
			continue
		}
		nodesInfo = append(nodesInfo, node)
	}
	return nodesInfo
}

func DecodeCompactNodeInfo(compactNodeInfo string) (*NodeInfo, error) {
	id := compactNodeInfo[:20]
	ip, port, _ := decodeCompactIPPortInfo(compactNodeInfo[20:])
	ipType := "udp4"
	if len(compactNodeInfo) != 26 {
		ipType = "udp6"
	}
	addr, err := net.ResolveUDPAddr(ipType, genAddress(ip, port))
	if err != nil {
		logx.Infof("DecodeCompactNodeInfo ipType=%v err=%v", ipType, err)
		return nil, err
	}
	return &NodeInfo{ID: id, addr: addr}, nil
}

func EncodeValues(nodes []*NodeInfo) (infos []string) {
	for _, node := range nodes {
		info, _ := encodeCompactIPPortInfo(node.addr.IP, node.addr.Port)
		infos = append(infos, info)
	}
	return infos
}

func DecodeCompactValues(nodes []string) []*net.UDPAddr {
	var nodesInfo []*net.UDPAddr
	for _, val := range nodes {
		ip, port, _ := decodeCompactIPPortInfo(val)
		ipType := "udp4"
		if len(val) != 6 {
			ipType = "udp6"
		}
		addr, err := net.ResolveUDPAddr(ipType, genAddress(ip, port))
		if err != nil {
			logx.Infof("DecodeCompactNodeInfo ipType=%v err=%v", ipType, err)
			continue
		}
		nodesInfo = append(nodesInfo, addr)
	}
	return nodesInfo
}
