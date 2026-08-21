# Bug Reproduction

API 的 `/console/` 静态页面路由缺失，worker shutdown helper 会过早返回且无法遵守截止时间。执行两个 cmd 目录的诊断测试可复现。
