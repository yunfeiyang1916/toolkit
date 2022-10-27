#!/bin/bash

nohup ./bin/${target} -config ./config/${config_file} >>nohup.out 2>&1 &
