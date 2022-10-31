#!/bin/bash

home=/usr/src/code
go mod tidy
cd ${home}/service/user/rpc && sh run.sh
cd ${home}/service/user/api && sh run.sh

cd ${home}/service/product/rpc && sh run.sh
cd ${home}/service/product/api && sh run.sh

cd ${home}/service/order/rpc && sh run.sh
cd ${home}/service/order/api && sh run.sh

cd ${home}/service/pay/rpc && sh run.sh
cd ${home}/service/pay/api && sh run.sh
