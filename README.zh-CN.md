# aprs-android-hook

[English](README.md) | **简体中文**

[aprs-android](https://github.com/Nigh/aprs-android) 的自部署 webhook 服务端：接收手机定位，在浏览器中按日期查看路径、距离与记录时长，并在 OSM 地图上回看行程。

每个部署只有一个账户，可登记最多16个手机上传 token。Go + SQLite + Svelte，前后端由同一个服务提供，无需单独部署数据库。界面遵循 [xianii-theme](https://github.com/Nigh/xianii-theme)，地图沿用 Android 的 OpenFreeMap 深色样式。

网页支持 English 和简体中文，默认英文。使用页面上方的语言选择器切换，选择保存在当前浏览器中，不影响部署时区或已有数据。

## 用 Docker Compose 部署

需要 Git、Docker 和 Docker Compose v2。初次部署会从源码构建镜像。

```sh
git clone https://github.com/Nigh/aprs-android-hook.git
cd aprs-android-hook
cp .env.example .env
docker compose up -d --build
```

默认在宿主机 `0.0.0.0:18080` 提供 HTTP 服务，可直接从公网访问。在服务器防火墙中放行 TCP 18080 端口，浏览器打开 `http://服务器公网IP:18080` 注册。**首个成功注册者即为唯一用户，此后拒绝所有注册请求。**密码至少12个字符，最多72字节。轨迹和上传设置仍需登录后访问。

如需 HTTPS，按下文配置 Caddy 后直接在 HTTPS 地址注册。直接 HTTP 访问不会加密密码或定位数据。不要执行 `docker compose down -v`，它会删除包含账户和轨迹的数据卷，并重新开放注册。

### 修改端口和时区

编辑 `.env`，然后执行 `docker compose up -d`：

```dotenv
PORT=18081
BIND_ADDRESS=0.0.0.0
TZ=Asia/Shanghai
PUBLIC_URL=
```

`PORT` 是宿主机端口，容器内始终为8080。`TZ` 控制日期筛选和界面时间；可改成其他 IANA 时区。`BIND_ADDRESS` 默认是 `0.0.0.0`，允许公网访问。已有部署若在 `.env` 中设置了 `BIND_ADDRESS=127.0.0.1`，需同步修改才能直接开放公网访问。

## 用 Caddy 提供 HTTPS

Android 只接受 HTTPS webhook 地址。先将域名 DNS 指向服务器，确保 Caddy 可以使用公网80和443端口；Caddy 运行在宿主机时，在 `.env` 设置：

```dotenv
PUBLIC_URL=https://tracks.example.com
BIND_ADDRESS=127.0.0.1
```

`PUBLIC_URL` 必须是浏览器访问的完整源地址，不能包含路径。它用于同源校验及 Secure 登录 Cookie；不要仅修改反代地址而忘记这个配置。

宿主机 Caddy 的 Caddyfile：

```caddyfile
tracks.example.com {
    reverse_proxy 127.0.0.1:18080
}
```

如果修改了 `PORT`，同步修改反代端口。应用与 Caddy 配置生效：

```sh
docker compose up -d
sudo caddy validate --config /etc/caddy/Caddyfile
sudo systemctl reload caddy
```

随后打开 `https://tracks.example.com` 注册或登录；已有账户时只允许登录。整个站点反代到应用，包括 `/api/webhook`，不要增加尾斜杠重定向。应用不信任任意客户端发送的代理头，以 `PUBLIC_URL` 判断外部地址。HTTP 到 HTTPS 的跳转由 Caddy 负责。[Caddy 反代文档](https://caddyserver.com/docs/caddyfile/directives/reverse_proxy)

### Caddy 也运行在容器内

Caddy 容器中的 `127.0.0.1` 指向 Caddy 自身。让现有 Caddy 容器加入应用的 Compose 网络，通过服务名连接即可，不需要把 Caddy 加入本项目。

默认网络名为 `aprs-android-hook_default`（更改 Compose 项目名时也会改变）：

```sh
docker network connect aprs-android-hook_default your-caddy-container
```

此时 Caddyfile 中使用 `reverse_proxy app:8080`。为避免 Caddy 重建后丢失网络连接，在它自己的 Compose 配置中声明这个外部网络，并加入 Caddy 服务的 `networks`：

```yaml
services:
  caddy:
    # 保留已有 image、ports、volumes 等配置
    networks:
      - default
      - aprs
networks:
  aprs:
    external: true
    name: aprs-android-hook_default
```

仅通过容器内 Caddy 提供公网访问时，可保留 `BIND_ADDRESS=127.0.0.1`，容器间通过 `app:8080` 连接。

## 连接 aprs-android

1. 在手机 aprs-android 打开 **Settings → Webhook**，选择 **Copy token**。token 由手机生成，更新应用时保持不变，直到主动刷新。
2. 登录网页，进入 **上传设置**，粘贴手机 token，填写便于辨认的备注并登记。点击页面显示的 Webhook HTTPS 地址即可复制完整地址，粘贴到手机应用后点击 Test。
3. 在手机启用 **Enable webhook**，填写：
   - **Webhook HTTPS URL**：`https://tracks.example.com/api/webhook`
   - **Webhook reporting ID**：一个用于区分记录的名称，例如 `my-phone`；它不是网页登录用户名。
4. 手机获取并上报可靠定位后，在操作日志中检查 **Webhook HTTP 200**。
5. 打开网页 **路径记录**，选择日期，点击 **查看路径**。可按上报 ID 和设备筛选，点击路径查看地图及定位点详情。

手机的 webhook 在 APRS 被停止区域等规则阻止时仍可能上传定位。服务端只记录实际收到的数据；它不能恢复网络失败时未上传的点。

刷新手机 token 后，旧 token 不会自动更新：在网页删除旧记录并登记新 token。删除 token 立即禁止后续上传，保留已有轨迹。服务端只存 token 哈希，不能取回完整值。

### 用 curl 测试 webhook

将 `TOKEN` 替换为手机复制且已登记的值。避免将真实 token 提交到仓库或贴入公开日志。

```sh
TOKEN='替换为手机上的64位token'
SESSION_ID=$(python3 -c 'import uuid; print(uuid.uuid4())')
curl --fail-with-body https://tracks.example.com/api/webhook \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  --data "{\"type\":\"test\",\"session_id\":\"$SESSION_ID\"}"
```

测试成功返回 HTTP 200 和 `{"ok":true,"type":"test"}`，请同时检查状态码和 JSON 响应。这验证连接、token 认证及生成的会话 ID 格式，不保存定位，也不修改 token 最近上传时间。服务端只校验 UUID 格式，无法验证客户端随机数的质量；仍需真实定位上传来验证数据保存。`(0,0)` 是合法坐标，不作为测试标识。

### 包类型与记录会话

`POST /api/webhook` 的**每个请求**都必须明确提供 `type` 和合法的 `session_id`，包括测试包：

| `type` | 行为 |
| --- | --- |
| `"location"` | 将定位保存到指定记录会话。 |
| `"test"` | 校验会话 ID 和 Bearer token 后返回测试响应，不要求或保存定位字段。 |

缺失、空字符串、null 或未知类型返回 HTTP 400，不再支持 `start`、`end`。定位包必须提供 `id`、`device_hash`、`session_id`、`timestamp_ms`、`latitude`、`longitude`；可选 `accuracy_m`、`speed_mps`、`bearing_deg`、`altitude_m` 可以为空。例如：

```json
{"type":"location","id":"my-phone","device_hash":"0123456789abcdef","session_id":"e0629f7c-57a5-4d47-9afc-00c3c6d4d3f9","timestamp_ms":1767225600000,"latitude":31.2304,"longitude":121.4737,"accuracy_m":5,"speed_mps":null,"bearing_deg":null,"altitude_m":null}
```

`session_id` 必须是**标准小写 UUID v4**：包含连字符的36个 ASCII 字符，版本位为 `4`，变体位为 `8`、`9`、`a` 或 `b`。接受的格式：

```text
^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$
```

缺失、null、大写、带多余空白或格式不符返回 HTTP 400 和 `error.sessionID`；其他 JSON 类型（数字、布尔值、数组或对象）返回 HTTP 400 和 `error.json`。服务端不去除空白或转换大小写。

Android 推荐在开始记录时调用 `java.util.UUID.randomUUID().toString()` 生成一次 ID，**在获取 GPS 或尝试上传之前生成**。同一会话的所有点及重试使用相同 ID，网络/GPS 失败或切换上报渠道都不更换。每次新记录生成新 ID，不能重复使用以前的 ID。如果进程重启后恢复原记录，需要持久化并恢复其 ID；如果开始新记录，则生成新 ID。独立的手动单次上报使用新 ID；属于正在进行的记录的手动上报复用该记录 ID。每次 `test` 请求用同一生成方法产生新的随机 ID，不替换正在记录的会话 ID。上面的 UUID 仅作示例，Android 不能将它硬编码为固定值。

定位上传成功返回 `{"ok":true}`。按 `(id, device_hash, session_id, timestamp_ms)` 去重，重试必须保留这些字段；不同会话的点可以使用相同时间戳。会话 ID 用于区分记录，不用于认证：所有请求仍需要已登记的 Bearer token，更换 token 本身不产生新会话。

Android 必须实现此协议才能成功上传，仅更新服务端不会让手机自动生成会话 ID。本服务端面向全新数据库，不兼容以前的请求格式或数据库结构。

## 路径与统计怎么算

- 相同 `(id, device_hash, session_id)` 的所有已收到定位点组成一条路径，按定位时间排序。不同会话即使时间重叠或时间戳相同也不合并；同一会话中间断档多久都不拆分，不再使用30分钟保底或起止标记状态。
- 距离为同一会话相邻点之间的球面距离总和，不是道路导航距离。GPS 漂移可能增加距离，本期不自动过滤漂移。缺失点无法恢复，断档两端的直线连接也可能与真实路线不同。
- 记录时长为每个会话最后收到的定位时间减去最先收到的定位时间，再求和；包含会话内的停留和断档，不包含会话之间的间隔，也不是实际运动时间或用户点击开始/停止之间的精确时长。
- 单点路径的距离和时长均为零。不同设备或上报 ID 即使复用同一会话 ID 也不连接；同一上报 ID、设备下复用会话 ID 会合并不同记录。
- 日期段包含结束日期全天，按页面显示的部署时区解释。统计和地图只使用选定日期范围内的点；跨日会话在筛选范围边界裁剪。
- 列表每页50条，汇总统计覆盖整个筛选结果。路径列表和详情均返回 `session_id`；路径 key 由上报 ID、设备和会话确定，不因日期裁剪或补传旧点而变化，但定位点和统计可以变化。

地图由 [OpenFreeMap](https://openfreemap.org/) 提供，使用 OpenStreetMap 数据。浏览器需要能访问地图服务并支持 WebGL；地图加载失败不会阻止查看路径列表和统计。PWA 只缓存界面静态资源，离线时不能查询私有轨迹。

## 数据备份、恢复和升级

SQLite 数据保存在 Compose 的 `data` 卷中，包含唯一账户、登录会话、token 哈希和全部定位点。备份应保存在可信位置。

停止应用后打包数据卷，保证数据库及 WAL 一致：

```sh
docker compose stop app
docker compose run --rm --no-deps --user 0 --entrypoint tar app -C /data -czf - . > aprs-backup.tar.gz
docker compose start app
```

恢复到同一部署或已经创建数据卷的新部署（先停止应用，恢复会覆盖同名文件）：

```sh
docker compose stop app
docker compose run --rm --no-deps --user 0 --entrypoint sh app -c 'rm -f /data/aprs.sqlite /data/aprs.sqlite-wal /data/aprs.sqlite-shm'
docker compose run --rm --no-deps -T --user 0 --entrypoint tar app -C /data -xzf - < aprs-backup.tar.gz
docker compose run --rm --no-deps --user 0 --entrypoint chown app -R 65532:65532 /data
docker compose start app
```

删除数据库前先确认备份可用。以上二进制重定向示例面向 Linux/macOS shell；Windows 使用 WSL，避免旧版 PowerShell 改写压缩文件字节。

升级前备份，再执行：

```sh
git pull --ff-only
docker compose up -d --build
docker compose ps
```

本期没有密码找回功能，请妥善保存密码和数据库备份。

## 常见问题

| 现象 | 检查 |
| --- | --- |
| 手机 HTTP 401 | token 是否已登记、是否刷新过；复制时是否完整 |
| 手机 HTTP 400 | 定位字段、时间及会话 ID 是否有效；测试 JSON 是否完整 |
| 页面 HTTP 403 | `.env` 的 `PUBLIC_URL` 是否与浏览器地址一致，包括协议和端口 |
| 登录后仍显示登录页 | 是否在 HTTP 地址使用了配置为 HTTPS 的服务；应通过 Caddy 的 HTTPS 地址访问 |
| 没有路径 | 日期、部署时区、ID/设备筛选，以及手机是否收到 HTTP 200 |
| token 显示“尚未上传” | 是否在手机启用 webhook、填写正确地址并得到可靠定位 |
| 地图不显示 | 到 OpenFreeMap 的网络连接、WebGL 和浏览器控制台 |
| 端口占用 | 修改 `.env` 的 `PORT`，并同步修改 Caddy 上游 |

查看服务状态和日志：`docker compose ps`、`docker compose logs --tail=100 app`。日志不记录上传 token 或定位请求正文。

## 本地开发

需要 Go 1.26.4+、Node.js 22.19+ 和 Corepack。

```sh
cd web
corepack enable
yarn install --immutable
yarn check
yarn lint
yarn test
yarn build
cd ..
go test ./...
go run .
```

打开 `http://localhost:8080`。前端热更新：在另一个终端进入 `web` 执行 `yarn dev`，同时将 Go 服务的 `PUBLIC_URL` 配置为 `http://localhost:5173`；Vite 将 `/api` 代理到8080。

支持环境变量：`LISTEN_ADDR`（默认 `:8080`）、`DB_PATH`（默认 `data/aprs.sqlite`）、`WEB_DIR`（默认 `web/build`）、`TZ`、`PUBLIC_URL`。Docker 使用固定容器端口，通过 Compose 修改宿主机端口。

### 扩展界面语言

在 `web/src/lib/locales` 中增加 JSON 字典，沿用 `en.json` 的文案键和命名占位符。在 `index.js` 导入字典，并在 `locales` 注册表中添加 `{ name, messages }` 项，以 BCP 47 语言标签作为键。选择器自动读取注册表，不使用中英二选一的布尔开关。

文案支持字符串或 `Intl.PluralRules` 的复数形式对象，例如 `one`、`other`。缺失文案回退到英文。日期和数字使用 `Intl` 格式化，部署时区独立于界面语言。测试检查文案键、占位符、回退、复数，以及第三种语言的注册。

API 错误保留原有 `error` 信息，并新增稳定的 `code` 翻译键。界面按 code 翻译，未知 code 使用本地化的通用错误提示。安装后的 PWA 名称和描述使用英文，界面按浏览器保存的语言显示。

## 许可证与来源

MIT。前端基于 [svelte-static-pwa-template](https://github.com/Nigh/svelte-static-pwa-template)，保留其 [MIT 许可证](web/LICENSE)。设计 tokens 使用 [xianii-theme](https://github.com/Nigh/xianii-theme)；地图保留 OpenFreeMap / OpenMapTiles / OpenStreetMap 署名。
