package dht

import (
	"errors"
	"net"
	"strings"
)

type PeerInfo struct {
	ID   string
	addr *net.UDPAddr
}

func CompactNodeInfo(peer *PeerInfo) string {
	info, _ := encodeCompactIPPortInfo(peer.addr.IP, peer.addr.Port)
	return peer.ID + info
}

func CompactNodesInfo(peers []*PeerInfo) string {
	infos := make([]string, len(peers))
	for i, peer := range peers {
		infos[i] = CompactNodeInfo(peer)
	}
	return strings.Join(infos, "")
}

// newNodeFromCompactInfo parses compactNodeInfo and returns a node pointer.
func newNodeFromCompactInfo(compactNodeInfo string) (*PeerInfo, error) {

	if len(compactNodeInfo) != 26 {
		return nil, errors.New("compactNodeInfo should be a 26-length string")
	}

	id := compactNodeInfo[:20]
	ip, port, _ := decodeCompactIPPortInfo(compactNodeInfo[20:])

	addr, err := net.ResolveUDPAddr("udp4", genAddress(ip.String(), port))
	if err != nil {
		return nil, err
	}
	return &PeerInfo{ID: id, addr: addr}, nil
}

func decodeCompactNodesInfo(nodes string) []*PeerInfo {
	var peers []*PeerInfo
	for i := 0; i < len(nodes)/26; i++ {
		peer, _ := newNodeFromCompactInfo(string(nodes[i*26 : (i+1)*26]))
		peers = append(peers, peer)
	}
	return peers
}
