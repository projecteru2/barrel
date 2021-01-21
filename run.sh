#!/bin/bash

if [ "$EUID" -ne 0 ]
  then echo "Please run as root"
  exit
fi

ETCD_ENDPOINTS=http://127.0.0.1:2379 ./eru-barrel --log-level DEBUG