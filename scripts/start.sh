#!/bin/bash
pid=`ps axu |grep './ciligo' | grep -v grep | awk '{print $2}'`
if [ "$pid" == ""  ];
then
    echo "pid: $pid, not to kill"
else
    kill $pid
fi
rm -rf log/*

# 启动n个进程
# ipv6
# ./ciligo -p 8050 -t "6">./console.out 2>&1 &

./ciligo -p 8050 >./log/console8050.out 2>&1 &
./ciligo -p 8053 -a localhost:8050 >./log/console8051.out  2>&1 &