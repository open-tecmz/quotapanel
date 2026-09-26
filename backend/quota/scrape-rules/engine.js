// webfetch 通用页面解析引擎（规则驱动，单一来源）。
//
// 设计原则：
//   1. 用 CSS 选择器在真实 DOM 上定位元素（不把页面转成纯文本再匹配）；
//   2. 能从属性（title / aria-* / href 等）拿到值就优先用属性；
//   3. match 仅用于「解析某个值内部的固定格式」（如 "已用 / 总量"），不用于定位元素；
//   4. 规则由 scrape-rules/<site>.json 提供，Go provider 与本脚本共用同一份。
//
// 引擎会被注入到页面上下文执行，规则占位符由构建时替换为站点规则 JSON。
(async () => {
  const RULES = __RULES__;
  const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
  const clean = (s) => (s || '').replace(/\s+/g, ' ').trim();

  // 等待页面动态渲染出关键节点
  const deadline = Date.now() + (RULES.timeoutMs || 12000);
  if (RULES.waitFor) {
    while (Date.now() < deadline && !document.querySelector(RULES.waitFor)) await sleep(500);
  }

  const out = {};
  for (const name of Object.keys(RULES.fields || {})) {
    const rule = RULES.fields[name];
    const nodes = Array.from(document.querySelectorAll(rule.selector || '*'));
    let value = null;
    for (const el of nodes) {
      let raw =
        rule.pick && rule.pick !== 'text' ? el.getAttribute(rule.pick) || '' : clean(el.textContent);
      if (!raw) continue;
      if (rule.match) {
        const m = raw.match(new RegExp(rule.match));
        if (!m) continue;
        raw = m[rule.group || 0];
      }
      value = clean(raw);
      break;
    }
    if (value !== null && rule.number) {
      const n = parseFloat(String(value).replace(/,/g, ''));
      value = isNaN(n) ? null : n;
    }
    out[name] = value;
  }
  return JSON.stringify({ status: 200, body: out });
})()
