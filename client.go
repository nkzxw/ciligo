package main

import (
	"bytes"
	"log"
	"net"
	"sync"
	"time"

	bencode "github.com/jackpal/bencode-go"
)

type Client struct {
	connection   *net.UDPConn
	mutex        sync.RWMutex
	disconnected bool
	ToFindAddrs  map[string]int
	port         string
	targetAddr   string
}
type structNested struct {
	T string            "bencode:t"
	Y string            "bencode:y"
	Q string            "bencode:q"
	A map[string]string "bencode:a"
}

func NewClient(port string, targetAddr string) *Client {
	return &Client{
		disconnected: false,
		port:         port,
		targetAddr:   targetAddr,
		ToFindAddrs:  map[string]int{},
	}
}

func (client *Client) Start() error {
	err := client.ListenUDP()
	if err != nil {
		log.Print(err)
		return err
	}
	go client.recv()
	go client.sendTimer()
	return err
}

func (client *Client) ListenUDP() error {
	s, err := net.ResolveUDPAddr("udp4", ":"+client.port)
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
	return err
}
func (client *Client) sendTimer() {
	ticker := time.NewTicker(time.Second * 5)
	for {
		<-ticker.C
		client.sendFindNode()
	}
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
			log.Printf("recv sendFindNode addr, do not echo back")
			continue
		}
		n, err = client.connection.WriteToUDP(buffer[:n], addr)
		if err != nil {
			log.Printf("WriteToUDP n:%v, err:%v", n, err)
		}
		log.Printf("send back to %v", addr.String())
	}
}
func (client *Client) sendFindNode() error {
	if client.targetAddr == "" {
		log.Print("sendFindNode targetAddr empty, return")
		return nil
	}
	findNodeMsg := structNested{
		T: "aa",
		Y: "q",
		Q: "find_node",
		A: map[string]string{"id": "abcdefghij0123456789"},
	}
	// SVPair{"d1:ad2:id20:abcdefghij0123456789e1:q4:ping1:t2:aa1:y1:qe", findNodeMsg},
	buf := new(bytes.Buffer)
	err := bencode.Marshal(buf, findNodeMsg)
	if err != nil {
		log.Printf("Marshal err:%v", err)
		return err
	}
	log.Printf("Marshal ok %v", buf)
	addr, err := net.ResolveUDPAddr("udp4", client.targetAddr)
	if err != nil {
		log.Print(err)
		return err
	}
	log.Printf("sendFindNode to addr:%v", addr)
	client.ToFindAddrs[addr.String()] = 1
	n, err := client.connection.WriteToUDP(buf.Bytes(), addr)
	if err != nil {
		log.Print(n, err)
	}
	return err
}
