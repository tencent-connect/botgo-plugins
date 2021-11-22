module github.com/tencent-connect/botgo-plugins/schedule/example

go 1.15

replace github.com/tencent-connect/botgo-plugins/schedule => ../

require (
	github.com/tencent-connect/botgo v0.0.0-20211122124126-a4936f507e42
	github.com/tencent-connect/botgo-plugins/cluster/base v0.0.0-20211122141011-6d922fabf381
	github.com/tencent-connect/botgo-plugins/cluster/impl/etcd v0.0.0-20211122142543-8a0b12d77536
	github.com/tencent-connect/botgo-plugins/schedule v0.0.0-00010101000000-000000000000
)
