#!/bin/bash

etcdctl rm /apps --recursive
etcdctl setdir /apps
etcdctl setdir /apps/u1
etcdctl setdir /apps/u1/backends
etcdctl setdir /apps/u1/frontends
etcdctl set /apps/u1/backends/b1 '{"url": "localhost:8000"}'
etcdctl set /apps/u1/frontends/f1 '{"hosts": ["localhost"]}'
etcdctl set /apps/u2/backends/b2 '{"url": "localhost:3000"}'
etcdctl set /apps/u2/frontends/f2  '{"hosts": ["garant43.dev"], "tls_crt": "", "tls_key": ""}'
