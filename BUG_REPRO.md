# Bug Reproduction

webhook 请求的 future 时间、空 secret、坏签名或 provider 路径不匹配时仍可能改变通知状态。security 与 HTTP handler 目标测试可复现鉴权缺口。
