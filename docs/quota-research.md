# AI 额度来源调研

调研日期：2026-09-24。参考项目：[OpenQuota](https://github.com/deviffyy/OpenQuota)、[Usage4Claude](https://github.com/f-is-h/Usage4Claude)、[TokenBar](https://github.com/Nanako0129/TokenBar)、[VibeGauge](https://github.com/MaxHaiCom/VibeGauge)。各项目的供应商列表、接口和响应映射用于核对实现；本项目适配器独立编写。

| 供应商 | 查询方式 | 当前状态 |
| --- | --- | --- |
| Claude 订阅 | 独立浏览器登录，站内用量接口 | 已接入 |
| ChatGPT / Codex | 独立浏览器登录，站内用量接口 | 已接入 |
| Cursor | 独立浏览器登录，站内用量接口 | 已接入 |
| Xiaomi MiMo Token Plan | 独立浏览器登录，按规则解析控制台页面 DOM（tp- API Key 仅用于模型调用，无法查询额度） | 已接入 |
| Z.ai Coding Plan | API Key | 已接入 |
| Kimi Coding Plan | Coding Plan Key | 已接入 |
| MiniMax Token Plan | API Key | 已接入 |
| GitHub Copilot | 可访问 Copilot 内部接口的 GitHub 登录令牌 | 已接入 |
| Devin / Windsurf | Devin 扩展 API Key | 已接入 |
| Grok CLI | Grok CLI OAuth access token | 已接入 |
| OpenCode | API Key | 已接入 |
| Command Goat | API Key | 已接入 |
| DeepSeek API | API Key | 已接入 |
| Kimi 开放平台 API | API Key | 已接入 |
| OpenRouter | API Key | 已接入 |
| OpenAI API | API Key | 已接入 |

浏览器账号通过 Chrome/Edge 独立应用窗口登录。每个账号的浏览器资料目录单独保存，登录后的 Cookie 留在浏览器资料内；应用每五分钟在相应会话中读取额度。Wails v2 当前程序为单窗口，因此这里使用独立 Chrome/Edge 窗口，并非嵌入式 WebView。删除账号时会删除其浏览器资料目录。

站内接口及第三方内部接口没有稳定性承诺，网站改版或登录策略变化可能使查询失效。实际账号的登录与远端响应需要用户在应用中验证；自动测试覆盖了样例响应映射、浏览器 Cookie 持久化及应用构建。Antigravity 等依赖本机其他应用运行状态和专有认证流程，尚未接入；不能把 GitHub 上所有项目的供应商无差别移植为可用适配器。
