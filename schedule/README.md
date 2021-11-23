# 概要说明
本模块用于机器人集群调度，按照集群中实例的数量，以及BOT Gateway AP要求的最小分区数量，计算当前每个服务实例需要消费的分区号，然后启动Websocket链接Gateway。本模块可搭配 cluster/impl/ 下的 configfile或者etcd等版本的集群管理器使用。

# 使用示例
参见example
