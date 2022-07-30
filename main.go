package main

import (
	"bytes"
	"flag"
	"log"
	"net"
	"sync"
	"time"

	bencode "github.com/jackpal/bencode-go"
)

var port = flag.String("p", "8050", "listen port")
var targetAddr = flag.String("t", "", "send findnode addr")

type Client struct {
	connection   *net.UDPConn
	mutex        sync.RWMutex
	disconnected bool
	ToFindAddrs  map[string]int
}
type structNested struct {
	T string            "bencode:t"
	Y string            "bencode:y"
	Q string            "bencode:q"
	A map[string]string "bencode:a"
}

func NewClient() *Client {
	return &Client{
		disconnected: false,
		ToFindAddrs:  map[string]int{},
	}
}
func (client *Client) Connect(port string) error {
	s, err := net.ResolveUDPAddr("udp4", ":"+port)
	if err != nil {
		log.Print(err)
		return err
	}
	connection, err := net.ListenUDP("udp4", s)
	if err != nil {
		log.Print(err)
		return err
	}
	client.connection = connection
	go client.recv()
	return err
}

func (client *Client) recv() {
	buffer := make([]byte, 4096)
	for {
		// if client.disconnected {
		// 	return
		// }
		n, addr, err := client.connection.ReadFromUDP(buffer)
		if err != nil {
			log.Print(err)
			continue
		}
		log.Printf("recv from %v, data: %v", addr.String(), string(buffer[:n]))
		if client.ToFindAddrs[addr.String()] == 1 {
			log.Printf("recv already know addr %v", addr.String())
			continue
		}
		n, err = client.connection.WriteToUDP(buffer[:n], addr)
		if err != nil {
			log.Print(n, err)
		}
		log.Printf("send back to %v", addr.String())
	}
}
func (client *Client) sendFindNode(targetAddr string) error {
	if targetAddr == "" {
		log.Print("sendFindNode targetAddr empty, return")
		return nil
	}
	unmarshalNestedDictionary := structNested{
		T: "aa",
		Y: "q",
		Q: "find_node",
		A: map[string]string{"id": "abcdefghij0123456789"},
	}
	// SVPair{"d1:ad2:id20:abcdefghij0123456789e1:q4:ping1:t2:aa1:y1:qe", unmarshalNestedDictionary},
	buf := new(bytes.Buffer)
	err := bencode.Marshal(buf, unmarshalNestedDictionary)
	if err != nil {
		log.Printf("Marshal err %v", err, buf)
		return err
	}
	log.Printf("Marshal ok %v", buf)
	addr, err := net.ResolveUDPAddr("udp4", targetAddr)
	if err != nil {
		log.Print(err)
		return err
	}
	client.ToFindAddrs[targetAddr] = 1
	log.Printf("sendFindNode %v", addr)
	n, err := client.connection.WriteToUDP(buf.Bytes(), addr)
	if err != nil {
		log.Print(n, err)
	}
	return err
}

func main() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	flag.Parse()
	log.Printf("main start, listen port:%v, findnode addr:%v ", *port, *targetAddr)
	ticker := time.NewTicker(time.Second * 5)
	c := NewClient()
	c.Connect(*port)
	stop := make(chan int, 1)
	go func() {
		for {
			<-ticker.C
			c.sendFindNode(*targetAddr)
		}
	}()
	<-stop
}
