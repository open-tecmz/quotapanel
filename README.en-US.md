# 🎯 QuotaPanel - AI Subscription Quota Panel

<div align="center">
  <img src="./demo/image/account.png" alt="QuotaPanel main screen" width="800"/>
</div>

> [🇨🇳 中文](README.md) | [🇺🇸 English](README.en-US.md)

<div align="center">
  <img src="https://img.shields.io/badge/Framework-Go%20%2B%20Wails%20%2B%20Vue3-blue" alt="Framework"/>
  <a href="https://open.tecmz.com/quotapanel"><img src="https://img.shields.io/badge/Website-open.tecmz.com%2Fquotapanel-blue" alt="Website"/></a>
  <a href="https://github.com/open-tecmz/quotapanel"><img src="https://img.shields.io/github/stars/open-tecmz/quotapanel.svg" alt="GitHub Stars"/></a>
  <a href="https://gitee.com/open-tecmz/quotapanel"><img src="https://gitee.com/open-tecmz/quotapanel/badge/star.svg" alt="Gitee Stars"/></a>
  <img src="https://img.shields.io/badge/License-Apache%202.0-green" alt="License"/>
</div>

## ✨ Overview

**QuotaPanel** is an open-source desktop panel for AI quotas. It gathers subscriptions and API balances that are otherwise scattered across provider dashboards — Claude, ChatGPT / Codex, Cursor, GitHub Copilot, Z.ai, Kimi, MiniMax, OpenRouter and more — into a single view, so one refresh shows every account's quota windows, remaining usage and reset times.

> 🎯 **Goal**: no more surprises when your subscriptions pile up — see all your AI quotas on one panel.

- 🔐 **Local first**: account keys and browser sessions stay on your machine, nothing is uploaded
- 🔄 **Auto refresh**: polls every 5 minutes, with manual refresh for a single account or all of them
- 🌐 **Browser login**: subscriptions that require a web login use a dedicated browser window with a persisted session
- 📊 **Quota visualization**: progress bars, balance details, and billing/reset times at a glance
- 🎨 **Light & dark themes**: one-click switch
- 🖥️ **Cross-platform**: **macOS / Windows / Linux**

## 📋 Features

| Feature | Description |
|---------|-------------|
| **Multi-source aggregation** | 15 AI subscription and API quota sources, shown as unified cards |
| **Quota visualization** | Quota windows, balances, usage stats, reset and billing times in one place |
| **Detail modal** | Click a card to see all quota windows, balances and usage details |
| **Status hints** | Normal / Nearly exhausted / Exceeded status colors with threshold warnings |
| **Key management** | Sensitive values are masked by default and can be copied |
| **Card screenshot** | Save a single subscription card as a PNG |
| **Browser login** | Sign in through a dedicated Chrome / Edge window; the session is kept and cleaned up with the account |
| **Scheduled refresh** | Automatic query every 5 minutes, plus manual refresh |
| **Multi-language** | Simplified Chinese / English |
| **Launch at login** | Optional auto start with the system |
| **Tray resident** | macOS menu bar / Windows, Linux system tray with show, restart and quit |
| **Close behavior** | Choose "ask every time / quit / hide to background" when closing the window |

## 🔌 Supported Quota Sources

### Browser Login

| Source | Description |
|--------|-------------|
| **Claude** | Read 5-hour and weekly quotas after signing in to Claude in the dedicated window |
| **ChatGPT / Codex** | Read the Codex subscription usage window after signing in to ChatGPT |
| **Cursor** | Read the current billing cycle usage after signing in to Cursor |

### API Key

| Source | Description |
|--------|-------------|
| **GitHub Copilot** | Premium, chat and completion quotas (requires a GitHub token with access to Copilot's usage endpoint) |
| **Z.ai Coding Plan** | GLM Coding Plan rolling and weekly usage |
| **Kimi Coding Plan** | Kimi Coding Plan subscription usage |
| **MiniMax Token Plan** | Rolling and weekly remaining quota |
| **Devin / Windsurf** | Daily and weekly quotas plus overage balance |
| **Grok CLI** | Grok CLI subscription period quota |
| **OpenCode** | Official Go plan rolling (5-hour) / weekly / monthly usage |
| **Command Goat** | Command Code (CommandGoat) account, plan and window usage |
| **DeepSeek API** | DeepSeek open platform balance |
| **Kimi API** | Kimi open platform balance |
| **OpenRouter** | Consumption and per-key limits for the current API key |
| **OpenAI API** | Organization Admin Key query for the last 30 days of API costs |

> Provider dashboards and internal endpoints offer no stability guarantee; site redesigns or login policy changes may break a query. The integration approach and response mappings are documented in [`docs/quota-research.md`](docs/quota-research.md).

## 🖼️ Screenshots

<div align="center">
  <img src="./demo/image/account.png" alt="Quota home" width="800"/>
  <p>Quota home</p>
</div>

<table width="100%">
  <thead>
    <tr>
      <th width="50%">Add subscription</th>
      <th width="50%">Subscription card</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><img src="./demo/image/account-add.png" alt="Add subscription" style="width:100%; border-radius: 8px;"/></td>
      <td><img src="./demo/image/account-card.png" alt="Subscription card" style="width:100%; border-radius: 8px;"/></td>
    </tr>
  </tbody>
  <thead>
    <tr>
      <th>Quota home (dark)</th>
      <th>Settings</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><img src="./demo/image/account-dark.png" alt="Quota home (dark)" style="width:100%; border-radius: 8px;"/></td>
      <td><img src="./demo/image/setting.png" alt="Settings" style="width:100%; border-radius: 8px;"/></td>
    </tr>
  </tbody>
</table>

## 🚀 Getting Started

### 📦 Install

Download the installer for your platform from [open.tecmz.com/quotapanel](https://open.tecmz.com/quotapanel) and run it.

Supported platforms: **macOS / Windows / Linux**.

### 🛠️ Local Development

#### Requirements

- [Go](https://go.dev/dl/) 1.24+
- [Wails v2](https://wails.io/docs/gettingstarted/installation)
- [Node.js](https://nodejs.org/) 20 + [pnpm](https://pnpm.io/)

```bash
# Install the Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Install pnpm
npm install -g pnpm
```

> Building on Linux also requires the Wails system dependencies (e.g. `libgtk-3-dev`, `libwebkit2gtk-4.0-dev`); see the Wails documentation.

#### Commands

```bash
# Install Go and frontend dependencies
make install

# Start development mode (frontend hot reload)
make dev

# Build the production app (output in build/bin/)
make build

# Build and install to /Applications (macOS)
make build-install
```

After changing a Go method exposed to the frontend, run `wails generate module` to regenerate the JS bindings.

## 📁 Project Structure

```
quotapanel/
├── main.go              # Wails app config and entry point
├── app*.go              # Methods exposed to the frontend
├── backend/
│   ├── base/            # Base library (database, config, logging, window, platform specifics)
│   └── quota/           # Quota provider adapters and query service
├── frontend/
│   └── packages/
│       ├── ui/          # Main UI (Vue3 + AntDesignVue + TailwindCSS)
│       └── quota/       # Quota business module
├── demo/image/          # UI screenshots
├── docs/                # Docs (quota source research, etc.)
└── Makefile             # Dev, build and release commands
```

## 🗂️ Data Directory

On first launch a data directory is created in your home folder (default `~/.quotapanel/`), holding the config file, SQLite database and per-account browser sessions. Set the `QUOTAPANEL_DATA_ROOT` environment variable to use another location.

## 🤝 Community

> Please mention "QuotaPanel" when adding us.

<table width="100%">
  <thead>
    <tr>
      <th width="50%">💬 WeChat Group</th>
      <th>🗣️ QQ Group</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><img src="https://open.tecmz.com/code_dynamic/wx" alt="WeChat group" style="width:100%; border-radius: 8px;"/></td>
      <td><img src="https://open.tecmz.com/code_dynamic/qq" alt="QQ group" style="width:100%; border-radius: 8px;"/></td>
    </tr>
  </tbody>
</table>

### 🌟 Support Us

- ⭐ Star us on [GitHub](https://github.com/open-tecmz/quotapanel) / [Gitee](https://gitee.com/open-tecmz/quotapanel)
- 🐛 Found a bug or have a suggestion? Open an Issue
- 🔌 Add a quota source: implement the `Provider` interface in `backend/quota/provider_*.go` and call `Register`
- 📖 Improve the docs: PRs are welcome

## 📄 License

Released under the [Apache-2.0](LICENSE) license — **free to use, modify and use commercially**.

---

<div align="center">
  <p>⭐ If this project helps you, please give us a Star!</p>
</div>
