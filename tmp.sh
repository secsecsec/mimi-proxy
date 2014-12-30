#!/bin/bash

etcdctl rm /apps --recursive
etcdctl setdir /apps
etcdctl setdir /apps/u1
etcdctl setdir /apps/u1/backends
etcdctl setdir /apps/u1/frontends
etcdctl set /apps/u1/backends/b1 '{"url": "localhost:8000"}'
etcdctl set /apps/u1/frontends/f1 '{"hosts": ["localhost"]}'
