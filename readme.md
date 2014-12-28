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
