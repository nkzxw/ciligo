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

func randomTranssionId() string {
	var n uint16
	binary.Read(rand.Reader, binary.LittleEndian, &n)
	return strconv.Itoa(int(n))
}

func newId(seed string) []byte {
	h := sha1.New()
	io.WriteString(h, seed)
	b := h.Sum(nil)
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
func bytes2int(b []byte) uint16 {
	bytesBuffer := bytes.NewBuffer(b)
	var x uint16
	binary.Read(bytesBuffer, binary.BigEndian, &x)
	return x
}

// int2bytes returns the byte array it represents.
func int2bytes(val uint16) []byte {
	bytesBuffer := bytes.NewBuffer([]byte{})
	binary.Write(bytesBuffer, binary.BigEndian, val)
	return bytesBuffer.Bytes()
}

// decodeCompactIPPortInfo decodes compactIP-address/port info in BitTorrent
// DHT Protocol. It returns the ip and port number.
func decodeCompactIPPortInfo(info string) (ip net.IP, port uint16, err error) {
	if len(info) == 6 {
		ip = net.IP(info[0 : len(info)-2]).To4()
	}
	if len(info) == 18 {
		ip = net.IP(info[0 : len(info)-2]).To16()
	}
	port = bytes2int([]byte(info)[len(info)-2 : len(info)])
	// logx.Infof("decodeCompactIPPortInfo %x %v %v %x", []byte(info), ip, port, []byte(info)[4:6])
	return
}

// encodeCompactIPPortInfo encodes an ip and a port number to
// compactIP-address/port info.
func encodeCompactIPPortInfo(ip net.IP, port int) (info string, err error) {
	if port > 65535 || port < 0 {
		err = errors.New(
			"port should be no greater than 65535 and no less than 0")
		return
	}

	p := int2bytes(uint16(port))
	info = string(append(ip[len(ip)-4:], p...))
	// logx.Infof("encodeCompactIPPortInfo %x ip=%v p=%x info=%x", ip[0:4], ip.String(), p, []byte(info))
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

// genAddress returns a ip:port address.
func genAddress(ip net.IP, port uint16) string {
	if ip.To4() != nil {
		return strings.Join([]string{ip.String(), ":", strconv.Itoa(int(port))}, "")
	}
	return strings.Join([]string{"[", ip.String(), "]:", strconv.Itoa(int(port))}, "")
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

func calcDistance(ID string, ToID string) int {
	if len(ID) != 20 || len(ToID) != 20 {
		return 0
	}
	for i := 0; i < 20; i++ {
		if ID[i] != ToID[i] {
			bit := 0
			for bit = 0; bit < 8; bit++ {
				// fmt.Println(ID[i], ToID[i], ID[i]&(1<<(7-bit)), ToID[i]&(1<<(7-bit)), bit, i)
				if (ID[i] & (1 << (7 - bit))) != (ToID[i] & (1 << (7 - bit))) {
					break
				}
			}
			return 160 - i*8 - bit
		}
	}
	return 0
}
