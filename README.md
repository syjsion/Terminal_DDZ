# Terminal 斗地主

一个使用 Go 编写、完全运行在终端中的三人斗地主游戏。它包含完整的 MVP 规则引擎、三档本地 AI、实时记牌器，也可以把两名电脑玩家分别连接到不同的 OpenAI-compatible Chat Completions 服务。

规则引擎始终拥有最终决定权：AI 只能从本地生成的合法动作 ID 中选择，无法构造非法牌，也看不到其他玩家的隐藏手牌。

## 界面示例

```text
Terminal 斗地主  第 12 局  倍率 x4  轮到：Seat 0 你
底牌：4 A BJ
记牌器：3:1 4:2 5:0 6:3 7:2 8:1 9:0 10:2 ...
Seat 1 Worker-01 [农民] 9 张    |    Seat 2 Worker-02 [地主] 4 张
----------------------------------------------------------------
当前目标：9 9 · Worker-02
出牌历史：
#04 Worker-01 7
#05 Worker-02 9 9
#06 Worker-01 PASS
#07 你        10 10
#08 Worker-02 PASS
#09 Worker-01 PASS
----------------------------------------------------------------
推荐候选（Tab 切换手选）：
> [1] 10 10（对子） [推荐]
  [2] PASS（过牌）
----------------------------------------------------------------
你的身份：农民 · 手牌 [11]
3 3 4 5 6 7 8 9 10 J A
```

所有牌面始终使用 `10`，不会使用 `T` 代替。

首出候选按实战用途排序，三带一、顺子、连对和飞机等组合会优先于零散单牌。按 `Tab` 可以切换到手动选牌，不需要在长候选列表中查找；提交的牌仍会由规则引擎验证。

## 支持平台

- Windows 10/11：CMD、PowerShell、Windows Terminal，amd64/arm64
- macOS：Intel amd64、Apple Silicon arm64
- Linux：amd64/arm64

程序不使用 CGO，发布产物是单个可执行文件。建议终端尺寸至少为 80×24。

## 快速开始

需要 Go 1.25 或更高版本：

```bash
go run ./cmd/terminal-ddz
```

不创建配置文件时，默认直接运行 `Human vs Local Normal vs Local Normal`，不需要 API Key。

常用命令：

```text
terminal-ddz
terminal-ddz --config ./my-config.toml
terminal-ddz --debug
terminal-ddz --seed 1234
terminal-ddz --version
terminal-ddz --help
```

## 配置

复制 `config.example.toml` 为当前目录下的 `config.toml`，然后按需修改。也可以通过 `--config` 指定其他路径。

本地 AI 支持 `easy`、`normal`、`hard`：

```toml
[ai1]
name = "Worker-01"
type = "local"
difficulty = "hard"
```

### 两个独立的 LLM Provider

```toml
[game]
llm_fallback = true
max_llm_legal_moves = 80

[ai1]
name = "Worker-01"
type = "llm"
provider = "provider_a"

[ai2]
name = "Worker-02"
type = "llm"
provider = "provider_b"

[providers.provider_a]
base_url = "https://api.openai.com/v1"
api_key_env = "OPENAI_API_KEY"
model = "your-model-a"
timeout_seconds = 15

[providers.provider_b]
base_url = "https://example.com/v1"
api_key_env = "TERMINAL_DDZ_PROVIDER_B_KEY"
model = "your-model-b"
timeout_seconds = 20
```

启动前设置对应环境变量：

```bash
export OPENAI_API_KEY="your-provider-a-key"
export TERMINAL_DDZ_PROVIDER_B_KEY="your-provider-b-key"
```

PowerShell 使用 `$env:OPENAI_API_KEY="your-provider-a-key"`。也可以改用 `api_key` 直接配置，但 `api_key` 与 `api_key_env` 只能二选一。

程序调用 `POST {base_url}/chat/completions`。模型输出无效时会重试一次；默认仍失败就由 Local Normal 完成本次决策，下一次决策会重新请求 LLM。错误面板中手动切换本地 AI 也只影响当前局，新局会恢复配置的 LLM。

> `config.toml` 已被 Git 忽略。若选择使用 `api_key`，文件会包含明文凭据，请勿提交或分享；推荐使用 `api_key_env`。macOS/Linux 用户仍可执行 `chmod 600 config.toml` 限制配置文件权限。

## 操作键位

候选模式：

| 键位 | 操作 |
| --- | --- |
| `↑` / `k` | 上一个候选 |
| `↓` / `j` | 下一个候选 |
| `Enter` | 确认候选 |
| `Space` / `P` | PASS（仅跟牌时） |
| `1`–`9` | 快速选择候选 |
| `Tab` | 切换到手动选牌 |

手选模式：

| 键位 | 操作 |
| --- | --- |
| `←` / `→` | 移动手牌光标 |
| `Space` | 选择或取消当前牌 |
| `Enter` | 提交所选牌 |
| `Backspace` | 清空选择 |
| `P` | PASS（仅跟牌时） |
| `Tab` | 返回推荐候选 |

通用键位：

| 键位 | 操作 |
| --- | --- |
| `R` | 打开/关闭记牌器 |
| `H` | 打开/关闭历史 |
| `?` | 帮助 |
| `Q` | 请求退出 |
| `Esc` | 关闭弹层；游戏中确认放弃本局并返回主菜单 |

## 规则

支持单张、对子、三张、三带一、三带二、顺子、连对、飞机及两种翅膀、四带二单、四带二对、炸弹和王炸。叫地主使用 0/1/2/3 分制，计分包含炸弹、王炸、春天和反春天。

完整且具有优先级的规则定义见 [Terminal_DDZ_MVP.md](./Terminal_DDZ_MVP.md)。

## 构建与测试

```bash
gofmt -w ./cmd ./internal
go vet ./...
go test ./...
go build -o terminal-ddz ./cmd/terminal-ddz
```

项目测试不会访问真实 LLM 服务；HTTP 行为由 `httptest.Server` 模拟。

## Release 下载

带 `v` 前缀的 tag 会触发 GitHub Actions，生成以下单文件产物：

```text
terminal-ddz_windows_amd64.exe
terminal-ddz_windows_arm64.exe
terminal-ddz_darwin_amd64
terminal-ddz_darwin_arm64
terminal-ddz_linux_amd64
terminal-ddz_linux_arm64
SHA256SUMS
```

发布前会重新执行格式检查、`go vet` 和完整测试，并在 CI 中验证上述六个目标平台均可交叉编译。下载后可使用 `sha256sum -c SHA256SUMS`（Linux）或 `shasum -a 256 -c SHA256SUMS`（macOS）校验文件完整性。

## 隐私说明

使用 LLM AI 时，只会向对应 Provider 发送该 AI 自己的手牌、公开底牌、公开历史、剩余牌数和本地生成的合法候选。不会发送其他玩家的隐藏手牌、其他 Provider 的凭据或完整配置文件。

## License

[MIT](./LICENSE)
