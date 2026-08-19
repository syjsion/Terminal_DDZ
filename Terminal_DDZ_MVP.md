# Terminal 斗地主 — MVP 产品与技术规格

> 文档状态：MVP Implementation Spec  
> 目标实现语言：Go  
> 目标平台：Windows / macOS / Linux  
> 交付对象：Codex / 开发实现  
> 项目暂定名：`terminal-ddz`  
> 游戏展示名：`Terminal 斗地主`

---

## 1. 项目概述

`Terminal 斗地主` 是一个完全运行在终端中的三人斗地主游戏。

核心目标：

- 不依赖浏览器或图形界面。
- Windows、macOS、Linux 只要有终端即可运行。
- Windows 支持 CMD、PowerShell、Windows Terminal。
- macOS / Linux 支持常见 ANSI 终端。
- 用户通过简单的键盘操作完成整局游戏，不要求手动输入牌面。
- 内置本地 AI，不配置任何网络服务也可以直接游玩。
- 支持将两名 AI 玩家分别配置为 OpenAI-compatible LLM。
- 两名 LLM AI 可以使用不同的 `base_url`、`api_key`、`model`。
- 支持实时记牌器和出牌历史。
- 真实配置保存在本地 `config.toml`，Git 仓库忽略该文件。
- 仓库提供 `config.example.toml`。
- 所有牌面使用 `10`，**禁止使用 `T` 表示 10**。

MVP 不追求动画、联网多人、账号系统或复杂皮肤，优先保证：

1. 斗地主规则正确。
2. 操作简单。
3. 跨平台稳定。
4. 本地 AI 可玩。
5. LLM AI 接入可靠。
6. AI 不得获得不应该知道的隐藏信息。
7. API 异常不会导致整局游戏卡死。

---

## 2. 非目标

以下内容不属于 MVP：

- 真人联网对战。
- Web UI / GUI。
- 用户注册、登录、云存档。
- 排位系统。
- 金币、商城、付费系统。
- 语音。
- AI 聊天或角色扮演。
- 多人房间。
- 自定义牌类规则编辑器。
- 扑克牌花色对大小产生影响。
- 复杂动画。
- 自动更新器。
- OpenAI Responses API、Anthropic 原生协议等非 OpenAI-compatible Chat Completions 协议。
- GUI 配置编辑器。
- API Key 加密存储。

---

## 3. 技术栈

### 3.1 语言

使用 Go。

建议：

- Go 1.24+ 或项目创建时的稳定版本。
- 避免 CGO。
- 尽量保持纯 Go，实现真正的跨平台单二进制发布。

### 3.2 TUI

推荐：

- Bubble Tea
- Lip Gloss
- Bubbles（仅在确实需要组件时使用）

原则：

- TUI 与游戏引擎分离。
- 游戏规则层不得依赖 Bubble Tea。
- AI 层不得直接操作 TUI。
- 所有终端绘制通过统一 UI 层完成。

### 3.3 HTTP

优先使用 Go 标准库：

- `net/http`
- `context`
- `encoding/json`

可以不引入额外 HTTP Client 依赖。

### 3.4 配置

建议 TOML。

可使用成熟轻量 TOML 库。

---

## 4. 项目目录建议

```text
terminal-ddz/
├── cmd/
│   └── terminal-ddz/
│       └── main.go
│
├── internal/
│   ├── config/
│   │   ├── config.go
│   │   └── load.go
│   │
│   ├── game/
│   │   ├── card.go
│   │   ├── deck.go
│   │   ├── hand.go
│   │   ├── hand_type.go
│   │   ├── rules.go
│   │   ├── move.go
│   │   ├── legal_moves.go
│   │   ├── bidding.go
│   │   ├── state.go
│   │   ├── engine.go
│   │   ├── counter.go
│   │   └── score.go
│   │
│   ├── player/
│   │   ├── player.go
│   │   └── view.go
│   │
│   ├── ai/
│   │   ├── ai.go
│   │   ├── fallback.go
│   │   ├── local/
│   │   │   ├── easy.go
│   │   │   ├── normal.go
│   │   │   └── hard.go
│   │   └── llm/
│   │       ├── client.go
│   │       ├── openai.go
│   │       ├── prompt.go
│   │       ├── response.go
│   │       └── player.go
│   │
│   ├── tui/
│   │   ├── app.go
│   │   ├── model.go
│   │   ├── update.go
│   │   ├── view.go
│   │   ├── game_view.go
│   │   ├── counter_view.go
│   │   ├── history_view.go
│   │   ├── help_view.go
│   │   └── keys.go
│   │
│   └── version/
│       └── version.go
│
├── testdata/
├── scripts/
├── config.example.toml
├── .gitignore
├── go.mod
├── go.sum
├── LICENSE
└── README.md
```

可以根据实际实现微调，但必须保持：

```text
game engine
player view
local AI
LLM AI
TUI
config
```

彼此解耦。

---

## 5. 扑克牌定义

完整牌堆共 54 张。

普通点数：

```text
3 4 5 6 7 8 9 10 J Q K A 2
```

王：

```text
SJ = 小王
BJ = 大王
```

### 5.1 显示规则

必须显示：

```text
10
```

不得显示：

```text
T
```

包括：

- TUI
- 日志
- AI Prompt
- JSON 中的人类可读牌面
- README
- 测试快照

### 5.2 点数大小

从小到大：

```text
3 < 4 < 5 < 6 < 7 < 8 < 9 < 10 < J < Q < K < A < 2 < SJ < BJ
```

### 5.3 花色

花色仅用于区分实体牌，可在内部保存，但**不参与斗地主大小比较**。

MVP 默认 UI 可以不显示花色，只显示点数。

---

## 6. 发牌

每局：

1. 创建 54 张牌。
2. 随机洗牌。
3. 三名玩家各 17 张。
4. 剩余 3 张作为底牌。
5. 进入叫地主阶段。
6. 地主确定后获得 3 张底牌，因此地主共 20 张牌。
7. 两名农民各 17 张。

随机应使用 Go 标准随机方案，不要求密码学安全。

测试必须允许注入固定牌堆或固定 seed。

---

## 7. 座位与玩家

固定三名玩家：

```text
Seat 0
Seat 1
Seat 2
```

默认：

- Seat 0：Human
- Seat 1：AI-1
- Seat 2：AI-2

允许配置：

- Local AI + Local AI
- Local AI + LLM AI
- LLM AI + Local AI
- LLM AI + LLM AI

MVP 不要求 AI 替换 Human，因此至少保留一个真人用户。

---

## 8. 叫地主规则

为了避免不同地区规则差异，MVP 明确定义为 **0/1/2/3 分叫地主**。

### 8.1 流程

1. 随机选择本局首叫座位。
2. 按顺时针，每名玩家最多行动一次。
3. 玩家可选择：
   - `不叫`，记为 0。
   - 当前最高叫分之上的分数。
4. 可叫分数：
   - 1
   - 2
   - 3
5. 后续玩家必须高于当前最高分，否则只能不叫。
6. 任何玩家叫 3 分，立即结束叫地主。
7. 三人都行动后，最高叫分者成为地主。
8. 如果三人全部不叫，重新洗牌发牌并重新开始。
9. 地主获得 3 张底牌。
10. 地主获得本局第一次出牌权。

示例：

```text
AI-1: 1
AI-2: 2
Human: 不叫
=> AI-2 成为地主，底分 2
```

### 8.2 本地 AI 叫分

Easy：

- 基于简单牌力分数随机决定。

Normal：

- 使用牌力评估决定 0~3 分。

Hard：

- 在 Normal 基础上额外考虑王、2、炸弹、长顺子、飞机潜力和牌组结构。

### 8.3 LLM 叫分

与出牌类似，程序提供合法叫分 ID：

```json
{
  "legal_bids": [0, 2, 3]
}
```

LLM 只能返回其中一个。

---

## 9. 牌型

MVP 必须支持以下牌型。

### 9.1 单张 Single

```text
3
A
2
SJ
BJ
```

### 9.2 对子 Pair

相同点数两张：

```text
33
1010
AA
22
```

王不能组成对子。

### 9.3 三张 Triple

相同点数三张：

```text
333
JJJ
222
```

### 9.4 三带一 TripleWithSingle

```text
333 + 4
AAA + SJ
```

带牌不能使用三张主体相同点数的第四张。

### 9.5 三带二 TripleWithPair

```text
333 + 44
AAA + 1010
```

附带部分必须是合法对子。

### 9.6 顺子 Straight

至少 5 张连续单牌。

允许：

```text
34567
678910J
10JQKA
```

禁止：

```text
JQKA2
QKA2SJ
```

即：

- 2 不能进入顺子。
- SJ/BJ 不能进入顺子。

比较时必须牌数相同，只比较最高点数。

### 9.7 连对 PairStraight

至少 3 个连续对子，即至少 6 张牌。

允许：

```text
334455
7788991010
QQKKAA
```

禁止包含：

```text
2
SJ
BJ
```

比较时对子数量必须一致。

### 9.8 飞机不带翅膀 Plane

至少 2 个连续三张。

允许：

```text
333444
777888999
QQQKKKAAA
```

禁止主体包含 2、SJ、BJ。

比较时连续三张数量必须一致。

### 9.9 飞机带单 PlaneWithSingles

假设飞机主体包含 `N` 组连续三张，则必须附带 `N` 张单牌。

例如：

```text
333444 + 5 + 6
777888999 + 3 + J + A
```

MVP 明确规则：

- 翅膀不能使用飞机主体点数。
- 单翅膀按“牌张”计算。
- 单翅膀允许两张牌具有相同点数。
- 单翅膀可以包含 2、SJ、BJ。
- 主体不能包含 2、SJ、BJ。
- 比较时飞机主体长度必须一致。

### 9.10 飞机带对 PlaneWithPairs

假设主体包含 `N` 组连续三张，则必须附带 `N` 个对子。

例如：

```text
333444 + 55 + 66
777888999 + 33 + JJ + AA
```

MVP 明确规则：

- 翅膀对子点数必须与飞机主体点数不同。
- 各附带对子必须使用不同点数。
- 对子可以包含 2。
- 王不能组成对子。
- 主体不能包含 2、SJ、BJ。
- 比较时飞机主体长度必须一致。

### 9.11 四带二单 FourWithTwoSingles

```text
6666 + 3 + A
9999 + JJ
```

这里的 `JJ` 可以作为两张单牌使用。

MVP 明确：

- 主体为四张相同点数。
- 附带两张任意非主体点数的牌。
- 两张附带牌可以点数相同。
- 可包含 2、SJ、BJ。
- 四带二不是炸弹，不享受炸弹优先级。

### 9.12 四带二对 FourWithTwoPairs

```text
6666 + 33 + 44
AAAA + JJ + QQ
```

规则：

- 主体为四张相同点数。
- 附带两个不同点数的对子。
- 附带对子点数不能等于主体。
- 王不能组成对子。
- 四带二对不是炸弹。

### 9.13 炸弹 Bomb

四张相同点数：

```text
3333
10101010
AAAA
2222
```

炸弹：

- 可以压过任意非炸弹、非王炸牌型。
- 炸弹之间比较主体点数。
- `2222` 是普通炸弹中最大。

### 9.14 王炸 Rocket

```text
SJ + BJ
```

王炸：

- 是全游戏最大牌型。
- 可以压过任何牌。
- 不存在能压王炸的合法动作。

---

## 10. 牌型比较

### 10.1 普通牌型

只有在以下条件全部一致时才能比较：

- 牌型相同。
- 牌张结构一致。
- 顺子长度一致。
- 连对数量一致。
- 飞机主体长度一致。

然后比较牌型的主体点数。

例如：

```text
34567 < 45678
33344 < 44455
333444 + 5 + 6 < 444555 + 7 + 8
```

### 10.2 炸弹

任意炸弹大于任意普通牌型。

```text
3333 > AAA
3333 > 34567
```

### 10.3 王炸

```text
SJ BJ > 任意炸弹
```

---

## 11. 出牌轮次

### 11.1 首出

地主获得第一手出牌权。

当某玩家拥有一个新的出牌轮次时：

- 不能选择 PASS。
- 可以打任意合法牌型。

### 11.2 跟牌

后续玩家：

- 可以打出能够压过当前目标牌的合法动作。
- 或选择 PASS。

### 11.3 两家连续 PASS

例如：

```text
A: 99
B: PASS
C: PASS
```

则：

- 当前一轮结束。
- A 获得新的首出权。
- `last_move` 清空。
- A 不能 PASS。

### 11.4 胜利条件

任何玩家手牌数变为 0，立即结束本局。

- 地主先出完：地主胜。
- 任一农民先出完：两名农民共同胜。

---

## 12. 倍数与计分

MVP 实现简单但完整的对局计分。

### 12.1 初始底分

等于地主叫分：

```text
1 / 2 / 3
```

### 12.2 炸弹与王炸

每出现一次炸弹或王炸：

```text
倍率 × 2
```

### 12.3 春天

地主获胜，并且两名农民整局都没有成功出过牌：

```text
倍率 × 2
```

### 12.4 反春天

农民获胜，并且地主在第一手之后再未成功出牌：

```text
倍率 × 2
```

### 12.5 最终积分

MVP 只用于统计，不涉及虚拟货币。

设：

```text
score = bid_score * multiplier
```

地主胜：

```text
地主 +2 * score
农民1 -score
农民2 -score
```

农民胜：

```text
地主 -2 * score
农民1 +score
农民2 +score
```

---

## 13. 游戏状态模型

建议核心结构类似：

```go
type GameState struct {
    Phase        Phase
    Round        int
    CurrentSeat  int
    LandlordSeat int

    Players      [3]PlayerState
    BottomCards  []Card

    CurrentTrick TrickState
    History      []ActionRecord

    BidState     BidState

    BidScore     int
    Multiplier   int

    WinnerTeam   Team
    Finished     bool
}
```

必须保证所有状态修改都经过 `game.Engine`。

TUI 和 AI 不允许直接修改：

- 玩家手牌。
- 当前出牌者。
- 倍率。
- 历史。
- 胜负状态。

---

## 14. Move 模型

建议：

```go
type Move struct {
    ID       int
    Type     HandType
    Cards    []Card
    MainRank Rank
    Length   int
    IsPass   bool
}
```

`ID` 是当前决策上下文中的临时唯一 ID。

例如：

```text
0 PASS
1 33
2 JJ
3 6666
```

下一次决策可以重新编号。

---

## 15. 合法动作生成器

这是项目最关键的模块之一。

必须提供类似：

```go
func GenerateLegalMoves(
    hand []Card,
    target *Move,
    canPass bool,
) []Move
```

### 15.1 首出

`target == nil`

生成该手牌可以组成的所有合法牌型。

### 15.2 跟牌

根据 `target`：

- 生成所有同类型且能压过目标的动作。
- 生成所有炸弹。
- 如果有 SJ + BJ，生成王炸。
- 如果允许 PASS，添加 PASS。

### 15.3 排序

合法动作必须稳定排序，方便 TUI 和 AI。

建议默认顺序：

1. PASS
2. 普通牌，按消耗牌力从低到高
3. 炸弹
4. 王炸

首出无 PASS。

必须保证同一局面产生确定的合法动作顺序。

---

## 16. 规则校验

即使动作来自合法动作列表，Engine 执行前仍应验证。

建议：

```go
func ValidateMove(
    state *GameState,
    seat int,
    move Move,
) error
```

检查：

- 是否轮到该玩家。
- 玩家是否拥有这些牌。
- PASS 是否允许。
- 牌型是否合法。
- 是否能压过当前目标牌。
- 游戏是否已经结束。

不得信任：

- TUI。
- Local AI。
- LLM AI。
- 测试之外的任何调用者。

---

## 17. PlayerView：信息隔离

LLM 和本地 AI 不应直接拿到完整 `GameState`。

必须构造玩家视角：

```go
type PlayerView struct {
    Seat          int
    Role          Role
    OwnCards      []Card

    OtherCounts   map[int]int
    LandlordSeat  int

    BottomPublic  []Card
    PlayedCards   []ActionRecord
    LastMove      *PublicMove

    BidScore      int
    Multiplier    int
}
```

### 17.1 绝对禁止泄露

AI 不得看到：

- 其他玩家当前手牌。
- 未公开隐藏牌。
- 通过程序内部状态可知但真实玩家不可知的信息。

### 17.2 底牌

地主确定后，三张底牌视为公开信息。

因此所有玩家和 AI 都可以看到底牌。

---

## 18. 记牌器

记牌器只使用合法公开信息。

计算：

```text
54 张完整牌
-
自己的当前手牌
-
所有其他玩家已经成功打出的牌
=
自己视角下的未见牌
```

注意：

- 自己已经打出的牌已经出现在历史中，不应重复扣除。
- 实现时可以从“完整牌集 - 自己当前手牌 - 所有已出牌”直接计算。
- 地主底牌已经公开，但如果其中某张仍然在别人的手上，它依然属于“未见于自己手牌但身份已知来源”的牌。为了记牌器语义简单，MVP 的“剩余未知牌”按“当前未在自己手中且尚未打出”的牌计数。

建议 UI：

```text
┌─ 记牌器 ──────────────────────────────────────────────┐
│ 3  4  5  6  7  8  9  10  J  Q  K  A  2  SJ  BJ     │
│ 1  2  0  3  2  1  0   2  3  1  4  1  2   1   0     │
└───────────────────────────────────────────────────────┘
```

必须使用 `10`。

MVP 记牌器额外显示：

- 地主剩余手牌数。
- 另一名农民剩余手牌数。
- 已出现炸弹数量。
- 王炸是否仍可能存在。

不得推断或显示精确对手手牌。

---

## 19. 本地 AI 接口

统一 AI 接口：

```go
type Agent interface {
    ChooseBid(
        ctx context.Context,
        view PlayerView,
        legalBids []int,
    ) (int, error)

    ChooseMove(
        ctx context.Context,
        view PlayerView,
        legalMoves []game.Move,
    ) (int, error)
}
```

`ChooseMove` 返回 `Move.ID`，而不是自行构造牌。

---

## 20. Local AI

MVP 支持：

```text
easy
normal
hard
```

### 20.1 Easy

目标：

- 合法。
- 明显弱。
- 快速。

策略：

- 跟牌时从所有非炸弹合法动作中随机选择。
- 有一定概率 PASS。
- 尽量不主动使用炸弹。
- 首出随机选择普通合法动作。
- 王炸最后考虑。

### 20.2 Normal

目标：

- 有基本斗地主策略。
- 默认难度。

核心启发式：

跟牌：

1. 优先选择能压住目标的最小成本普通牌。
2. 如果队友是上一手成功出牌者，默认倾向 PASS。
3. 对手即将只剩 1~2 张时，提高抢回出牌权的优先级。
4. 默认保留炸弹和王炸。
5. 自己剩余牌较少时提高强行接牌倾向。

首出：

1. 优先减少手牌“组数”。
2. 优先长顺子、连对、飞机等高效率组合。
3. 避免无意义拆炸弹。
4. 在不明显破坏牌型的情况下优先出较小牌。

### 20.3 Hard

目标：

- 在 Normal 上增加轻量搜索。
- 不要求专业比赛 AI。

建议：

1. 对候选 Move 模拟执行。
2. 对剩余手牌做简单分组评估。
3. 估算：
   - 剩余出牌组数。
   - 高牌控制力。
   - 炸弹价值。
   - 是否能连续走完。
4. 农民考虑队友协作。
5. 对手剩 1 张时优先阻止其重新获得首出。
6. 候选数量过大时先做启发式裁剪。

禁止使用对手真实手牌。

---

## 21. LLM AI

### 21.1 设计原则

LLM 不负责：

- 判断某组牌是否合法。
- 自己生成牌。
- 修改游戏状态。
- 读取其他玩家隐藏手牌。

LLM 只负责：

```text
在程序已经生成的合法动作 ID 中选择一个。
```

### 21.2 OpenAI-compatible MVP 协议

使用：

```text
POST {base_url}/chat/completions
```

约定：

- `base_url` 建议包含 `/v1`。
- 实际 URL 通过安全拼接得到，避免出现重复 `/`。
- 使用 Bearer Token：

```text
Authorization: Bearer <api_key>
```

请求核心字段：

```json
{
  "model": "model-name",
  "messages": [
    {
      "role": "system",
      "content": "..."
    },
    {
      "role": "user",
      "content": "..."
    }
  ],
  "temperature": 0.2,
  "max_tokens": 100
}
```

兼容性优先，不强依赖：

- tool calling
- function calling
- JSON Schema
- response_format

### 21.3 模型输出

要求模型仅输出 JSON：

```json
{"move": 3}
```

叫地主：

```json
{"bid": 2}
```

程序必须允许模型偶尔包裹 Markdown code fence，并进行轻量清洗。

### 21.4 解析失败

如果出现：

- HTTP 非 2xx。
- 超时。
- JSON 解析失败。
- 没有 choices。
- 返回 ID 不在合法集合中。
- 内容为空。

则认为 LLM 本次决策失败。

### 21.5 Retry

MVP 每次决策最多：

```text
1 次正常请求
+ 1 次重试
```

重试 Prompt 明确：

```text
Your previous response was invalid.
Return JSON only and choose one of the provided IDs.
```

如果仍失败：

```text
fallback → Local AI
```

### 21.6 Timeout

每个 Provider 可配置：

```toml
timeout_seconds = 15
```

建议默认 15 秒。

Context 必须支持取消。

退出游戏时不得留下阻塞 goroutine。

---

## 22. LLM Prompt

Prompt 必须短、结构化、不可泄露隐藏信息。

示例：

```text
You are playing Dou Dizhu (斗地主).

You are AI-1.
Role: Farmer.
Landlord: Seat 2.

Card rank:
3 < 4 < 5 < 6 < 7 < 8 < 9 < 10 < J < Q < K < A < 2 < SJ < BJ

Your cards:
3 3 5 6 7 8 9 10 J Q K A 2

Cards remaining:
Seat 0: 8
Seat 2: 3

Public bottom cards:
4 A BJ

Recent actions:
Seat 2: 99
Seat 0: PASS

Legal moves:
0 = PASS
1 = JJ
2 = 2222

Choose the strategically best move.
As a farmer, cooperate with the other farmer when appropriate.

Return JSON only:
{"move": <id>}
```

禁止在 Prompt 使用 `T`。

### 22.1 历史长度

为控制 token：

- 默认只发送最近 12 个 Action。
- 另外发送累计已出牌统计。
- 不需要每次发送从发牌开始的全部自然语言记录。

### 22.2 Legal Moves 数量过大

首出时合法动作可能很多。

MVP 可以限制发给 LLM 的候选数，例如：

```text
max_llm_legal_moves = 80
```

裁剪必须由本地启发式完成，并保证：

- 不裁掉明显关键牌型。
- 至少保留各种类型的合理候选。
- 如果候选少于阈值，则全部发送。

注意：LLM 只能选择“提供给它”的动作，但提供动作仍然必须全部合法。

---

## 23. LLM Provider 配置

配置允许多个 Provider。

推荐：

```toml
[providers.openai]
base_url = "https://api.openai.com/v1"
api_key = ""
model = "example-model"
timeout_seconds = 15

[providers.provider_b]
base_url = "https://example.com/v1"
api_key = ""
model = "example-model-b"
timeout_seconds = 20
```

玩家引用 Provider：

```toml
[ai1]
type = "llm"
provider = "openai"

[ai2]
type = "llm"
provider = "provider_b"
```

因此可以：

```text
Human
vs
Provider A
vs
Provider B
```

两个 Provider 可以：

- URL 不同。
- Key 不同。
- Model 不同。
- Timeout 不同。

---

## 24. config.toml

建议完整 MVP 配置：

```toml
[general]
language = "zh-CN"
theme = "work"
show_card_counter = true
show_history_panel = true
ai_delay_ms = 350

[game]
local_ai_difficulty = "normal"
llm_fallback = true
max_llm_legal_moves = 80

[player]
name = "You"

[ai1]
name = "Worker-01"
type = "local"
difficulty = "normal"

[ai2]
name = "Worker-02"
type = "local"
difficulty = "normal"

[providers.openai]
base_url = "https://api.openai.com/v1"
api_key = ""
model = ""
timeout_seconds = 15

[providers.provider_b]
base_url = "https://example.com/v1"
api_key = ""
model = ""
timeout_seconds = 15
```

LLM 示例：

```toml
[ai1]
name = "Worker-01"
type = "llm"
provider = "openai"

[ai2]
name = "Worker-02"
type = "llm"
provider = "provider_b"
```

---

## 25. 配置文件加载

CLI：

```text
terminal-ddz
terminal-ddz --config ./my-config.toml
```

加载顺序：

1. 如果指定 `--config`，使用该路径。
2. 否则尝试当前工作目录的 `./config.toml`。
3. 如果不存在 `config.toml`：
   - 不报错退出。
   - 使用默认配置。
   - 两名 AI 均为 Local Normal。
4. 如果配置文件存在但 TOML 无法解析：
   - 明确显示错误。
   - 退出码非 0。

### 25.1 Provider 校验

只有当某 AI 被配置为 `type = "llm"` 时才校验对应 Provider。

必须要求：

- provider 存在。
- base_url 非空。
- api_key 非空。
- model 非空。
- timeout_seconds > 0。

Local AI 不要求任何 Key。

---

## 26. API Key 保存策略

MVP 明确允许：

```text
API Key 明文写入 config.toml
```

不要求环境变量。

仓库必须包含：

```gitignore
config.toml
*.local.toml
```

仓库不得提交真实 Key。

必须提供：

```text
config.example.toml
```

其中：

```toml
api_key = ""
```

README 提醒：

- `config.toml` 包含敏感凭据。
- 不要提交。
- 不要分享。
- macOS / Linux 可选执行：

```bash
chmod 600 config.toml
```

MVP 不实现 Key 加密。

---

## 27. TUI 设计

### 27.1 设计原则

风格：

```text
低调
终端原生
信息密度高
少动画
黑白/低饱和
像开发工具或任务监控器
```

默认主题名：

```text
work
```

不要依赖特殊 Unicode 图形才能使用。

可以使用 Box Drawing 字符，但应提供 ASCII fallback 或保证主要信息即使字符显示异常仍可读。

### 27.2 最小终端尺寸

建议：

```text
80 x 24
```

如果低于最低尺寸：

```text
Terminal window too small.
Minimum: 80x24
Current: 70x20
Resize the terminal to continue.
```

监听 resize 事件并自动恢复。

---

## 28. 主游戏界面

示意：

```text
┌──────────────────────────────────────────────────────────────┐
│ Terminal 斗地主                         Round #12   x4       │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│      Worker-01 [农民]  9           Worker-02 [地主]  4       │
│                                                              │
│                     Last: 99                                 │
│                   by Worker-02                               │
│                                                              │
├──────────────────────────────────────────────────────────────┤
│ Your hand [11]                                               │
│                                                              │
│ 3  3  4  5  6  7  8  9  10  J  A                           │
│                                                              │
│ > [1] 33                                                    │
│   [2] PASS                                                  │
│                                                              │
├──────────────────────────────────────────────────────────────┤
│ ↑/↓ Select   Enter Play   Space Pass   R Counter   H History │
│ Q Quit   ? Help                                             │
└──────────────────────────────────────────────────────────────┘
```

必须显示：

- 当前身份。
- 两名对手身份。
- 各自剩余牌数。
- 当前倍率。
- 上一手合法牌。
- 当前动作候选。
- 自己完整手牌。
- 当前轮到谁。

---

## 29. 键位

MVP：

```text
↑ / k          上一个候选
↓ / j          下一个候选
Enter          确认当前候选
Space          PASS（仅可 PASS 时）
R              打开/关闭记牌器
H              打开/关闭出牌历史
?              帮助
Q              请求退出
Esc            关闭弹层 / 返回游戏
```

可选：

```text
1-9            快速选择对应候选
```

如果合法动作超过可视区域：

- 列表滚动。
- 不要求用户输入牌。

---

## 30. 首出候选过多时的 UI

首出可能有大量合法动作。

TUI 不应一次显示数十行。

建议：

- 默认展示排序后的前 N 个候选，例如 12。
- 支持翻页/滚动。
- 可以按牌型分组。
- 状态栏显示：

```text
Moves 1-12 / 47
```

后续可以增加搜索，但不是 MVP 必需。

---

## 31. 出牌历史

按 `H`：

```text
┌─ History ───────────────────────────────────────────────┐
│ #01 Worker-02  34567                                   │
│ #02 You        45678                                   │
│ #03 Worker-01  PASS                                    │
│ #04 Worker-02  PASS                                    │
│ #05 You        33                                      │
└─────────────────────────────────────────────────────────┘
```

历史必须包含：

- 顺序编号。
- 玩家名。
- 动作。
- PASS。
- 炸弹时可以标记 `BOMB`。
- 王炸可标记 `ROCKET`。

仍然使用 `10`。

---

## 32. AI 思考状态

LLM 调用期间不能冻结界面。

显示：

```text
Worker-02 thinking...
```

或：

```text
Worker-02 [LLM] thinking...
```

HTTP 请求应在 Bubble Tea command / goroutine 模型中异步执行。

主 TUI event loop 不得被网络请求阻塞。

用户按 Q 后应能取消 context 并退出。

---

## 33. AI 延迟

本地 AI 默认增加轻微人为延迟：

```toml
ai_delay_ms = 350
```

目的：

- 让对局动作可读。
- 避免瞬间刷屏。

LLM 实际网络等待已经足够，不需要额外 delay。

如果配置：

```text
ai_delay_ms = 0
```

本地 AI 可立即行动。

---

## 34. LLM Fallback

配置：

```toml
[game]
llm_fallback = true
```

当 LLM 请求失败：

```text
Worker-02 LLM request failed: timeout
Fallback: local normal AI
```

然后本回合由本地 AI 在同一合法动作列表中选择。

默认 fallback 难度：

```text
normal
```

可以复用：

```toml
local_ai_difficulty = "normal"
```

如果：

```text
llm_fallback = false
```

则显示错误弹层，允许：

- Retry
- Switch to local AI
- Quit

但 MVP 推荐默认始终开启 fallback。

---

## 35. 并发模型

原则：

- Game Engine 状态修改应串行。
- 不允许多个 goroutine 同时修改 GameState。
- 网络请求可异步。
- 请求结果通过 TUI message 回到主循环，再执行 Engine action。

建议：

```text
TUI event loop
   |
   +--> request AI decision asynchronously
   |
   <-- AIResultMsg
   |
Engine.ApplyMove()
```

每个 AI 请求绑定：

- 当前 game ID。
- 当前 turn ID。

如果结果返回时局面已变化，则丢弃旧结果。

---

## 36. 错误处理

### 36.1 配置错误

明确显示字段。

示例：

```text
config error:
ai2.type is "llm" but providers.deepseek.api_key is empty
```

### 36.2 LLM 错误

不得直接 panic。

记录：

- provider name
- HTTP status
- timeout
- parse error

但日志中**不得打印完整 API Key**。

### 36.3 游戏逻辑错误

规则层返回 typed error。

开发模式可 panic/assert，但正常用户流程不应因为模型返回错误动作而崩溃。

---

## 37. 日志

MVP 默认不输出调试日志破坏 TUI。

支持：

```text
terminal-ddz --debug
```

Debug 日志可以写：

```text
./terminal-ddz.log
```

或者 stderr（开发时）。

必须脱敏：

```text
api_key = sk-abc...xyz
```

禁止完整记录 Key。

建议不要默认记录完整 LLM 请求体，因为其中包含用户对局信息和可能的 Provider 数据。

---

## 38. CLI

MVP：

```text
terminal-ddz
terminal-ddz --config <path>
terminal-ddz --debug
terminal-ddz --version
terminal-ddz --help
```

可选：

```text
terminal-ddz --seed 1234
```

`--seed` 主要供开发与测试。

---

## 39. 跨平台要求

必须支持：

### Windows

- Windows 10/11
- CMD
- PowerShell
- Windows Terminal
- amd64
- arm64（如构建环境允许）

### macOS

- Intel amd64
- Apple Silicon arm64

### Linux

至少：

- amd64
- arm64

避免：

- 平台专用 shell 命令。
- 强依赖 bash。
- CGO。
- 仅 Unix 可用的终端控制方案。

---

## 40. 构建产物

建议 GitHub Releases：

```text
terminal-ddz_windows_amd64.exe
terminal-ddz_windows_arm64.exe
terminal-ddz_darwin_amd64
terminal-ddz_darwin_arm64
terminal-ddz_linux_amd64
terminal-ddz_linux_arm64
```

版本信息建议通过 ldflags 注入。

---

## 41. GitHub Actions

MVP 建议配置：

### CI

每次 PR / push：

```text
go fmt check
go vet
go test ./...
go build ./...
```

### Release

tag：

```text
v0.1.0
```

自动编译跨平台二进制并上传 Release。

---

## 42. `.gitignore`

至少：

```gitignore
# Local configuration / secrets
config.toml
*.local.toml

# Logs
*.log

# Build output
dist/
bin/

# IDE
.idea/
.vscode/

# OS
.DS_Store
Thumbs.db
```

注意：

如果 `.vscode` 中未来需要提交公共配置，可以再调整。

---

## 43. 测试策略

规则测试是 MVP 的核心。

### 43.1 Card

测试：

- Rank 顺序。
- `10` 序列化显示。
- SJ/BJ。
- 排序。

### 43.2 Hand Detection

每种牌型必须有：

- 正例。
- 反例。
- 边界案例。

必须覆盖：

```text
Single
Pair
Triple
TripleWithSingle
TripleWithPair
Straight
PairStraight
Plane
PlaneWithSingles
PlaneWithPairs
FourWithTwoSingles
FourWithTwoPairs
Bomb
Rocket
```

### 43.3 Straight

覆盖：

```text
34567       valid
10JQKA      valid
JQKA2       invalid
23456       invalid
```

### 43.4 Plane

覆盖：

- 两组三张。
- 三组三张。
- 带单。
- 带对。
- 主体包含 2 时 invalid。
- 主体重复/断开时 invalid。
- 翅膀使用主体点数时 invalid。

### 43.5 Compare

覆盖：

- 同类型可压。
- 长度不同不可压。
- 炸弹压普通牌。
- 大炸弹压小炸弹。
- 王炸压炸弹。
- 普通牌不能压炸弹。

### 43.6 Legal Move Generator

使用固定手牌和目标动作验证生成集合。

必须验证：

- 不生成玩家没有的牌。
- 不生成非法牌型。
- 跟牌时不会产生压不过目标的普通牌。
- 炸弹始终可作为额外候选。
- 王炸始终最高。
- 首出不包含 PASS。
- 跟牌包含 PASS。

### 43.7 Engine

覆盖：

- 正常发牌。
- 叫地主。
- 三人全不叫重新发牌。
- 地主拿底牌。
- 两次 PASS 重置 trick。
- 玩家出完牌立即结束。
- 计分。
- 炸弹倍率。
- 春天。
- 反春天。

### 43.8 信息隔离

必须专门测试：

```text
PlayerView 不包含其他玩家手牌
```

这是 LLM 安全边界。

### 43.9 LLM Client

使用 `httptest.Server`。

覆盖：

- 正常 JSON。
- Markdown code fence JSON。
- HTTP 500。
- timeout。
- malformed JSON。
- 不存在的 move ID。
- retry。
- fallback。

不得在测试里调用真实付费 API。

---

## 44. 可测试性要求

Game Engine 需要支持：

- 注入 deck。
- 注入随机 seed。
- 注入 AI。
- 注入 clock/timer（如需要）。
- LLM Client 通过 interface 注入。

不能把：

```text
random
HTTP
TUI
game state
```

全部写进一个 `main.go`。

---

## 45. 性能要求

斗地主规模很小，不需要极端优化。

目标：

- 本地 AI 决策常规局面 < 100ms。
- TUI 输入无明显延迟。
- Legal Moves 不造成 UI 卡顿。
- LLM 网络时间不计入本地性能目标。

如果首出合法动作枚举过大，应通过算法和去重控制，而不是生成海量重复 permutation。

---

## 46. Move 去重

同一点数组成的等价动作只保留一个。

例如玩家有不同花色：

```text
♠3 ♥3 ♦3 ♣3
```

当生成：

```text
33
```

不应该因为不同花色组合生成 6 个 UI 候选。

UI 和 AI 按点数组合看待动作。

Engine 执行时再从手牌中选择具体实体牌移除。

这是控制合法动作数量的重要要求。

---

## 47. 手牌排序

默认显示顺序：

```text
3 4 5 6 7 8 9 10 J Q K A 2 SJ BJ
```

相同点数相邻。

例如：

```text
3 3 4 5 5 5 9 10 J Q K A 2 SJ BJ
```

不按花色拆散。

---

## 48. 游戏流程状态机

建议：

```text
Boot
  ↓
LoadConfig
  ↓
MainMenu
  ↓
Deal
  ↓
Bidding
  ↓
LandlordConfirmed
  ↓
Playing
  ↓
RoundFinished
  ↓
Result
  ↓
PlayAgain / MainMenu / Quit
```

### 48.1 Main Menu

MVP 菜单：

```text
Terminal 斗地主

> Start Game
  Settings Info
  Quit
```

`Settings Info` 只显示当前配置摘要，不要求在游戏内修改 API Key。

例如：

```text
AI-1: Local / Normal
AI-2: LLM / provider_b / model-x
Counter: On
Theme: work
```

API Key 只能显示：

```text
Configured
```

绝不能显示完整值。

---

## 49. 对局结束界面

示例：

```text
┌───────────────────────────────┐
│ Round Finished                │
│                               │
│ Farmers Win                   │
│                               │
│ Bid:        2                 │
│ Bombs:      1                 │
│ Multiplier: x2                │
│ Score:      4                 │
│                               │
│ You:        +4                │
│ Worker-01:  +4                │
│ Worker-02:  -8                │
│                               │
│ [Enter] Play Again            │
│ [Esc]   Main Menu             │
└───────────────────────────────┘
```

---

## 50. MVP 本地统计

建议在内存中记录当前启动周期：

- 总局数。
- Human 胜局。
- Human 地主胜率。
- Human 农民胜率。
- 炸弹次数。

MVP **不要求持久化统计**。

如实现持久化，放到后续版本。

---

## 51. 安全与隐私

### 51.1 API Key

- 不打印完整 API Key。
- 不写入日志。
- `config.toml` gitignore。
- `config.example.toml` 不包含真实 Key。

### 51.2 LLM 数据

调用外部 LLM 时会把：

- AI 自己的手牌。
- 公开游戏状态。
- 最近出牌历史。
- 合法候选。

发送给对应 Provider。

不发送：

- Human 的隐藏手牌给不该看到它的 AI。
- 另一 AI 的隐藏手牌。
- 配置文件完整内容。
- 其他 Provider 的 API Key。

---

## 52. README 必须包含

MVP README 至少：

1. 项目介绍。
2. 截图或终端示例。
3. 支持平台。
4. 快速开始。
5. 配置文件说明。
6. Local AI 用法。
7. 两个 LLM AI 独立 Provider 示例。
8. API Key 安全提醒。
9. 操作键位。
10. 斗地主规则说明入口。
11. 构建命令。
12. 测试命令。
13. Release 下载说明。
14. License。

---

## 53. 推荐实现顺序

### Phase 1 — 纯规则引擎

实现：

- Card。
- Deck。
- Hand detection。
- Compare。
- Legal Moves。
- Engine。
- Bidding。
- Scoring。

此阶段不做 TUI，不做 LLM。

验收：

```text
go test ./internal/game/...
```

规则测试全部通过。

### Phase 2 — Local AI

实现：

- Agent interface。
- Easy。
- Normal。
- Hard 基础版本。
- PlayerView。

验收：

让 3 个 Local AI 自动跑大量固定 seed 对局：

- 不出现非法动作。
- 不死循环。
- 每局最终必然结束。

### Phase 3 — 基础 TUI

实现：

- 主菜单。
- 叫地主。
- 游戏面板。
- 候选动作。
- 键盘操作。
- 记牌器。
- 历史。
- 结果页。

此时：

```text
Human vs Local vs Local
```

必须完整可玩。

### Phase 4 — LLM

实现：

- Config。
- Provider。
- OpenAI-compatible client。
- Prompt。
- JSON parse。
- Retry。
- Fallback。

支持：

```text
Human vs LLM-A vs Local
Human vs LLM-A vs LLM-B
```

### Phase 5 — 跨平台与 Release

实现：

- GitHub Actions。
- Windows/macOS/Linux build。
- README。
- config.example.toml。
- Release assets。

---

## 54. MVP 验收标准

以下全部满足才算 MVP 完成。

### 游戏

- [ ] 可以启动新游戏。
- [ ] 每局 54 张牌正确发放。
- [ ] 支持 0/1/2/3 叫地主。
- [ ] 三人全不叫自动重新发牌。
- [ ] 地主拿 3 张底牌。
- [ ] 支持全部 MVP 牌型。
- [ ] 所有牌型比较正确。
- [ ] 炸弹与王炸正确。
- [ ] 两家 PASS 后正确重置轮次。
- [ ] 任一玩家出完牌立即结束。
- [ ] 春天/反春天和倍率正确。

### UI

- [ ] Windows/macOS/Linux 终端可运行。
- [ ] 不要求输入牌面字符串。
- [ ] ↑↓/jk 可选择动作。
- [ ] Enter 出牌。
- [ ] Space PASS。
- [ ] R 记牌器。
- [ ] H 历史。
- [ ] Q 退出。
- [ ] 终端过小时有明确提示。
- [ ] 所有 10 都显示为 `10`，从不显示 `T`。

### Local AI

- [ ] 不配置 API 即可玩。
- [ ] Easy 可用。
- [ ] Normal 可用。
- [ ] Hard 可用。
- [ ] AI 永远只能执行合法 Move。
- [ ] 农民 AI 至少具有基本队友让牌策略。

### LLM

- [ ] 两个 AI 可以独立使用不同 Provider。
- [ ] Provider 分别配置 base_url。
- [ ] Provider 分别配置 api_key。
- [ ] Provider 分别配置 model。
- [ ] 使用 OpenAI-compatible `/chat/completions`。
- [ ] LLM 只返回合法动作 ID。
- [ ] 非法返回会被拒绝。
- [ ] 有 timeout。
- [ ] 有 retry。
- [ ] 有 Local AI fallback。
- [ ] AI 看不到不应看到的隐藏牌。

### 配置与安全

- [ ] `config.toml` 被 `.gitignore` 忽略。
- [ ] 仓库包含 `config.example.toml`。
- [ ] 允许明文 API Key。
- [ ] 日志不打印完整 Key。
- [ ] Local-only 模式无需 Key。

### 工程

- [ ] `go test ./...` 通过。
- [ ] `go vet ./...` 通过。
- [ ] `go build ./...` 通过。
- [ ] 关键规则有单元测试。
- [ ] LLM 使用 `httptest.Server` 测试。
- [ ] 无真实 API 集成测试依赖。
- [ ] Game Engine 与 TUI 解耦。
- [ ] Game Engine 与 HTTP 解耦。

---

## 55. 建议的 Codex 实现约束

将本节直接作为 Codex 的工程要求。

### 55.1 必须

1. 先实现并测试规则层，再做 TUI。
2. 不允许把整个项目写在单个文件。
3. 所有牌面表示统一由 `Rank.String()` 处理。
4. `Rank.String()` 对 10 必须返回 `"10"`。
5. 不允许出现 UI 专用 `"T"`。
6. Legal Move Generator 是唯一合法候选来源。
7. Local AI 和 LLM AI 都只返回 Move ID。
8. Engine 执行前再次验证动作。
9. LLM 使用 PlayerView，禁止传完整 GameState。
10. 网络请求必须支持 context cancellation。
11. HTTP 请求不得阻塞 Bubble Tea 主循环。
12. LLM 异常默认 fallback Local AI。
13. config.toml 不得提交。
14. 不在日志中记录完整 API Key。
15. 所有规则核心函数必须有单元测试。

### 55.2 不要

1. 不要让 LLM 自己自由输出牌字符串并直接执行。
2. 不要把对手隐藏牌发送给 LLM。
3. 不要使用 `fmt.Scanln()` 作为主要游戏操作方式。
4. 不要要求用户输入类似 `play 3 3`。
5. 不要将牌型判断逻辑复制到 TUI 或 AI 层。
6. 不要让 Local AI 直接修改 GameState。
7. 不要因为 API 请求失败而让整局游戏永久卡死。
8. 不要把真实 API Key 放到代码、测试或 example config。
9. 不要依赖 CGO。
10. 不要使用 `T` 代表 10。

---

## 56. `config.example.toml` 建议内容

```toml
[general]
language = "zh-CN"
theme = "work"
show_card_counter = true
show_history_panel = true
ai_delay_ms = 350

[game]
local_ai_difficulty = "normal"
llm_fallback = true
max_llm_legal_moves = 80

[player]
name = "You"

# type: "local" or "llm"

[ai1]
name = "Worker-01"
type = "local"
difficulty = "normal"
# provider = "provider_a"

[ai2]
name = "Worker-02"
type = "local"
difficulty = "normal"
# provider = "provider_b"

[providers.provider_a]
base_url = "https://api.openai.com/v1"
api_key = ""
model = ""
timeout_seconds = 15

[providers.provider_b]
base_url = "https://example.com/v1"
api_key = ""
model = ""
timeout_seconds = 15
```

---

## 57. 第一版完成后的建议路线

不属于 MVP，但架构应预留。

### v0.2

- 游戏内切换主题。
- 配置热加载。
- 更智能 Hard AI。
- 持久化统计。
- 对局 replay。
- 自定义快捷键。
- 多语言。
- Ollama / LM Studio preset。
- 更多 OpenAI-compatible API 兼容策略。

### v0.3

AI Arena：

```text
LLM A vs LLM B vs Local
```

支持无人值守跑 N 局。

统计：

- 胜率。
- 地主胜率。
- 农民胜率。
- 平均决策耗时。
- API 错误率。
- fallback 次数。
- 平均 token 使用量（Provider 返回 usage 时）。

注意斗地主是三人阵营游戏，因此 Arena 排名需要区分地主和农民角色，不应直接只看总胜率。

---

## 58. MVP 最终定义

`Terminal 斗地主 v0.1` 的一句话定义：

> 一个使用 Go 实现、可在 Windows/macOS/Linux 终端直接运行的三人斗地主 TUI 游戏，支持完整基础斗地主规则、简单键盘操作、实时记牌器、本地 AI，以及两个可独立配置不同 OpenAI-compatible Provider 的 LLM AI；游戏规则完全由本地引擎控制，大模型只能从合法动作中做选择。

MVP 核心技术原则：

```text
Rules Engine owns truth.
UI only presents state.
AI only chooses legal action IDs.
LLM sees only legitimate player-visible information.
Network failure must not break the game.
10 is always displayed as "10", never "T".
```

---

# Appendix A — 核心接口草案

以下仅用于约束结构，不要求逐字实现。

```go
type Agent interface {
    ChooseBid(
        ctx context.Context,
        view player.PlayerView,
        legalBids []int,
    ) (int, error)

    ChooseMove(
        ctx context.Context,
        view player.PlayerView,
        legalMoves []game.Move,
    ) (int, error)
}
```

```go
type Engine interface {
    State() game.GameState
    StartRound() error
    LegalBids(seat int) []int
    ApplyBid(seat int, bid int) error
    LegalMoves(seat int) []game.Move
    ApplyMove(seat int, moveID int) error
}
```

```go
type LLMProvider struct {
    Name           string
    BaseURL        string
    APIKey         string
    Model          string
    TimeoutSeconds int
}
```

```go
type PlayerView struct {
    Seat          int
    Role          Role
    OwnCards      []Card
    OtherCounts   map[int]int
    LandlordSeat  int
    BottomPublic  []Card
    PlayedCards   []PublicAction
    LastMove      *PublicMove
    BidScore      int
    Multiplier    int
}
```

---

# Appendix B — 规则层不可破坏的不变量

任何时候都应满足：

1. 三名玩家手牌 + 已出牌 == 54 张实体牌。
2. 同一实体牌不能同时存在于两个位置。
3. 游戏进行时只有一个 CurrentSeat。
4. 未结束时当前玩家必定至少有一个合法动作。
5. 首出玩家不能 PASS。
6. 跟牌玩家永远有 PASS，因此至少有一个合法动作。
7. ApplyMove 后手牌数量严格减少，除非是 PASS。
8. 玩家手牌变为 0 时立即 Finished。
9. LLM 不能改变任何不变量。
10. TUI 不能改变任何不变量。

---

# Appendix C — Codex 开工指令建议

可以将本文件放到仓库根目录，例如：

```text
MVP.md
```

然后给 Codex：

```text
请严格按照 MVP.md 实现 Terminal 斗地主。

先完成 Phase 1 的 game engine 和全部单元测试，不要先做 TUI。
完成 Phase 1 后再按文档 Phase 2、3、4、5 的顺序实现。

任何存在地区差异的斗地主规则，以 MVP.md 中明确写出的规则为准。
所有牌面中的 10 必须显示为 "10"，禁止使用 "T"。

LLM 永远只能从本地规则引擎生成的合法 Move ID 中选择，
不得让模型自由构造出牌，不得向模型泄露其他玩家隐藏手牌。

每完成一个 Phase：
1. 运行 gofmt。
2. 运行 go vet ./...
3. 运行 go test ./...
4. 修复所有失败后再进入下一 Phase。
```
