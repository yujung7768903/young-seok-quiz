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
		Question: "지금 가장 가고 싶은 여행 순위는?",
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
		Question: "같이 일하기 싫은 사람 순위는?",
		Options: []string{
			"열정이 넘쳐서 실수도 넘쳐나는 후배",
			"피드백이라고 간섭을 수시로 하는 동료",
			`본인도 제대로 모르면서 "이게 맞아?", "확실해?"라고 계속 나한테 되묻는 상사`,
			"뭘 물어봐도 제대로 도와주지 않고 본인 일만 하는 동료",
		},
	},
	{
		Question: "애인이 했을 때 가장 싫은 순위는?",
		Options: []string{
			"먹을 때 쩝쩝거리면서 다 흘리고, 같이 먹는 음식 더럽게 먹음",
			"이성 친구와 전화로 연애 고민 상담 1시간 이상",
			"가는 길에 어쩌다 마주쳐서 편의점에서 맥주",
			"옷 세일한다고 이성친구와 같은 브랜드에서 같은 디자인 옷 구매",
		},
	},
	{
		Question: "모처럼의 긴 연휴가 왔다. 이 때 내가 하고싶은 순위는?",
		Options: []string{
			"쉬는 게 짱이다",
			"시간이 아까우니 생산적인 활동을 한다.",
			"친구들과 즐겁게 논다.",
			"가족들과 시간을 보낸다.",
		},
	},
	{
		Question: "여행갈 때 더 같이 가고 싶은 사람 순위는?",
		Options: []string{
			"계획은 없지만, 하자는 대로 다 하는 OK맨",
			"휴식부터 먹거리, 놀거리까지 완벽한 플랜을 짜온 계획러",
			"길 좀 잘못 들면 어때? 여행 온 것 자체로 행복한 긍정파!",
			"말은 좀 툴툴대도, 맛있는 거 많이 사주고 잘 챙겨주는 츤데레",
		},
	},
	{
		Question: "일상 속에서 내가 더 견디기 힘든 순위는?",
		Options: []string{
			"약속 시간에 나만 있고, 다른 친구들 1시간 이상 늦는다고 함",
			"계획에 없는 야근 or 계획에 없던 보충 학습",
			"오랜만에 외출+예쁘게 꾸민 날. 흙탕물 + 비 + 바람 콤보",
			"내 얘기 안들어주고, 본민 불평불만만 얘기하는 사람",
		},
	},
	{
		Question: "내가 좋아하는 과목 순위는?",
		Options: []string{
			"수학",
			"영어",
			"미술",
			"체육",
		},
	},
	{
		Question: "다음 생에 다시 태어난다면, 살아보고 싶은 삶의 순위는?",
		Options: []string{
			"지금 이대로 똑같이 한 번 더",
			"전 세계적인 스타",
			"재벌",
			"대통령",
		},
	},
	{
		Question: "매주 1회 배워야 한다면, 배우고 싶은 순위는?",
		Options: []string{
			"악기",
			"춤",
			"노래",
			"언어(외국어, 수화 등)",
		},
	},
	{
		Question: "친구가 약속 시간에 한시간 늦었다면?",
		Options: []string{
			"어떻게 이렇게 늦냐며 화를 낸다.",
			"시간이 많이 남았으니 나 혼자 논다.",
			"시간 아까우니 집에 간다.",
			"다음에 만날 땐 나도 늦는다.",
		},
	},
	{
		Question: "돈을 많이 사용하는 곳의 순위는?",
		Options: []string{
			"식비",
			"쇼핑",
			"취미",
			"인테리어",
		},
	},
	{
		Question: "좋아하는 프로그램 순위는?",
		Options: []string{
			"먹방",
			"추리&탈출",
			"다큐",
			"레슬링",
		},
	},
	{
		Question: "연애 프로그램에 출연했다면, 호감가는 사람 순위는?",
		Options: []string{
			"취미나 성향이 비슷한 사람",
			"성향이 반대인 사람",
			"먼저 다가와주는 사람",
			"조용하고 차분한 사람",
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
