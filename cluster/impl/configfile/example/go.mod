module github.com/tencent-connect/botgo-plugins/cluster/impl/configfile/example

go 1.15

replace github.com/tencent-connect/botgo-plugins/cluster/base => ../../../base

replace github.com/tencent-connect/botgo-plugins/cluster/impl/configfile => ../

require (
	github.com/tencent-connect/botgo-plugins/cluster/base v0.0.0-20211124073815-757ae5fa4913
	github.com/tencent-connect/botgo-plugins/cluster/impl/configfile v0.0.0-00010101000000-000000000000
)
