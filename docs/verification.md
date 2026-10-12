# 验证记录

2026-10-13，使用模拟数据完成，未接入真实手机或公网域名。

- Go 测试及 race 检查通过：并发首次注册、认证与同源校验、会话过期和限流、并发 token 上限、删除后禁止上传、Android JSON 格式、重复与乱序点、数据库重开、30分钟分段边界、球面距离、跨日期裁剪、列表分页和详情一致性、HTTPS 上传。
- 前端：`yarn check` 零错误、零警告；`yarn lint`、`yarn test`、`yarn build` 通过。
- 浏览器：1440px、992px、390px 界面检查；首次注册、登录、token 登记、键盘方向键切换、删除确认的取消操作、路径列表和地图。12个模拟点形成3段，总记录时长18分钟。地图起终点、署名和容器尺寸变化正常，浏览器无错误日志。
- Docker Compose：从源码构建，默认18080和更改为18081均健康启动，非 root 用户可写数据卷。
- Caddy：独立容器接入应用网络，使用本地 CA 验证 HTTPS 登记账户、Secure Cookie、登记 token、上传和查询；容器重启后会话保留。证书链和主机名校验启用，未使用 `curl -k`。
- 备份恢复：停止服务打包完整数据卷，恢复至独立测试卷；账户、会话和定位点可读取。

复现自动检查：在根目录执行 `go test .` 和 `go vet .`；在 `web` 执行 `yarn install --immutable`、`yarn check`、`yarn lint`、`yarn test`、`yarn build`。CI 另外执行 Go race 检查。

尚未验证真实 Android 到公网服务器的端到端连接及公网 ACME 证书签发。上线后按 README 的手机配对流程检查 `Webhook HTTP 200`。
