# Bug Reproduction

半开 provider 探活持锁重入会卡住路由，策略更新还会受到外部切片修改影响。并发 routeprobe 测试可复现死锁、快照污染和取消传播问题。
