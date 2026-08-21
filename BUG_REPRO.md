# Bug Reproduction

模板诊断会泄露敏感变量，重复或类型不符的变量没有被拒绝，非法 provider/shutdown duration 也可能被吞掉。执行 template 和 config 目标测试可复现。
