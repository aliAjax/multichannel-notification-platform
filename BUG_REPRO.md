# Bug Reproduction

HTTP provider 未配置 client 时 Health 或 Send 会触发 nil pointer，registry 查询缺失配置也不会返回明确错误。执行 provider 目标测试可复现。
