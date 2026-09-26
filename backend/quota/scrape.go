package quota

import (
	"embed"
	"fmt"
	"strings"
)

// 页面抓取规则与通用引擎放在 scrape-rules/ 下，作为「单一来源」：
// Go provider 通过 embed 使用，开发工具 scripts/webfetch 直接读取同一份文件。
//
//go:embed scrape-rules/engine.js scrape-rules/*.json
var scrapeRulesFS embed.FS

// buildScrapeRuleScript 将站点规则注入通用引擎，生成可在页面上下文执行的脚本。
// 平台改版时只需更新对应的 scrape-rules/<site>.json 选择器，无需改 Go 代码。
func buildScrapeRuleScript(siteID string) (string, error) {
	engine, err := scrapeRulesFS.ReadFile("scrape-rules/engine.js")
	if err != nil {
		return "", fmt.Errorf("读取抓取引擎失败: %w", err)
	}
	rules, err := scrapeRulesFS.ReadFile("scrape-rules/" + siteID + ".json")
	if err != nil {
		return "", fmt.Errorf("缺少站点抓取规则 scrape-rules/%s.json: %w", siteID, err)
	}
	if !strings.Contains(string(engine), "__RULES__") {
		return "", fmt.Errorf("抓取引擎缺少 __RULES__ 占位符")
	}
	return strings.ReplaceAll(string(engine), "__RULES__", string(rules)), nil
}
