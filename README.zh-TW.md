# Lotus

<p align="center">
  <img src="https://pub-50dac11dd9ed4bc88adfff4ce0fcef3a.r2.dev/imgs/cards/purity.png" alt="Lotus" width="400" />
  <br/>
  <sub>圖片：<a href="https://store.steampowered.com/app/2868840/Slay_the_Spire_2/">Slay the Spire 2</a> 的 <b>Purity</b> 卡牌 —— 「從牌組中移除一張牌。」</sub>
</p>

<p align="center">
  <a href="README.md">English</a>
</p>

AI 程式助手的最小化配置推薦工具。

AI 程式工具生態系很吵 —— 上千個 skills、數十個 MCP registry、每週都有新框架。Lotus 只回答一個問題：**對你的專案來說，最少需要什麼？**

## 功能

1. **分析** 你的專案 — 偵測程式語言、框架、資料庫、CI、以及現有的 AI 設定
2. **推薦** 最少且有效的 skills、agents、MCP servers 和 hooks（來自策劃目錄）
3. **比較** 不同方案的 token 成本、時間和品質
4. **套用** 設定直接寫入 `.claude/` 目錄

## 安裝

```bash
go install github.com/texliao/lotus/cmd/lotus@latest
```

## 使用方式

```bash
# 偵測技術堆疊，掃描現有 AI 設定
lotus analyze .

# 取得推薦
lotus recommend .

# 預覽變更（不寫入檔案）
lotus apply . --dry-run

# 套用推薦設定
lotus apply .

# 瀏覽目錄
lotus catalog list
lotus catalog list --kind bundle
lotus catalog list --stack go

# 顯示項目詳情
lotus catalog show superpowers
```

## 目錄

Lotus 內建一份策劃過的 AI 程式工具目錄：

| 類型 | 說明 | 範例 |
|------|------|------|
| `skill` | 單一 SKILL.md 檔案 | minimax-frontend-dev, git-commit |
| `bundle` | 多檔案套件（skills + agents + hooks） | superpowers, gstack, D-Team |
| `source` | 可從中挑選的大型資料庫 | agency-agents（144 個角色） |
| `agent` | 單一 agent 定義 | - |
| `mcp-server` | MCP server 設定 | - |
| `hook` | Claude Code 的 shell hook | - |

### 收錄項目

**Bundles（套件）**
- [superpowers](https://github.com/obra/superpowers) — 結構化開發流程（腦力激盪、計畫、TDD、審查）。零依賴。
- [gstack](https://github.com/garrytan/gstack) — 虛擬工程團隊（計畫、建構、QA、部署、回顧）。需要 Bun + Playwright。
- [A-Team](https://github.com/chemistrywow31/A-Team) — 透過訪談生成客製化 agent 團隊的 meta-agent。

**Skills（技能）**
- [MiniMax-AI/skills](https://github.com/MiniMax-AI/skills) — 前端、全端、Android、iOS、Flutter、React Native、shader、PDF/PPTX/XLSX/DOCX 生成。
- git-commit — 規範化 commit 工作流程。

**Sources（來源）**
- [agency-agents](https://github.com/msitarzewski/agency-agents) — 144 個 agent 角色，涵蓋工程、設計、行銷、銷售、QA、遊戲開發等。

## 推薦原理

1. **技術堆疊偵測** — 掃描 `go.mod`、`package.json`、`Cargo.toml`、`pyproject.toml`、CI 設定、Docker Compose
2. **使用情境推斷** — 將偵測到的堆疊對應到使用情境（後端開發、前端開發等）
3. **目錄比對** — 找出符合使用情境和堆疊的項目
4. **評分** — `基礎分數 = 情境匹配 + 堆疊匹配 + 等級加成 - 重量懲罰`
5. **衝突解決** — 若兩個項目衝突（如 superpowers vs gstack），保留分數較高者

## 新增項目

在 `catalogdata/data/<kind>/` 下新增 YAML 檔案：

```yaml
id: my-skill
kind: skill
name: "My Skill"
source:
  registry: github
  repo: "user/repo"
  url: "https://github.com/user/repo"
use_cases:
  - backend-development
stacks:
  - go
requires:
  tools: []
  mcp_servers: []
  runtime: []
lotus:
  tier: recommended
  notes: "這個工具的用途與推薦原因。"
  conflicts_with: []
  pairs_well_with: []
```

## 授權

MIT
