#!/bin/bash

home=/usr/src/code
go mod tidy
cd ${home}/service/user/rpc && rm -rf nohup.out && sh run.sh
cd ${home}/service/user/api && rm -rf nohup.out && sh run.sh

#cd ${home}/service/product/rpc && rm -rf nohup.out && sh run.sh
cd ${home}/service/product/api && rm -rf nohup.out && sh run.sh

cd ${home}/service/order/rpc && rm -rf nohup.out && sh run.sh
cd ${home}/service/order/api && rm -rf nohup.out && sh run.sh

cd ${home}/service/pay/rpc && rm -rf nohup.out && sh run.sh
cd ${home}/service/pay/api && rm -rf nohup.out && sh run.sh
