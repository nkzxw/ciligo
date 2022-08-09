package dht

import (
	"bytes"
	"log"
	"net"
	"time"

	bencode "github.com/jackpal/bencode-go"
)

func (client *Client) sendTimer() {
	ticker := time.NewTicker(time.Second * 1)
	i := 0
	for {
		<-ticker.C
		if client.targetAddr == "" {
			log.Print("sendFindNode targetAddr empty, return")
			continue
		}
		addr, err := net.ResolveUDPAddr("udp4", client.targetAddr)
		if err != nil {
			log.Print(err)
			continue
		}
		if i%3 == 0 {
			client.sendPing(addr)
		}
		if i%3 == 1 {
			client.sendFindNode(addr)
		}
		if i%3 == 2 {
			client.sendAnnouncePeer(addr)
		}
		if i%3 == 3 {
			client.sendError(addr)
		}
		i++
	}
}

func (client *Client) sendFindNode(addr *net.UDPAddr) error {
	findNodeMsg := structNested{
		T: randomTranssionId(),
		Y: "q",
		Q: "find_node",
		A: RequestArg{
			Id:     client.ID(),
			Target: randomString(20),
		},
	}
	// SVPair{"d1:ad2:id20:abcdefghij0123456789e1:q4:ping1:t2:aa1:y1:qe", findNodeMsg},
	log.Printf("sendFindNode msg")
	return client.sendMsg(findNodeMsg, addr)
}

func (client *Client) sendPing(addr *net.UDPAddr) error {
	// 一般错误={"t":"aa", "y":"e", "e":[201,"A Generic Error Ocurred"]}
	// B编码=d1:eli201e23:AGenericErrorOcurrede1:t2:aa1:y1:ee
	pingMsg := structNested{
		T: randomTranssionId(),
		Y: "q",
		Q: "ping",
		A: RequestArg{
			Id: client.ID(),
		},
	}
	log.Printf("send ping msg")
	return client.sendMsg(pingMsg, addr)
}
func (client *Client) sendAnnouncePeer(addr *net.UDPAddr) error {
	// 一般错误={"t":"aa", "y":"e", "e":[201,"A Generic Error Ocurred"]}
	// B编码=d1:eli201e23:AGenericErrorOcurrede1:t2:aa1:y1:ee
	errMsg := structNested{
		T: randomTranssionId(),
		Y: "q",
		Q: "announce_peer",
		A: RequestArg{
			Id:           client.ID(),
			Token:        randomString(20),
			Info_hash:    randomString(20),
			Port:         8080,
			Implied_port: 8082,
		},
	}
	log.Printf("send announce_peer msg")
	return client.sendMsg(errMsg, addr)
}

func (client *Client) sendError(addr *net.UDPAddr) error {
	// 一般错误={"t":"aa", "y":"e", "e":[201,"A Generic Error Ocurred"]}
	// B编码=d1:eli201e23:AGenericErrorOcurrede1:t2:aa1:y1:ee
	errMsg := structNested{
		T: randomTranssionId(),
		Y: "e",
		E: []interface{}{201, "A Generic Error Ocurred"},
	}
	log.Printf("send Error msg")
	return client.sendMsg(errMsg, addr)
}

// map的tag变成了大写--fixed
// 空map也进行了编码 -- fixed
// 都是结构体tag问题
func (client *Client) sendMsg(msg structNested, addr *net.UDPAddr) error {
	// rmsg := reflect.ValueOf(msg)
	// typ := rmsg.Type()
	// field := typ.Field(1)
	// tag := field.Tag
	// key := tag.Get("bencode")
	// log.Printf("key: %v", key)
	// log.Printf("tag: %v", tag)
	buf := new(bytes.Buffer)
	err := bencode.Marshal(buf, msg)
	if err != nil {
		log.Printf("Marshal err:%v", err)
		return err
	}
	log.Printf("Marshal ok %v", buf)
	log.Printf("sendMsg to addr:%v", addr)
	// client.ToFindAddrs[addr.String()] = 1
	n, err := client.connection.WriteToUDP(buf.Bytes(), addr)
	if err != nil {
		log.Print(n, err)
	}
	return err
}
