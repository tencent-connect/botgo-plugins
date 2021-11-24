module github.com/tencent-connect/botgo-plugins/schedule/example

go 1.15

replace github.com/tencent-connect/botgo-plugins/schedule => ../

require (
	github.com/tencent-connect/botgo v0.0.0-20211122124126-a4936f507e42
	github.com/tencent-connect/botgo-plugins/cluster/base v0.0.0-20211124034518-37adad080eb7
	github.com/tencent-connect/botgo-plugins/cluster/impl/etcd v0.0.0-20211124035403-b2b577538a18
	github.com/tencent-connect/botgo-plugins/schedule v0.0.0-00010101000000-000000000000
)
