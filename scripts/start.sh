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
# ./ciligo -p 8051 -a localhost:8050 >./console.out 2>&1 &
# ./ciligo -p 8050 -a test >./console.out 2>&1 &
# ./ciligo -p 8050 -t "6">./console.out 2>&1 &
./ciligo -p 8050 >./log/console.out 2>&1 &