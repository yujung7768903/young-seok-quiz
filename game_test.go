package main

import "testing"

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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validRanking(tt.input)
			if got != tt.want {
				t.Errorf("validRanking(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
