package llm

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/syjsion/Terminal_DDZ/internal/ai"
	"github.com/syjsion/Terminal_DDZ/internal/config"
	"github.com/syjsion/Terminal_DDZ/internal/game"
	"github.com/syjsion/Terminal_DDZ/internal/player"
)

type stubAgent struct{ id int }

type blockingDoer struct{}

type countingDoer struct{ calls atomic.Int32 }

func (blockingDoer) Do(req *http.Request) (*http.Response, error) {
	<-req.Context().Done()
	return nil, req.Context().Err()
}

func (d *countingDoer) Do(*http.Request) (*http.Response, error) {
	d.calls.Add(1)
	return nil, errors.New("unexpected HTTP request")
}

func (s stubAgent) ChooseBid(context.Context, player.PlayerView, []int) (int, error) {
	return s.id, nil
}
func (s stubAgent) ChooseMove(context.Context, player.PlayerView, []game.Move) (int, error) {
	return s.id, nil
}

func providerFor(url string) config.ProviderConfig {
	return config.ProviderConfig{BaseURL: url + "/v1/", APIKey: "secret-test-key", Model: "test-model", TimeoutSeconds: 1}
}

func response(content string) []byte {
	data, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"role": "assistant", "content": content}}}})
	return data
}

func TestChooseMoveRetriesCodeFenceAndUsesAuth(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer secret-test-key" {
			t.Errorf("missing auth")
		}
		if calls.Add(1) == 1 {
			_, _ = w.Write(response(`{"move":999}`))
			return
		}
		_, _ = w.Write(response("```json\n{\"move\": 7}\n```"))
	}))
	defer server.Close()
	agent := NewAgent("test", providerFor(server.URL), 80, nil, false, server.Client())
	legal := []game.Move{{ID: 7, Type: game.Single, Cards: []game.Card{{Rank: game.Rank3}}}}
	id, err := agent.ChooseMove(context.Background(), player.PlayerView{Seat: 1, OwnCards: legal[0].Cards, OtherCounts: map[int]int{0: 17, 2: 17}}, legal)
	if err != nil || id != 7 || calls.Load() != 2 {
		t.Fatalf("ChooseMove = %d, %v, calls=%d", id, err, calls.Load())
	}
}

func TestChooseMoveOnlyPassSkipsHTTP(t *testing.T) {
	doer := &countingDoer{}
	agent := NewAgent("test", providerFor("https://example.invalid"), 80, nil, false, doer)
	pass := game.PassMove()
	pass.ID = 17
	id, err := agent.ChooseMove(context.Background(), player.PlayerView{}, []game.Move{pass})
	if err != nil || id != 17 || doer.calls.Load() != 0 {
		t.Fatalf("ChooseMove = %d, %v, calls=%d", id, err, doer.calls.Load())
	}
}

func TestChooseMoveOnlyPassHonorsCancellation(t *testing.T) {
	doer := &countingDoer{}
	agent := NewAgent("test", providerFor("https://example.invalid"), 80, nil, false, doer)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	pass := game.PassMove()
	pass.ID = 17
	_, err := agent.ChooseMove(ctx, player.PlayerView{}, []game.Move{pass})
	if !errors.Is(err, context.Canceled) || doer.calls.Load() != 0 {
		t.Fatalf("ChooseMove error = %v, calls=%d", err, doer.calls.Load())
	}
}

func TestChooseMovePassAndBombStillCallsHTTP(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		_, _ = w.Write(response(`{"move":9}`))
	}))
	defer server.Close()
	agent := NewAgent("test", providerFor(server.URL), 80, nil, false, server.Client())
	pass := game.PassMove()
	pass.ID = 0
	bomb := game.Move{ID: 9, Type: game.Bomb, MainRank: game.Rank3, Cards: []game.Card{{Rank: game.Rank3}, {Rank: game.Rank3}, {Rank: game.Rank3}, {Rank: game.Rank3}}}
	id, err := agent.ChooseMove(context.Background(), player.PlayerView{Seat: 1, Role: game.RoleFarmer, LandlordSeat: 0, OwnCards: bomb.Cards, OtherCounts: map[int]int{0: 1, 2: 2}}, []game.Move{pass, bomb})
	if err != nil || id != 9 || calls.Load() != 1 {
		t.Fatalf("ChooseMove = %d, %v, calls=%d", id, err, calls.Load())
	}
}

func TestMalformedResponseFallsBack(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(response("not json"))
	}))
	defer server.Close()
	agent := NewAgent("broken", providerFor(server.URL), 80, stubAgent{id: 4}, true, server.Client())
	id, err := agent.ChooseBid(context.Background(), player.PlayerView{}, []int{0, 4})
	var fallbackErr *ai.FallbackError
	if id != 4 || !errors.As(err, &fallbackErr) {
		t.Fatalf("fallback = %d, %v", id, err)
	}
	if strings.Contains(err.Error(), "secret-test-key") {
		t.Fatal("error leaked API key")
	}
}

func TestHTTPErrorWithoutFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	agent := NewAgent("broken", providerFor(server.URL), 80, nil, false, server.Client())
	if _, err := agent.ChooseBid(context.Background(), player.PlayerView{}, []int{0}); err == nil || !strings.Contains(err.Error(), "500") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEmptyChoicesAndMalformedHTTPJSON(t *testing.T) {
	responses := [][]byte{[]byte(`{"choices":[]}`), []byte(`not-json`)}
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		index := int(calls.Add(1)) - 1
		_, _ = w.Write(responses[index])
	}))
	defer server.Close()
	agent := NewAgent("broken", providerFor(server.URL), 80, nil, false, server.Client())
	if _, err := agent.ChooseBid(context.Background(), player.PlayerView{}, []int{0}); err == nil {
		t.Fatal("expected response validation error")
	}
}

func TestProviderTimeout(t *testing.T) {
	agent := NewAgent("slow", config.ProviderConfig{BaseURL: "https://example.invalid/v1", APIKey: "x", Model: "x", TimeoutSeconds: 1}, 80, nil, false, blockingDoer{})
	started := time.Now()
	if _, err := agent.ChooseBid(context.Background(), player.PlayerView{}, []int{0}); err == nil {
		t.Fatal("expected timeout")
	}
	if elapsed := time.Since(started); elapsed < time.Second || elapsed > 3*time.Second {
		t.Fatalf("unexpected timeout duration: %v", elapsed)
	}
}

func TestCancellationStopsRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(100 * time.Millisecond)
		_, _ = w.Write(response(`{"bid":0}`))
	}))
	defer server.Close()
	agent := NewAgent("slow", providerFor(server.URL), 80, nil, false, server.Client())
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := agent.ChooseBid(ctx, player.PlayerView{}, []int{0}); err == nil {
		t.Fatal("expected cancellation")
	}
}

func TestPrunePreservesCriticalMoves(t *testing.T) {
	var moves []game.Move
	for i := 0; i < 20; i++ {
		moves = append(moves, game.Move{ID: i, Type: game.Single, MainRank: game.Rank3, Cards: []game.Card{{Rank: game.Rank3}}})
	}
	moves[0] = game.PassMove()
	moves[0].ID = 0
	moves[18] = game.Move{ID: 18, Type: game.Bomb, MainRank: game.RankA, Cards: []game.Card{{Rank: game.RankA}, {Rank: game.RankA}, {Rank: game.RankA}, {Rank: game.RankA}}}
	moves[19] = game.Move{ID: 19, Type: game.Rocket, Cards: []game.Card{{Rank: game.RankSJ}, {Rank: game.RankBJ}}}
	pruned := PruneLegalMoves(moves, 5, 10)
	ids := map[int]bool{}
	for _, move := range pruned {
		ids[move.ID] = true
	}
	if len(pruned) != 5 || !ids[0] || !ids[18] || !ids[19] {
		t.Fatalf("critical moves missing: %+v", ids)
	}
}

func TestMovePromptNeverUsesTRank(t *testing.T) {
	view := player.PlayerView{Seat: 1, OwnCards: []game.Card{{Rank: game.Rank10}}, OtherCounts: map[int]int{0: 1, 2: 1}, LandlordSeat: 0}
	prompt := movePrompt(view, []game.Move{{ID: 1, Type: game.Single, Cards: view.OwnCards}})
	if strings.Contains(prompt, " T ") || !strings.Contains(prompt, "10") {
		t.Fatalf("invalid rank rendering: %s", prompt)
	}
}

func TestMovePromptIncludesPublicStrategyContext(t *testing.T) {
	view := player.PlayerView{
		Seat:         1,
		Role:         game.RoleFarmer,
		LandlordSeat: 0,
		Multiplier:   4,
		OwnCards:     []game.Card{{Rank: game.Rank3}, {Rank: game.Rank3}},
		OtherCounts:  map[int]int{0: 1, 2: 2},
		BottomPublic: []game.Card{{Rank: game.Rank2}},
		LastMove: &player.PublicMove{Seat: 2, Move: game.Move{
			Type: game.Single, MainRank: game.Rank4, Cards: []game.Card{{Rank: game.Rank4}},
		}},
		PlayedCards: []game.ActionRecord{{Kind: game.ActionPlay, Seat: 0, Move: game.Move{Type: game.Single, Cards: []game.Card{{Rank: game.Rank5}}}}},
	}
	legal := []game.Move{{ID: 7, Type: game.Pair, MainRank: game.Rank3, Cards: view.OwnCards}}
	prompt := movePrompt(view, legal)
	for _, want := range []string{
		"Current target: Seat 2 (teammate)",
		"Seat 0 (opponent/landlord)=1",
		"Unplayed cards outside your hand by rank:",
		"type=PAIR",
		"cards_used=2",
		"cards_remaining=0",
		"finishes_hand=yes",
		"usually PASS when your teammate controls",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q:\n%s", want, prompt)
		}
	}
}
