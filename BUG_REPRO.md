# Bug Reproduction

取消 context 后 dispatcher 仍可能安排重试，histogram 快照还会被后续采样修改。执行 worker 与 observability 目标测试可复现。
