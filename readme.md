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
