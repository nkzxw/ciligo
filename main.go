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
}
type structNested struct {
	T string            "bencode:t"
	Y string            "bencode:y"
	Q string            "bencode:q"
	A map[string]string "bencode:a"
}

func NewClient() *Client {
	return &Client{disconnected: false}
}
func (client *Client) Connect() error {
	s, err := net.ResolveUDPAddr("udp4", ":1111")
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
		n, _, err := client.connection.ReadFromUDP(buffer)
		if err != nil {
			log.Print(err)
			continue
		}
		log.Print("recv")
		log.Print(n, buffer[:n])
	}
}
func (client *Client) sendFindNode() error {
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
		log.Print(err, buf)
		return err
	}
	log.Print(err, buf)
	addr, err := net.ResolveUDPAddr("udp4", "192.168.230.133:1111")
	if err != nil {
		log.Print(err)
	}
	n, err := client.connection.WriteToUDP(buf.Bytes(), addr)
	if err != nil {
		log.Print(n, err)
	}
	return err
}

func main() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	log.Print("start")
	ticker := time.NewTicker(time.Second * 5)
	// i := 0
	c := NewClient()
	c.Connect()
	stop := make(chan int, 1)
	go func() {
		for {
			<-ticker.C
			c.sendFindNode()
			// i++
			// fmt.Println("i = ", i)
			// if i == 1 {
			// 	stop <- 1
			// }
		}
	}()
	<-stop
}
