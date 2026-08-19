package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/syjsion/Terminal_DDZ/internal/ai"
	"github.com/syjsion/Terminal_DDZ/internal/config"
	"github.com/syjsion/Terminal_DDZ/internal/game"
	"github.com/syjsion/Terminal_DDZ/internal/player"
)

type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type Agent struct {
	providerName string
	provider     config.ProviderConfig
	maxMoves     int
	fallback     ai.Agent
	fallbackOn   bool
	http         HTTPDoer
}

func NewAgent(name string, provider config.ProviderConfig, maxMoves int, fallback ai.Agent, fallbackOn bool, doer HTTPDoer) *Agent {
	if doer == nil {
		doer = http.DefaultClient
	}
	if maxMoves < 1 {
		maxMoves = 80
	}
	return &Agent{providerName: name, provider: provider, maxMoves: maxMoves, fallback: fallback, fallbackOn: fallbackOn, http: doer}
}

func (a *Agent) ChooseBid(ctx context.Context, view player.PlayerView, legalBids []int) (int, error) {
	prompt := bidPrompt(view, legalBids)
	valid := make(map[int]bool, len(legalBids))
	for _, bid := range legalBids {
		valid[bid] = true
	}
	selected, err := a.choose(ctx, prompt, "bid", valid)
	if err == nil {
		return selected, nil
	}
	if a.fallbackOn && a.fallback != nil {
		bid, fallbackErr := a.fallback.ChooseBid(ctx, view, legalBids)
		if fallbackErr != nil {
			return 0, errors.Join(err, fallbackErr)
		}
		return bid, &ai.FallbackError{Provider: a.providerName, Cause: err}
	}
	return 0, err
}

func (a *Agent) ChooseMove(ctx context.Context, view player.PlayerView, legalMoves []game.Move) (int, error) {
	candidates := PruneLegalMoves(legalMoves, a.maxMoves, len(view.OwnCards))
	prompt := movePrompt(view, candidates)
	valid := make(map[int]bool, len(candidates))
	for _, move := range candidates {
		valid[move.ID] = true
	}
	selected, err := a.choose(ctx, prompt, "move", valid)
	if err == nil {
		return selected, nil
	}
	if a.fallbackOn && a.fallback != nil {
		moveID, fallbackErr := a.fallback.ChooseMove(ctx, view, legalMoves)
		if fallbackErr != nil {
			return 0, errors.Join(err, fallbackErr)
		}
		return moveID, &ai.FallbackError{Provider: a.providerName, Cause: err}
	}
	return 0, err
}

func (a *Agent) choose(ctx context.Context, prompt, field string, valid map[int]bool) (int, error) {
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		requestPrompt := prompt
		if attempt == 1 {
			requestPrompt += "\n\nYour previous response was invalid. Return JSON only and choose one of the provided IDs."
		}
		content, err := a.request(ctx, requestPrompt)
		if err == nil {
			selected, parseErr := parseSelection(content, field)
			if parseErr == nil && valid[selected] {
				return selected, nil
			}
			if parseErr != nil {
				err = parseErr
			} else {
				err = fmt.Errorf("返回的 %s ID %d 不在合法集合中", field, selected)
			}
		}
		lastErr = err
		if ctx.Err() != nil {
			return 0, ctx.Err()
		}
	}
	return 0, fmt.Errorf("LLM 两次决策均失败: %w", lastErr)
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}

func (a *Agent) request(parent context.Context, prompt string) (string, error) {
	timeout := time.Duration(a.provider.TimeoutSeconds) * time.Second
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	payload := chatRequest{
		Model: a.provider.Model,
		Messages: []chatMessage{
			{Role: "system", Content: "You are a Dou Dizhu AI. Select only from legal IDs. Never invent cards. Return JSON only."},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.2,
		MaxTokens:   100,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	endpoint := strings.TrimRight(a.provider.BaseURL, "/")
	if !strings.HasSuffix(endpoint, "/chat/completions") {
		endpoint += "/chat/completions"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.provider.APIKey)
	resp, err := a.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP 请求失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("HTTP 状态 %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}
	var decoded chatResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		return "", fmt.Errorf("响应 JSON 无效: %w", err)
	}
	if len(decoded.Choices) == 0 || strings.TrimSpace(decoded.Choices[0].Message.Content) == "" {
		return "", errors.New("响应不包含 choices 内容")
	}
	return decoded.Choices[0].Message.Content, nil
}

func parseSelection(content, field string) (int, error) {
	clean := strings.TrimSpace(content)
	if strings.HasPrefix(clean, "```") {
		if nl := strings.IndexByte(clean, '\n'); nl >= 0 {
			clean = clean[nl+1:]
		}
		clean = strings.TrimSuffix(strings.TrimSpace(clean), "```")
	}
	if first, last := strings.IndexByte(clean, '{'), strings.LastIndexByte(clean, '}'); first >= 0 && last >= first {
		clean = clean[first : last+1]
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(clean), &raw); err != nil {
		return 0, fmt.Errorf("模型输出不是有效 JSON: %w", err)
	}
	value, ok := raw[field]
	if !ok {
		return 0, fmt.Errorf("模型输出缺少 %q", field)
	}
	var selected int
	if err := json.Unmarshal(value, &selected); err != nil {
		return 0, fmt.Errorf("%s 必须是整数", field)
	}
	return selected, nil
}

func PruneLegalMoves(moves []game.Move, max, ownCount int) []game.Move {
	if max <= 0 || len(moves) <= max {
		return append([]game.Move(nil), moves...)
	}
	selected := make(map[int]bool, max)
	add := func(move game.Move) {
		if len(selected) < max {
			selected[move.ID] = true
		}
	}
	for _, move := range moves {
		if move.IsPass || len(move.Cards) == ownCount || move.Type == game.Bomb || move.Type == game.Rocket {
			add(move)
		}
	}
	seenType := map[game.HandType]bool{}
	for _, move := range moves {
		if !move.IsPass && !seenType[move.Type] {
			add(move)
			seenType[move.Type] = true
		}
	}
	type scoredMove struct {
		move  game.Move
		score int
	}
	var remaining []scoredMove
	for _, move := range moves {
		if !selected[move.ID] {
			remaining = append(remaining, scoredMove{move: move, score: len(move.Cards)*100 - int(move.MainRank)})
		}
	}
	sort.SliceStable(remaining, func(i, j int) bool { return remaining[i].score > remaining[j].score })
	for _, item := range remaining {
		add(item.move)
	}
	result := make([]game.Move, 0, len(selected))
	for _, move := range moves {
		if selected[move.ID] {
			result = append(result, move)
		}
	}
	return result
}
