package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zxw/ciligo/dht"
)

var (
	version          = "v1.0.1"
	port             = flag.String("p", "8050", "listen port")
	targetAddr       = flag.String("a", "", "send findnode addr")
	ipv46            = flag.String("t", "4", "4/6")
	showVer    *bool = flag.Bool("v", false, "to show version of mini_datapipe")
)

func initInerLog() {
	log.SetFlags(log.Lshortfile | log.Lmicroseconds | log.Ldate)
	fmt.Println("start ", os.Getpid())
	f, err := os.OpenFile("./log/log-"+strconv.Itoa(os.Getpid())+".log", os.O_CREATE|os.O_APPEND|os.O_RDWR, os.ModePerm)
	if err != nil {
		return
	}
	log.SetOutput(f)

}
func initLog() error {
	// initInerLog()
	lc := logx.LogConf{Path: "./log/" + strconv.Itoa(os.Getpid()), Mode: "file"}
	err := logx.SetUp(lc)
	if err != nil {
		return err
	}
	logx.Infof("setup ok")
	logx.Errorf("setup ok")
	logx.Slowf("Slowf ok")
	logx.Statf("Statf ok")
	logx.Severef("Statf ok")
	return err
}

func main() {
	flag.Parse()

	if *showVer {
		fmt.Printf("%s\n", version)
		return
	}

	if initLog() != nil {
		return
	}
	// log.Print(os.Args)
	logx.Infof("main start listen port:%v, findnode addr:%v ", *port, *targetAddr)
	c := dht.NewClient(*port, *targetAddr, *ipv46)
	if c == nil {
		logx.Infof("NewClient fail")
	} else {
		c.Start()
	}
	stop := make(chan int, 1)
	<-stop
}
