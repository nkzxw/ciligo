package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

	"ogg.com/gocili/dht"
)

var port = flag.String("p", "8050", "listen port")
var targetAddr = flag.String("t", "", "send findnode addr")

func initLog() error {
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	fmt.Println("start ", os.Getpid())
	f, err := os.OpenFile("./log/log-"+strconv.Itoa(os.Getpid())+".log", os.O_CREATE|os.O_APPEND|os.O_RDWR, os.ModePerm)
	if err != nil {
		return err
	}
	// log.SetPrefix("[" + strconv.Itoa(os.Getpid()) + "]")
	log.SetOutput(f)
	return nil
}

func main() {
	flag.Parse()
	if initLog() != nil {
		return
	}
	log.Print(os.Args)
	log.Printf("main start listen port:%v, findnode addr:%v ", *port, *targetAddr)
	c := dht.NewClient(*port, *targetAddr)
	c.Start()
	stop := make(chan int, 1)
	<-stop
}
