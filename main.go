package main

import (
	"flag"
	"log"
)

var port = flag.String("p", "8050", "listen port")
var targetAddr = flag.String("t", "", "send findnode addr")

func main() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	flag.Parse()
	log.Printf("main start listen port:%v, findnode addr:%v ", *port, *targetAddr)
	c := NewClient(*port, *targetAddr)
	c.Start()
	stop := make(chan int, 1)
	<-stop
}
