package main

import (
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"

	"golang.org/x/text/unicode/norm"
)

const (
	type1ThinkTime  = 30 * time.Second
	type1AnswerTime = 30 * time.Second
	type2StartWait  = 5 * time.Second
	type2QTime      = 10 * time.Second
	type2ResultWait = 0 * time.Second
	type2NextWait   = 3 * time.Second
)

type Game struct {
	room     *Room
	quizType string
	state    string
	mu       sync.Mutex

	// ── Type 1 fields ─────────────────────────────
	t1Rounds       int
	t1CurrentRound int
	t1QOrder       []string // player IDs in questioner order
	t1QuestionerID string
	t1Question     Type1Question
	t1QuestRanking []int            // ranking[i] = rank of option i (1-4)
	t1Guesses      map[string][]int // clientID -> ranking
	t1ReadyPlayers map[string]bool

	// ── Type 2 fields ─────────────────────────────
	t2Questions   []PersonQuestion
	t2CurrentIdx  int
	t2Scores      map[string]int
	t2FailPlayers map[string]bool
	t2RoundDone   bool

	timer *time.Timer
}

func newGame(room *Room) *Game {
	return &Game{
		room:           room,
		t1Guesses:      make(map[string][]int),
		t1ReadyPlayers: make(map[string]bool),
		t2Scores:       make(map[string]int),
		t2FailPlayers:  make(map[string]bool),
	}
}

func (g *Game) start() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.room.broadcastJSON("game_started", nil)
		g.initType1()
}

// ═══════════════════════════════════════════════
// TYPE 1
// ═══════════════════════════════════════════════

func (g *Game) initType1() {
	g.room.mu.RLock()
	ids := make([]string, 0, len(g.room.clients))
	for id := range g.room.clients {
		ids = append(ids, id)
	}
	g.room.mu.RUnlock()

	rand.Shuffle(len(ids), func(i, j int) { ids[i], ids[j] = ids[j], ids[i] })
	g.t1QOrder = ids
	g.t1Rounds = len(ids)
	g.t1CurrentRound = 0

	g.nextType1Round()
}

func (g *Game) nextType1Round() {
	if g.t1CurrentRound >= g.t1Rounds {
		g.initType2()
		return
	}

	g.t1QuestionerID = g.t1QOrder[g.t1CurrentRound]
	g.t1CurrentRound++
	g.t1Guesses = make(map[string][]int)
	g.t1QuestRanking = nil
	g.t1Question = randomType1Question()
	g.state = "type1_thinking"

	questioner, ok := g.room.getClient(g.t1QuestionerID)
	if !ok {
		// Questioner left, skip round
		g.nextType1Round()
		return
	}

	g.room.sendTo(questioner, "type1_questioner", map[string]interface{}{
		"round":        g.t1CurrentRound,
		"total_rounds": g.t1Rounds,
		"question":     g.t1Question,
		"time_limit":   30,
	})

	others := g.room.allClientsExcept(g.t1QuestionerID)
	for _, c := range others {
		g.room.sendTo(c, "type1_waiting", map[string]interface{}{
			"round":               g.t1CurrentRound,
			"total_rounds":        g.t1Rounds,
			"questioner_nickname": questioner.nickname,
		})
	}

	g.resetTimer(type1ThinkTime, func() {
		g.mu.Lock()
		defer g.mu.Unlock()
		if g.state != "type1_thinking" {
			return
		}
		// Auto-assign random ranking
		g.t1QuestRanking = shuffledRanks()
		g.startType1AnswerPhase()
	})
}

func (g *Game) onType1QuestionerRanking(c *Client, ranking []int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.state != "type1_thinking" || c.id != g.t1QuestionerID {
		return
	}
	if !validRanking(ranking) {
		return
	}
	g.stopTimer()
	g.t1QuestRanking = ranking
	g.startType1AnswerPhase()
}

func (g *Game) startType1AnswerPhase() {
	g.state = "type1_answering"

	questioner, _ := g.room.getClient(g.t1QuestionerID)
	questNick := ""
	if questioner != nil {
		questNick = questioner.nickname
		g.room.sendTo(questioner, "type1_questioner_waiting", map[string]interface{}{
			"message": "참여자들이 답변 중입니다. 잠시 기다려주세요.",
		})
	}

	others := g.room.allClientsExcept(g.t1QuestionerID)
	for _, c := range others {
		g.room.sendTo(c, "type1_answer_phase", map[string]interface{}{
			"questioner_nickname": questNick,
			"question":            g.t1Question,
			"time_limit":          30,
		})
	}

	totalAnswerers := len(others)
	g.resetTimer(type1AnswerTime, func() {
		g.mu.Lock()
		defer g.mu.Unlock()
		if g.state != "type1_answering" {
			return
		}
		// Auto-fill missing guesses
		for _, c := range g.room.allClientsExcept(g.t1QuestionerID) {
			if _, ok := g.t1Guesses[c.id]; !ok {
				g.t1Guesses[c.id] = shuffledRanks()
			}
		}
		_ = totalAnswerers
		g.showType1Results()
	})
}

func (g *Game) onType1PlayerGuess(c *Client, ranking []int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.state != "type1_answering" || c.id == g.t1QuestionerID {
		return
	}
	if !validRanking(ranking) {
		return
	}
	if _, already := g.t1Guesses[c.id]; already {
		return
	}
	g.t1Guesses[c.id] = ranking

	nonQuesters := g.room.allClientsExcept(g.t1QuestionerID)
	g.room.broadcastJSON("type1_player_submitted", map[string]interface{}{
		"nickname":        c.nickname,
		"submitted_count": len(g.t1Guesses),
		"total_count":     len(nonQuesters),
	})

	if len(g.t1Guesses) >= len(nonQuesters) {
		g.stopTimer()
		g.showType1Results()
	}
}

type t1PlayerResult struct {
	ID       string `json:"id"`
	Nickname string `json:"nickname"`
	Ranking  []int  `json:"ranking"`
	Score    int    `json:"score"`
}

func (g *Game) showType1Results() {
	g.state = "type1_results"
	g.t1ReadyPlayers = make(map[string]bool)

	questioner, _ := g.room.getClient(g.t1QuestionerID)
	questNick := ""
	if questioner != nil {
		questNick = questioner.nickname
	}

	var results []t1PlayerResult
	g.room.mu.RLock()
	for id, c := range g.room.clients {
		if id == g.t1QuestionerID {
			continue
		}
		ranking := g.t1Guesses[id]
		if ranking == nil {
			ranking = []int{0, 0, 0, 0}
		}
		score := 0
		for i := 0; i < 4 && i < len(ranking) && i < len(g.t1QuestRanking); i++ {
			if ranking[i] == g.t1QuestRanking[i] {
				score++
			}
		}
		results = append(results, t1PlayerResult{
			ID: id, Nickname: c.nickname, Ranking: ranking, Score: score,
		})
	}
	g.room.mu.RUnlock()

	perfectCount := 0
	for _, pr := range results {
		if pr.Score == 4 {
			perfectCount++
		}
	}
	accuracy := 0.0
	if len(results) > 0 {
		accuracy = float64(perfectCount) / float64(len(results)) * 100
	}

	g.room.broadcastJSON("type1_results", map[string]interface{}{
		"round":               g.t1CurrentRound,
		"total_rounds":        g.t1Rounds,
		"questioner_nickname": questNick,
		"question":            g.t1Question,
		"questioner_ranking":  g.t1QuestRanking,
		"player_results":      results,
		"accuracy":            accuracy,
		"is_last_round":       g.t1CurrentRound >= g.t1Rounds,
	})
}

func (g *Game) onReadyNext(c *Client) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.state != "type1_results" {
		return
	}
	g.t1ReadyPlayers[c.id] = true
	total := g.room.clientCount()
	g.room.broadcastJSON("ready_next_update", map[string]interface{}{
		"ready_count": len(g.t1ReadyPlayers),
		"total_count": total,
	})
	if len(g.t1ReadyPlayers) >= total {
		g.t1ReadyPlayers = make(map[string]bool)
		g.nextType1Round()
	}
}

// ═══════════════════════════════════════════════
// TYPE 2
// ═══════════════════════════════════════════════

func (g *Game) initType2() {
	g.t2FailPlayers = make(map[string]bool)
	questions, err := loadPersonQuestions()
	if err != nil || len(questions) == 0 {
		g.room.broadcastJSON("error", map[string]interface{}{
			"message": "인물 퀴즈 이미지를 찾을 수 없습니다.",
		})
		return
	}
	g.t2Questions = questions
	g.t2CurrentIdx = 0
	g.room.mu.RLock()
	for id := range g.room.clients {
		g.t2Scores[id] = 0
	}
	g.room.mu.RUnlock()

	g.state = "type2_countdown"
	g.room.broadcastJSON("type2_starting", map[string]interface{}{"countdown": 5})

	g.resetTimer(type2StartWait, func() {
		g.mu.Lock()
		defer g.mu.Unlock()
		g.startType2Question()
	})
}

func (g *Game) startType2Question() {
	if g.t2CurrentIdx >= len(g.t2Questions) {
		g.endGame()
		return
	}
	g.t2FailPlayers = make(map[string]bool)
	g.state = "type2_question"
	g.t2RoundDone = false
	q := g.t2Questions[g.t2CurrentIdx]

	g.room.broadcastJSON("type2_question", map[string]interface{}{
		"question_number": g.t2CurrentIdx + 1,
		"total_questions": len(g.t2Questions),
		"image_url":       q.ImageURL,
		"time_limit":      10,
	})

	g.resetTimer(type2QTime, func() {
		g.mu.Lock()
		defer g.mu.Unlock()
		if g.state != "type2_question" {
			return
		}
		g.showType2ResultFail(q.Answer, q.ImageURL)
	})
}

func (g *Game) onType2Answer(c *Client, answer string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.state != "type2_question" || g.t2RoundDone {
		return
	}
	q := g.t2Questions[g.t2CurrentIdx]
	if normalizeAnswer(answer) == normalizeAnswer(q.Answer) {
		g.showType2Correct(c, q.Answer, q.ImageURL)
	} else {
		g.showType2Wrong(c, q.Answer, q.ImageURL)
	}
}

// 참여자가 정답을 틀렸을 경우
func (g *Game) showType2Wrong(c *Client, answer string, imageURL string) {
	log.Println("onType2Answer wrong!!")
	g.t2FailPlayers[c.id] = true
	total := g.room.clientCount()
	if len(g.t2FailPlayers) < total {
		g.room.sendTo(c, "type2_wrong", map[string]interface{}{
			"image_url": imageURL,
		})
	} else {
		g.showType2ResultFail(answer, imageURL)
	}
}

// 게임 종료: 참여자가 정답을 맞췄을 경우
func (g *Game) showType2Correct(c *Client, answer string, imageURL string) {
	g.state = "type2_result"
	log.Println("onType2Answer correct!!")
	g.t2RoundDone = true
	g.stopTimer()
	g.t2Scores[c.id]++
	g.room.broadcastJSON("type2_result_correct", map[string]interface{}{
		"winner_nickname": c.nickname,
		"answer":          answer,
		"image_url":       imageURL,
	})
	g.resetTimer(type2ResultWait, func() {
		g.mu.Lock()
		defer g.mu.Unlock()
		g.type2NextCountdown()
	})
}

// 게임 종료: 정답을 맞힌 참여자가 없을 경우
func (g *Game) showType2ResultFail(answer string, imageURL string) {
	g.state = "type2_result"
	g.room.broadcastJSON("type2_result_fail", map[string]interface{}{
		"answer":    answer,
		"image_url": imageURL,
	})
	g.resetTimer(type2ResultWait, func() {
		g.mu.Lock()
		defer g.mu.Unlock()
		g.type2NextCountdown()
	})
}

func (g *Game) type2NextCountdown() {
	g.t2CurrentIdx++
	if g.t2CurrentIdx >= len(g.t2Questions) {
		g.endGame()
		return
	}
	g.state = "type2_next_countdown"
	g.room.broadcastJSON("type2_next_countdown", map[string]interface{}{"countdown": 3})
	g.resetTimer(type2NextWait, func() {
		g.mu.Lock()
		defer g.mu.Unlock()
		g.startType2Question()
	})
}

// ═══════════════════════════════════════════════
// COMMON
// ═══════════════════════════════════════════════

func (g *Game) endGame() {
	g.state = "game_over"

	type ScoreEntry struct {
		ID       string `json:"id"`
		Nickname string `json:"nickname"`
		Score    int    `json:"score"`
	}
	var scores []ScoreEntry
	if g.quizType == "type2" {
		g.room.mu.RLock()
		for id, c := range g.room.clients {
			scores = append(scores, ScoreEntry{ID: id, Nickname: c.nickname, Score: g.t2Scores[id]})
		}
		g.room.mu.RUnlock()
	}

	g.room.broadcastJSON("game_over", map[string]interface{}{
		"quiz_type": g.quizType,
		"scores":    scores,
	})
	g.room.game = nil
}

func (g *Game) resetTimer(d time.Duration, f func()) {
	if g.timer != nil {
		g.timer.Stop()
	}
	g.timer = time.AfterFunc(d, f)
}

func (g *Game) stopTimer() {
	if g.timer != nil {
		g.timer.Stop()
	}
}

func shuffledRanks() []int {
	r := []int{1, 2, 3, 4}
	rand.Shuffle(len(r), func(i, j int) { r[i], r[j] = r[j], r[i] })
	return r
}

func validRanking(r []int) bool {
	if len(r) != 4 {
		return false
	}
	seen := make(map[int]bool)
	for _, v := range r {
		if v < 1 || v > 4 || seen[v] {
			return false
		}
		seen[v] = true
	}
	return true
}

func normalizeAnswer(s string) string {
	s = norm.NFC.String(s)
	return strings.ToLower(strings.TrimSpace(s))
}
