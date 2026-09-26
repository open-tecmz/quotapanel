#!/usr/bin/env node
/**
 * webfetch — 网页额度取数工具（规则驱动，开发/维护用）
 *
 * 背景：部分供应商（如小米 MiMo、Claude、ChatGPT、Cursor）的额度只能在登录后的
 * 网页上看到，官方没有开放额度查询 API。本工具用真实浏览器打开该页面，按规则从
 * 渲染后的 DOM 中解析数据。
 *
 * 规则来源（唯一来源）：backend/quota/scrape-rules/<site>.json
 *   - Go 端 provider 通过 embed 使用同一份规则；本脚本直接读取同一文件。
 *   - 规则用 CSS 选择器在真实 DOM 上定位元素，优先读属性（title/aria-*），
 *     match 仅用于解析值内部的固定格式，不用于定位元素。
 *
 * ── 维护流程（平台改版导致取数失败时）──────────────────────────────
 *   1) node scripts/webfetch/cli.mjs login <site>    # 登录（会话过期时才需要）
 *   2) node scripts/webfetch/cli.mjs dump  <site>    # 抓取改版后的真实 HTML
 *   3) 编辑 backend/quota/scrape-rules/<site>.json   # 更新选择器（只改规则）
 *   4) node scripts/webfetch/cli.mjs check <site>    # 用最新 dump 离线校验规则
 *   5) 校验通过后，App 端 provider 自动生效（Go embed 同一份规则，无需改代码）
 *
 * 命令：
 *   sites                              列出已配置站点
 *   login <site> [--timeout 300]       打开浏览器人工登录一次
 *   dump  <site> [--headed]            导出渲染后的 HTML + 可见文本（分析结构用）
 *   check <site> [--file x.html]       对最新 dump 离线校验规则（--live 则走登录会话）
 *   fetch <site> [--headed]            用登录会话按规则抓取并输出 JSON
 *   rules <site>                       打印站点规则
 *   eval  <site> --file expr.js        在页面执行任意表达式（临时排查用）
 *
 * 运行依赖复用 test/node_modules 里的 playwright，无需单独安装。
 */

import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath, pathToFileURL } from 'node:url'

const HERE = path.dirname(fileURLToPath(import.meta.url))
const REPO = path.resolve(HERE, '../..')
const PROFILES = path.join(HERE, '.profiles')
const OUT = path.join(HERE, '.out')
const RULES_DIR = path.join(REPO, 'backend/quota/scrape-rules')

const sleep = (ms) => new Promise((r) => setTimeout(r, ms))

async function loadChromium() {
  const candidates = [
    path.join(REPO, 'test/node_modules/playwright/index.mjs'),
    path.join(REPO, 'frontend/node_modules/playwright/index.mjs'),
  ]
  for (const candidate of candidates) {
    if (fs.existsSync(candidate)) return (await import(pathToFileURL(candidate).href)).chromium
  }
  return (await import('playwright')).chromium
}

// 站点配置：只需声明登录判定；取数规则在 scrape-rules/<id>.json。
const SITES = {
  mimo: {
    name: 'Xiaomi MiMo Token Plan',
    url: 'https://platform.xiaomimimo.com/console/plan-manage',
    // 登录成功后会停在 /console/... 且页面上没有密码输入框；SSO 跳转过程中的 /sts 不算
    ready: async (page) => {
      const u = page.url()
      if (!u.startsWith('https://platform.xiaomimimo.com/console')) return false
      const hasPassword = await page.locator('input[type="password"]').count().catch(() => 0)
      return hasPassword === 0
    },
  },
  'deepseek-usage': {
    name: 'DeepSeek 用量明细',
    url: 'https://platform.deepseek.com/usage',
    ready: async (page) => {
      const u = page.url()
      if (!u.includes('platform.deepseek.com') || /sign[_-]?in|login/i.test(u)) return false
      const hasPassword = await page.locator('input[type="password"]').count().catch(() => 0)
      return hasPassword === 0
    },
  },
}

function siteOf(id) {
  const site = SITES[id]
  if (!site) {
    console.error(`未知站点: ${id}\n可用: ${Object.keys(SITES).join(', ')}`)
    process.exit(2)
  }
  return site
}

// 读取引擎 + 站点规则，组装成可在页面执行的脚本（与 Go 端 buildScrapeRuleScript 等价）。
function loadScrapeScript(id) {
  const enginePath = path.join(RULES_DIR, 'engine.js')
  const rulesPath = path.join(RULES_DIR, `${id}.json`)
  if (!fs.existsSync(enginePath) || !fs.existsSync(rulesPath)) return null
  const engine = fs.readFileSync(enginePath, 'utf8')
  const rules = fs.readFileSync(rulesPath, 'utf8')
  JSON.parse(rules) // JSON 非法时直接抛出
  return engine.replaceAll('__RULES__', rules)
}

function latestDumpHtml(id) {
  if (!fs.existsSync(OUT)) return null
  const files = fs
    .readdirSync(OUT)
    .filter((f) => f.startsWith(`${id}-`) && f.endsWith('.html'))
    .sort()
  return files.length ? path.join(OUT, files[files.length - 1]) : null
}

const statePathOf = (id) => path.join(PROFILES, id, 'storage-state.json')

function parseArgs(argv) {
  const flags = {}
  const positional = []
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i]
    if (a.startsWith('--')) {
      const key = a.slice(2)
      const next = argv[i + 1]
      if (next && !next.startsWith('--')) {
        flags[key] = next
        i++
      } else {
        flags[key] = true
      }
    } else {
      positional.push(a)
    }
  }
  return { flags, positional }
}

// 把上次 login 保存的 storageState 注入到新会话（部分站点登录 Cookie 为会话级，重启会丢）。
async function applyState(context, statePath) {
  if (!fs.existsSync(statePath)) return
  let state
  try {
    state = JSON.parse(fs.readFileSync(statePath, 'utf8'))
  } catch {
    return
  }
  const cookies = (state.cookies || []).map((c) => ({
    ...c,
    expires: c.expires && c.expires > 0 ? c.expires : Math.floor(Date.now() / 1000) + 86400,
  }))
  if (cookies.length) await context.addCookies(cookies).catch(() => {})
  for (const origin of state.origins || []) {
    await context.addInitScript(
      ({ o, items }) => {
        try {
          if (location.origin === o) for (const it of items) localStorage.setItem(it.name, it.value)
        } catch {
          // ignore
        }
      },
      { o: origin.origin, items: origin.localStorage || [] }
    )
  }
}

function writeDump(id, html, text) {
  fs.mkdirSync(OUT, { recursive: true })
  const stamp = new Date().toISOString().replace(/[:.]/g, '-').slice(0, 19)
  const htmlPath = path.join(OUT, `${id}-${stamp}.html`)
  const textPath = path.join(OUT, `${id}-${stamp}.txt`)
  fs.writeFileSync(htmlPath, html)
  fs.writeFileSync(textPath, text)
  return { htmlPath, textPath }
}

async function withPage(site, id, flags, fn) {
  const chromium = await loadChromium()
  const profile = path.join(PROFILES, id)
  fs.mkdirSync(profile, { recursive: true })
  const context = await chromium.launchPersistentContext(profile, {
    headless: !flags.headed,
    viewport: { width: 1440, height: 900 },
  })
  try {
    await applyState(context, statePathOf(id))
    const page = context.pages()[0] ?? (await context.newPage())
    await page.goto(site.url, { waitUntil: 'domcontentloaded', timeout: 60000 }).catch(() => {})
    await page.waitForLoadState('networkidle', { timeout: 30000 }).catch(() => {})
    await sleep(2500)
    return await fn(page)
  } finally {
    await context.close().catch(() => {})
  }
}

async function cmdSites() {
  for (const [id, site] of Object.entries(SITES)) {
    console.log(`${id}\t${site.name}\t${site.url}`)
  }
}

async function cmdRules(id) {
  const rulesPath = path.join(RULES_DIR, `${id}.json`)
  if (!fs.existsSync(rulesPath)) {
    console.error(`没有规则文件: ${rulesPath}`)
    process.exit(3)
  }
  console.log(fs.readFileSync(rulesPath, 'utf8'))
}

async function cmdLogin(id, flags) {
  const site = siteOf(id)
  const chromium = await loadChromium()
  const profile = path.join(PROFILES, id)
  fs.mkdirSync(profile, { recursive: true })
  const timeoutMs = (Number(flags.timeout) || 300) * 1000

  const context = await chromium.launchPersistentContext(profile, {
    headless: false,
    viewport: { width: 1280, height: 900 },
  })
  const page = context.pages()[0] ?? (await context.newPage())
  await page.goto(site.url, { waitUntil: 'domcontentloaded', timeout: 60000 }).catch(() => {})

  console.log(`请在打开的浏览器窗口中登录：${site.name}`)
  console.log(`地址：${site.url}`)

  let ok = false
  let streak = 0
  const deadline = Date.now() + timeoutMs
  while (Date.now() < deadline) {
    await sleep(1000)
    let pages = []
    try {
      pages = context.pages()
    } catch {
      break
    }
    if (pages.length === 0) break
    const current = pages[pages.length - 1]
    let ready = false
    try {
      ready = await site.ready(current)
    } catch {
      // 页面导航中，稍后重试
    }
    streak = ready ? streak + 1 : 0
    console.log(`  ${ready ? '✓' : '·'} ${current.url().slice(0, 110)}`)
    if (streak >= 2) {
      ok = true
      break
    }
  }

  if (ok) {
    await sleep(2500)
    try {
      const current = context.pages()[context.pages().length - 1]
      const html = await current.content()
      const text = await current.evaluate(() => document.body?.innerText ?? '')
      const { htmlPath, textPath } = writeDump(id, html, text)
      await context.storageState({ path: statePathOf(id) }).catch(() => {})
      console.log(`DUMP_OK ${htmlPath} ${textPath}`)
    } catch (e) {
      console.log(`DUMP_FAIL ${e?.message || e}`)
    }
  }

  await context.close().catch(() => {})
  console.log(ok ? 'LOGIN_OK' : 'LOGIN_TIMEOUT')
  process.exit(ok ? 0 : 1)
}

async function cmdDump(id, flags) {
  const site = siteOf(id)
  const { htmlPath, textPath } = await withPage(site, id, flags, async (page) => {
    const html = await page.content()
    const text = await page.evaluate(() => document.body?.innerText ?? '')
    const paths = writeDump(id, html, text)
    console.log(`url: ${page.url()}`)
    console.log(`html: ${paths.htmlPath} (${html.length} bytes)`)
    console.log(`text: ${paths.textPath}`)
    return paths
  })
  console.log(`DUMP_OK ${htmlPath} ${textPath}`)
}

// 用规则解析最近一次 dump 的 HTML（离线，不需要登录）；--live 则走登录会话。
async function cmdCheck(id, flags) {
  const site = siteOf(id)
  const script = loadScrapeScript(id)
  if (!script) {
    console.error(`缺少规则文件 backend/quota/scrape-rules/${id}.json 或 engine.js`)
    process.exit(3)
  }

  if (flags.live) {
    const result = await withPage(site, id, flags, (page) => page.evaluate(script))
    console.log(JSON.stringify(JSON.parse(result), null, 2))
    return
  }

  const file = flags.file || latestDumpHtml(id)
  if (!file || !fs.existsSync(file)) {
    console.error('没有可用的 dump，请先 `dump <site>`，或加 --live 直接查线上')
    process.exit(3)
  }
  const html = fs.readFileSync(file, 'utf8')
  const chromium = await loadChromium()
  const browser = await chromium.launch({ headless: true })
  try {
    const page = await browser.newPage()
    await page.setContent(html, { waitUntil: 'domcontentloaded' })
    const result = await page.evaluate(script)
    console.log(JSON.stringify({ source: file, ...JSON.parse(result) }, null, 2))
  } finally {
    await browser.close()
  }
}

async function cmdFetch(id, flags) {
  const site = siteOf(id)
  const script = loadScrapeScript(id)
  if (!script) {
    console.error(`缺少规则文件 backend/quota/scrape-rules/${id}.json 或 engine.js`)
    process.exit(3)
  }
  const result = await withPage(site, id, flags, (page) => page.evaluate(script))
  console.log(JSON.stringify(JSON.parse(result), null, 2))
}

async function cmdEval(id, flags, scriptPath) {
  const site = siteOf(id)
  if (!scriptPath) {
    console.error('eval 需要 --file <js 文件>')
    process.exit(2)
  }
  const expr = fs.readFileSync(scriptPath, 'utf8')
  const result = await withPage(site, id, flags, (page) => page.evaluate(expr))
  console.log(JSON.stringify(result, null, 2))
}

async function main() {
  const { flags, positional } = parseArgs(process.argv.slice(2))
  const [cmd, id] = positional
  switch (cmd) {
    case 'sites':
      return cmdSites()
    case 'login':
      return cmdLogin(id, flags)
    case 'dump':
      return cmdDump(id, flags)
    case 'check':
      return cmdCheck(id, flags)
    case 'fetch':
      return cmdFetch(id, flags)
    case 'rules':
      return cmdRules(id)
    case 'eval':
      return cmdEval(id, flags, flags.file)
    default:
      console.log(
        '用法: node scripts/webfetch/cli.mjs <sites|login|dump|check|fetch|rules|eval> <site> [--headed] [--live] [--timeout n] [--file x]'
      )
      process.exit(2)
  }
}

main().catch((e) => {
  console.error('webfetch 失败:', e?.message || e)
  process.exit(1)
})
