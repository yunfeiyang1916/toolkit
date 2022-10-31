#!/bin/bash
server=pay
nohup go run ${server}.go -f etc/${server}.yaml >>nohup.out 2>&1 &
