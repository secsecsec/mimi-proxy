# Installation

```
go get github.com/mimicloud/reverse-proxy
go install github.com/mimicloud/reverse-proxy
# or
git clone git@github.com:mimicloud/reverse-proxy.git
go get -v .
```

# Etcd scheme

/apps/u1/frontends/f1 {"tls_cert": "", "tls_key": "", "hosts": ["*.example.com", "example.com"]}
/apps/u1/backends/b1 {"url": "192.168.0.1:5000", "connection_timeout": 1000}

# API


### Application

Application list
```
GET /v1/
```

Application detail
```
GET /v1/<appId>
```

Create application
```
POST /v1/ {"id": <appId>}
```

Delete application
```
DELETE /v1/<appId>
```

### Frontend

Frontend detail
```
GET /v1/<appId>/frontend/<appId>
```

Frontend create / update
```
POST /v1/<appId>/frontend/<frontendId> {"hosts": ["*.example.com", "example.com"], "tls_crt": "", "tls_key": ""}
```

Frontend delete
```
DELETE /v1/<appId>/frontend/<frontendId>
```

### Backend

Backend detail
```
GET /v1/<appId>/backend/<backendId>
```

Create backend (url without http or https)
```
POST /v1/<appId>/backend {"url": "192.168.0.5:5000", "connect_timeout": 1000}
```

Delete backend
```
DELETE /v1/<appid>/backend/<backendId>
```

# Benchmark

### dummy server

```
# wrk -t10 -c500 -d30s http://127.0.0.1:3000/
Running 30s test @ http://127.0.0.1:3000/
  10 threads and 500 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency    12.22ms    2.92ms  34.25ms   79.24%
    Req/Sec     4.22k   561.02     5.77k    73.21%
  1241843 requests in 30.00s, 153.96MB read
Requests/sec:  41401.35
Transfer/sec:      5.13MB
```

### nginx -> dummy server via http

```
# wrk -t10 -c500 -d30s http://127.0.0.1:81/
Running 30s test @ http://127.0.0.1:81/
  10 threads and 500 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency    33.17ms    6.98ms  75.29ms   74.23%
    Req/Sec     1.53k   161.80     1.87k    78.64%
  456868 requests in 30.00s, 73.63MB read
Requests/sec:  15226.43
Transfer/sec:      2.45MB
```

### reverse-proxy > dummy server via http

```
wrk -t10 -c500 -d30s http://mindy.dev/
Running 30s test @ http://mindy.dev/
  10 threads and 500 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency    16.52ms    3.66ms  43.74ms   75.28%
    Req/Sec     3.11k   406.54     4.69k    72.84%
  917085 requests in 30.00s, 113.70MB read
Requests/sec:  30569.53
Transfer/sec:      3.79MB
```

### dummy server

```
siege -c25 -t1M 127.0.0.1:3000
** SIEGE 3.0.5
** Preparing 25 concurrent users for battle.
The server is now under siege...
Lifting the server siege...      done.

Transactions:               3031 hits
Availability:             100.00 %
Elapsed time:              59.42 secs
Data transferred:           0.04 MB
Response time:              0.00 secs
Transaction rate:          51.01 trans/sec
Throughput:             0.00 MB/sec
Concurrency:                0.02
Successful transactions:        3031
Failed transactions:               0
Longest transaction:            0.01
Shortest transaction:           0.00
```

### nginx > dummy server via http

```
siege -c25 -t1M 127.0.0.1:81
** SIEGE 3.0.5
** Preparing 25 concurrent users for battle.
The server is now under siege...
Lifting the server siege...      done.

Transactions:               3033 hits
Availability:             100.00 %
Elapsed time:              59.17 secs
Data transferred:           0.04 MB
Response time:              0.00 secs
Transaction rate:          51.26 trans/sec
Throughput:             0.00 MB/sec
Concurrency:                0.03
Successful transactions:        3033
Failed transactions:               0
Longest transaction:            0.01
Shortest transaction:           0.00
```

### reverse-proxy > dummy server via http

```
### siege -c25 -t1M mindy.dev
** SIEGE 3.0.5
** Preparing 25 concurrent users for battle.
The server is now under siege...
Lifting the server siege...      done.

Transactions:               2910 hits
Availability:             100.00 %
Elapsed time:              59.33 secs
Data transferred:           0.04 MB
Response time:              0.00 secs
Transaction rate:          49.05 trans/sec
Throughput:             0.00 MB/sec
Concurrency:                0.09
Successful transactions:        2910
Failed transactions:               0
Longest transaction:            0.01
Shortest transaction:           0.00
```
