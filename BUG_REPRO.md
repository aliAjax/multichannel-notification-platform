# Bug Reproduction

暂停通知恢复时状态转换失败，重放记录保留旧 provider 和错误信息。触发方式是分别执行状态转换与 replay 目标测试；埋错版本会出现 `invalid notification transition` 或旧投递字段未清理。
