#!/bin/bash
server=product
app=${server}_rpc
printf "start build ${app}\n"
go build -o ${app}
if [[ $? != 0 ]]; then
  printf "build failed\n"
  exit 101
fi

printf "start run ${app}\n"
nohup ./${app} -f etc/${server}.yaml >>nohup.out 2>&1 &
