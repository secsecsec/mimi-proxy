#!/bin/bash

# etcdctl set /apps/u1/frontends/f1  '{"hosts": ["garant43.dev"]}'
# etcdctl set /apps/u1/frontends/f1  '{"hosts": ["localhost"]}'

etcdctl set /apps/u1/backends/b1 '{"url": "localhost:3000"}'
# etcdctl set /apps/u1/backends/b1 '{"url": "localhost:8000"}'
