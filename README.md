# 概要说明
本仓库主要提供botgo相关的机器人开发组件。

# 目录说明
```
|-- cluster         // 集群管理模块
|   |-- base        // 集群管理模块接口定义，开发者可以基于etcd、zookeeper等方案实现该模块下定义的Cluster相关接口，实现这些接口既可以与schedule模块配合使用
|   `-- impl        // 该目录下存放各种实现方案的cluster
|       ├── configfile  // 基于yaml配置文件的集群管理器实现
|       |-- etcd        // Etcd版本集群管理器实现
`-- schedule        // 调度器模块，该模块基于cluster/base提供的接口，实现机器人集群的sharding计算管理功能，可搭配cluster/impl下的实现来使用
```

# 加入官方社区

欢迎扫码加入 **QQ 频道开发者社区**。

<img alt="QQ频道开发者社区" src="https://mpqq.gtimg.cn/privacy-info/qq-guild.png" width="400">
