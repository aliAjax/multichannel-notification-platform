# Bug Reproduction

不存在通知的查询、更新和回执路径虽然打印 `not found`，但错误链无法被上层识别。触发目标测试即可看到 `errors.Is` 断言失败。
