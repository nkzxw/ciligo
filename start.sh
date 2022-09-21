
# 启动两个进程
pid=`ps axu |grep './ciligo' | grep -v grep | awk '{print $2}'`
if [ "$pid" == ""  ];
then
    echo "pid: $pid, not to kill"
else
    kill $pid
fi
rm -rf log/*
# ./ciligo -p 8051 -a localhost:8050 >./console.out 2>&1 &
# ./ciligo -p 8050 -a ll >./console.out 2>&1 &
./ciligo -p 8050 -a "" >./console.out 2>&1 &