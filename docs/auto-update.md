# 自动升级（Auto Update）

QuotaPanel 的自动升级由「更新后端 + GitHub Releases 回退 + 平台安装器」三部分组成。
前端只通过统一的 `Call` 入口访问 `update.*` 命名空间，所有状态通过 `update:state` 事件推送。

## 1. 调用接口

| 方法 | 参数 | 返回 | 说明 |
| --- | --- | --- | --- |
| `update.getState` | 无 | `State` | 最近一次状态快照 |
| `update.check` | `{ channel?: "latest" \| "stable" \| "beta" }` | `State` | 查询最新版本并选出租户平台的安装包 |
| `update.setChannel` | `{ channel: string }` | `State` | 切换通道并清空已选版本 |
| `update.download` | 无 | 无 | 后台下载，进度走事件 |
| `update.install` | 无 | `InstallResult` | 原位置换当前安装并重启 |

状态机：`idle → checking → available → downloading → ready → installing → restarting`，
任何一步失败进入 `failed` 并保留 `error` 与可重试信息。

### 事件

```ts
EventsOn('update:state', (state: UpdateState) => { ... })
```

`bindUpdateEvents()`（`frontend/packages/ui/src/utils/version.ts`）负责订阅并拉取初始状态；
`UpdateDownloadModal.vue` 负责下载进度、后台最小化与安装按钮。

## 2. 更新来源

1. **更新后端**：`appConfig.VersionCheckUrl`
   （`https://open.tecmz.com/open_app/QuotaPanel/app_manager/updater`）。
   请求体：`{ uuid, version, channel, platform: { name, arch, version } }`，
   响应：`{ code: 0, msg, data: { version, summary, notes, url, assets: [...] } }`。
   每个资产的 `urls` 由后端统一生成（`OpenReleaseUtil::buildUrls`：配置的 GitHub 代理前缀 +
   GitHub 直连），客户端**不内置任何代理前缀**，只对返回的 `urls` 做并发测速排序与失败回退。
2. **GitHub Releases 回退**：后端不可用时读取 `open-tecmz/quotapanel` 的已发布
   Release（跳过 draft；`stable` 跳过 prerelease），使用 GitHub 官方直连地址，并从
   `checksums.txt` 读取 SHA-256。

安装包命名约定（`QuotaPanel-<version>-<os>-<arch>.<ext>`）：

| 平台 | 自动安装格式 | 说明 |
| --- | --- | --- |
| macOS | `.zip` | 解压出 `.app` 后由后台脚本置换并重新打开，要求已公证 / 已装订 |
| Windows | `.exe` | 后台批处理等待退出后重命名替换并重新启动 |
| Linux | `.AppImage` | 直接替换当前 AppImage 并自动重启，替换前检查 WebKitGTK 运行时依赖 |
| 其他 | `.dmg` / `.deb` | 不自动安装，引导到下载页 |

自动安装的前提是资产带 `sha256`（后端 `assets[].sha256`，或 GitHub 的
`checksums.txt` / `digest`）。缺失校验值时只提供手动下载，绝不安装未校验的文件。

## 3. 环境变量

| 变量 | 作用 |
| --- | --- |
| `QUOTAPANEL_UPDATE_ENDPOINT` | 覆盖更新后端地址（本地联调 / 端到端测试） |
| `QUOTAPANEL_DISABLE_UPDATE_CHECK=1` | 关闭启动后的自动检测，设置页手动检测仍可用 |

## 4. 发布产物（GitHub Actions）

工作流位于 Open 仓库 `.github/workflows`，构建时从私有仓库拉取 Pro 源码：

- `release.yml`（tag `v*.*.*`）：Linux `.deb` + `.AppImage`、Windows `.exe`、
  macOS `.dmg` + 已公证的 `.zip`；发布前生成包含全部产物的 `checksums.txt`，
  创建**已发布**的 Release（带 `-` 的 tag 自动标记 prerelease）。
- `main.yml`：每次推送到 main 产出同样的安装包，并重建 `vsnapshot` 快照 Release。
- `test.yml`：仅做多平台编译验证。

macOS 的 zip 必须先公证、`stapler staple` 到 `.app` 后再重新打包 zip，
否则自动替换后的应用会被 Gatekeeper 拒绝。

## 5. 测试

```bash
# 后端单元测试 + 完整升级流程（本地 HTTP 更新服务器 + 真实下载 / 安装）
go test ./backend/base/updater/ ./ -v

# 前端类型检查与构建
cd frontend && pnpm --filter @quotapanel/ui build

# 真实应用端到端（Linux，需要 Xvfb 与 autotest 构建）
go build -tags "desktop,production,autotest,webkit2_41" -o build/bin/QuotaPanel .
QUOTAPANEL_DISABLE_UPDATE_CHECK=1 \
QUOTAPANEL_UPDATE_ENDPOINT=http://127.0.0.1:53999/app_manager/updater \
xvfb-run -a ./build/bin/QuotaPanel --auto-test-port=53080
# 然后通过 POST http://127.0.0.1:53080/auto 调用
#   { type: "call", data: { name: "update.check", params: { channel: "latest" } } }
#   { type: "call", data: { name: "update.download" } }
#   { type: "call", data: { name: "update.getState" } }
```
