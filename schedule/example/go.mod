module github.com/tencent-connect/botgo-plugins/schedule/example

go 1.15

replace github.com/tencent-connect/botgo-plugins/schedule => ../

require (
	github.com/tencent-connect/botgo v0.0.0-20211122124126-a4936f507e42
	github.com/tencent-connect/botgo-plugins/cluster/base v0.0.0-20211124073815-757ae5fa4913
	github.com/tencent-connect/botgo-plugins/cluster/impl/etcd v0.0.0-20211124080534-0fab75bca648
	github.com/tencent-connect/botgo-plugins/schedule v0.0.0-00010101000000-000000000000
)
