module github.com/tencent-connect/botgo-plugins/schedule/example

go 1.15

replace github.com/tencent-connect/botgo-plugins/schedule => ../
replace github.com/tencent-connect/botgo-plugins/base => ../../cluster/base
replace github.com/tencent-connect/botgo-plugins/cluster/impl/etcd => ../../cluster/impl/etcd

require (
	github.com/tencent-connect/botgo v0.0.0-20211122124126-a4936f507e42
	github.com/tencent-connect/botgo-plugins/cluster/base v0.0.0-20211124073815-757ae5fa4913
	github.com/tencent-connect/botgo-plugins/cluster/impl/etcd v0.0.0-20220111065311-5a79f5b09cfd
	github.com/tencent-connect/botgo-plugins/schedule v0.0.0-00010101000000-000000000000
)
