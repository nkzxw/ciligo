package dht

import (
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// randomString generates a size-length string randomly.
func randomString(size int) string {
	buff := make([]byte, size)
	rand.Read(buff)
	return string(buff)
}

func newId(seed string) []byte {
	h := sha1.New()
	io.WriteString(h, seed)
	b := h.Sum(nil)
	log.Printf("data: %x", b)
	log.Printf("len: %v", len(b))
	return b
}

// // bytes2int returns the int value it represents.
// func bytes2int(data []byte) uint64 {
// 	n, val := len(data), uint64(0)
// 	if n > 8 {
// 		panic("data too long")
// 	}

// 	for i, b := range data {
// 		val += uint64(b) << uint64((n-i-1)*8)
// 	}
// 	return val
// }

//字节转换成整形
func bytes2int(b []byte) int16 {
	bytesBuffer := bytes.NewBuffer(b)

	var x int32
	binary.Read(bytesBuffer, binary.BigEndian, &x)

	return int16(x)
}

// int2bytes returns the byte array it represents.
func int2bytes(val int16) []byte {
	bytesBuffer := bytes.NewBuffer([]byte{})
	binary.Write(bytesBuffer, binary.BigEndian, val)
	return bytesBuffer.Bytes()
}

// decodeCompactIPPortInfo decodes compactIP-address/port info in BitTorrent
// DHT Protocol. It returns the ip and port number.
func decodeCompactIPPortInfo(info string) (ip net.IP, port int, err error) {
	if len(info) != 6 {
		err = errors.New("compact info should be 6-length long")
		return
	}

	ip = net.IPv4(info[0], info[1], info[2], info[3])
	port = int((uint16(info[4]) << 8) | uint16(info[5]))
	return
}

// genAddress returns a ip:port address.
func genAddress(ip string, port int) string {
	return strings.Join([]string{ip, strconv.Itoa(port)}, ":")
}

// encodeCompactIPPortInfo encodes an ip and a port number to
// compactIP-address/port info.
func encodeCompactIPPortInfo(ip net.IP, port int) (info string, err error) {
	if port > 65535 || port < 0 {
		err = errors.New(
			"port should be no greater than 65535 and no less than 0")
		return
	}

	p := int2bytes(int16(port))
	info = string(append(ip, p...))
	return
}

// getLocalIPs returns local ips.
func getLocalIPs() (ips []string) {
	ips = make([]string, 0, 6)

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return
	}

	for _, addr := range addrs {
		ip, _, err := net.ParseCIDR(addr.String())
		if err != nil {
			continue
		}
		ips = append(ips, ip.String())
	}
	return
}
func getMacAddrs() (macAddrs []string) {
	netInterfaces, err := net.Interfaces()
	if err != nil {
		fmt.Printf("fail to get net interfaces: %v", err)
		return macAddrs
	}
	for _, netInterface := range netInterfaces {
		macAddr := netInterface.HardwareAddr.String()
		if len(macAddr) == 0 {
			continue
		}
		macAddrs = append(macAddrs, macAddr)
	}
	return macAddrs
}

// getRemoteIP returns the wlan ip.
func getRemoteIP() (ip string, err error) {
	client := &http.Client{
		Timeout: time.Second * 30,
	}

	req, err := http.NewRequest("GET", "http://ifconfig.me", nil)
	if err != nil {
		return
	}

	req.Header.Set("User-Agent", "curl")
	res, err := client.Do(req)
	if err != nil {
		return
	}

	defer res.Body.Close()

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return
	}
	ip = string(data)

	return
}
