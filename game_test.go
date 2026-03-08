package main

import (
	"testing"

	"golang.org/x/text/unicode/norm"
)

func TestType1Results(t *testing.T) {
	hub := newHub()
	room := newRoom("ROOM01", hub)
	game := newGame(room)
	host := newTestClient()
	game.t1QuestionerID = host.id
	// 정답
	game.t1QuestRanking = []int{1, 2, 3, 4}

	// player1: 1,2,3,4 모두 일치 -> 4점
	player1 := newTestClient()
	player1.nickname = "player1(정답자)"
	game.t1Guesses[player1.id] = []int{1, 2, 3, 4}
	room.clients[player1.id] = player1
	// player2: 1,2 번 일치, 3,4번 불일치 -> 2점
	player2 := newTestClient()
	player2.nickname = "player2"
	game.t1Guesses[player2.id] = []int{1, 2, 4, 3}
	room.clients[player2.id] = player2
	// player3: 1,2,3,4 불일치 -> 0점
	player3 := newTestClient()
	player3.nickname = "player3"
	game.t1Guesses[player3.id] = []int{4, 3, 2, 1}
	room.clients[player3.id] = player3

	// 수행
	results := game.collectType1Results()
	count := game.countPerfectScores(results)

	// 검증
	if len(results) != 3 {
		t.Errorf("result length = %d, want = %d", len(results), 3)
	}
	for _, result := range results {
		switch result.ID {
		case player1.id:
			if result.Score != 4 {
				t.Errorf("player1 score = %d, want = %d", result.Score, 4)
			}
		case player2.id:
			if result.Score != 2 {
				t.Errorf("player2 score = %d, want = %d", result.Score, 2)
			}
		case player3.id:
			if result.Score != 0 {
				t.Errorf("player3 score = %d, want = %d", result.Score, 0)
			}
		}
	}

	if count != 1 {
		t.Errorf("정답자 수 = %d, want = %d", count, 1)
	}
}

func TestValidRanking(t *testing.T) {
	tests := []struct {
		name  string
		input []int
		want  bool
	}{
		{"정상 입력 오름차순", []int{1, 2, 3, 4}, true},
		{"정상 입력 내림차순", []int{4, 3, 2, 1}, true},
		{"정상 입력 섞인 순서", []int{3, 1, 4, 2}, true},
		{"길이 부족", []int{1, 2, 3}, false},
		{"길이 초과", []int{1, 2, 3, 4, 5}, false},
		{"중복 포함", []int{1, 1, 3, 4}, false},
		{"범위 초과 (5 포함)", []int{1, 2, 3, 5}, false},
		{"0 포함", []int{0, 1, 2, 3}, false},
		{"빈 슬라이스", []int{}, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := validRanking(test.input)
			if got != test.want {
				t.Errorf("validRanking(%v) = %v, want %v", test.input, got, test.want)
			}
		})
	}
}

func TestNormalizeAnswer(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		target string
		want   bool
	}{
		{"일치", "김유정", "김유정", true},
		{"대문자", "KIM YOUJUNG", "kim youjung", true},
		{"공백 포함", " 김유정 ", "김유정", true},
		{"NFC vs NFD", norm.NFC.String("김유정"), norm.NFD.String("김유정"), true},
		{"NFD vs NFD", norm.NFD.String("김유정"), norm.NFD.String("김유정"), true},
		{"다른 이름", "김유정", "김유미", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := normalizeAnswer(test.input) == normalizeAnswer(test.target)
			if got != test.want {
				t.Errorf("%s vs %s = %t, want: %t", test.input, test.target, got, test.want)
			}
		})
	}
}
