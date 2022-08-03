pid=`ps axu |grep './ciligo' | grep -v grep | awk '{print $2}'`
if [ "$pid" == ""  ];
then
    echo "pid: $pid, not to kill"
else
    kill $pid
fi
rm -rf log/*
./ciligo -p 8051 -t localhost:8050 >/dev/null &
./ciligo >/dev/null &