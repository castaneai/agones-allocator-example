# agones-allocator-example

```sh
$ go run main.go [agones namespace] [agones fleetname]
...
```

```sh
$ curl http://localhost:8888/allocate
{"name":"fleetname-xxx","addresses":{"tcp":"192.168.99.100:7619","udp":"192.168.99.100:7775"}}
```

