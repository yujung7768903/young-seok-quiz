package main

import (
	"math/rand"
	"os"
	"path/filepath"
	"strings"
)

// ──────────────────────────────────────────────
// Type 1 — 너는 읽기 쉬운 마음이야~
// ──────────────────────────────────────────────

type Type1Question struct {
	Question string   `json:"question"`
	Options  []string `json:"options"`
}

var type1Questions = []Type1Question{
	{
		Question: "지금 가장 가고 싶은 여행은?",
		Options: []string{
			"아기자기한 소도시 골목 투어",
			"살아숨쉬는 역사의 도시. 아름다운 건물과 미술관 투어",
			"먹는 게 삶의 낙. 맛있는 것들로 가득한 먹방 투어",
			"도파민 풀충전, 아름다운 풍경을 배경으로 액티비티 투어",
		},
	},
	{
		Question: "애인이 동아리 또는 동호회에 든다면, 싫은 순위",
		Options:  []string{"러닝", "봉사", "독서", "악기"},
	},
	{
		Question: "같이 일하기 싫은 사람은?",
		Options: []string{
			"열정이 넘쳐서 실수도 넘쳐나는 후배",
			"피드백이라고 간섭을 수시로 하는 선배",
			`"이게 맞아?", "확실해?"라고 계속 나한테 되묻는 상사`,
			"간섭은 안하지만, 뭘 물어봐도 제대로 답변해주지 않는 회피형 선배",
		},
	},
	{
		Question: "애인이 했을 때 가장 싫은 행동은?",
		Options: []string{
			"먹을 때 쩝쩝거리면서 다 흘리고, 같이 먹는 음식 더럽게 먹음",
			"이성 친구와 전화로 연애 고민 상담 1시간 이상",
			"가는 길에 어쩌다 마주쳐서 편의점에서 맥주",
			"옷 세일한다고 이성친구와 같은 브랜드에서 같은 디자인 옷 구매",
		},
	},
	{
		Question: "모처럼의 긴 연휴가 왔다. 이 때 내가 하고싶은 것은?",
		Options: []string{
			"쉬는 게 짱이다",
			"시간이 아까우니 생산적인 활동을 한다.",
			"친구들과 즐겁게 논다.",
			"가족들과 시간을 보낸다.",
		},
	},
}

func randomType1Question() Type1Question {
	return type1Questions[rand.Intn(len(type1Questions))]
}

// ──────────────────────────────────────────────
// Type 2 — 인물 퀴즈
// ──────────────────────────────────────────────

type PersonQuestion struct {
	ImageURL string `json:"image_url"`
	Answer   string `json:"answer"`
}

func loadPersonQuestions() ([]PersonQuestion, error) {
	dir := "static/quiz/person"
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var questions []PersonQuestion
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		ext := strings.ToLower(filepath.Ext(name))
		switch ext {
		case ".jpg", ".jpeg", ".png", ".webp", ".gif":
			answer := strings.TrimSuffix(name, filepath.Ext(name))
			questions = append(questions, PersonQuestion{
				ImageURL: "/static/quiz/person/" + name,
				Answer:   answer,
			})
		}
	}
	rand.Shuffle(len(questions), func(i, j int) { questions[i], questions[j] = questions[j], questions[i] })
	if len(questions) > 5 {
		questions = questions[:5]
	}
	return questions, nil
}
