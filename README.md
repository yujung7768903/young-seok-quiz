## 우주오락실
링크: https://young-seok-quiz.fly.dev/

## 소개
지구오락실에 출제된 문제를 친구들과 함께 웹에서 플레이 할 수 있습니다.
### 게임1. 너는 읽기 쉬운 마음이야.
* 출제자: 문제를 읽고, 1,2,3,4 순위 선택
* 참여자: 출제자가 선택한 1,2,3,4 순위 추측
<img width="825" height="662" alt="스크린샷 2026-03-02 오후 7 45 11" src="https://github.com/user-attachments/assets/df881a24-3aeb-4841-9a7b-e4d2cc454ff5" />

### 게임2. 인물 퀴즈
* 참여자: 화면에 나온 인물의 이름 맞추기
<img width="825" alt="스크린샷 2026-03-02 오후 7 47 13" src="https://github.com/user-attachments/assets/8c7b1e5d-1252-46b9-b1f1-84e97a296989" />

---

## 기술 스택

| 분류 | 기술 |
|---|---|
| Backend | Go, gorilla/websocket |
| Frontend | Vanilla JS / HTML / CSS |
| Deploy | Docker, fly.io |

---

## 아키텍처

```
Hub
├── Room [ABC123]
│   ├── Client (영석)  ── send chan ── writePump ── WebSocket conn ── 브라우저
│   │                               readPump  ──────────────────────────────▶
│   ├── Client (민준)  ── send chan ── writePump ── WebSocket conn ── 브라우저
│   └── Game
│       ├── Type1 (마음 퀴즈) 상태 관리
│       └── Type2 (인물 퀴즈) 상태 관리
└── Room [XYZ789]
    └── ...
```

**Hub**: 전체 방 목록을 관리하는 최상위 구조체 (`map[string]*Room` + `sync.RWMutex`)

**Room**: 같은 방의 클라이언트를 관리하고 브로드캐스트를 담당. `register` / `unregister` 채널로 클라이언트 입퇴장을 처리하며, 별도 고루틴(`room.run()`)에서 이벤트를 순차 처리한다.

**Client**: WebSocket 연결 하나에 대응하며, 고루틴 2개를 가진다.
- `readPump`: 브라우저 → 서버 수신 전담. `conn.ReadMessage()`로 블로킹 대기.
- `writePump`: 서버 → 브라우저 송신 전담. `send` 채널에서 꺼내 `conn.Write()` 호출.

> gorilla/websocket은 concurrent write를 허용하지 않으므로, Read/Write를 각각의 고루틴이 전담한다.

**Game**: 게임 상태 머신. `state` 필드로 현재 단계를 관리하고, `sync.Mutex`로 동시 접근을 보호한다. `time.AfterFunc`으로 제한 시간을 구현한다.

---

## WebSocket 메시지 흐름

### 기본 흐름

```
[브라우저]                        [Go 서버]

   send(msg)  ─────────────▶  readPump
                               └─ handleMessage()
                                   └─ 게임 로직 (game.go)
                                       └─ broadcastJSON() / sendTo()
                                           └─ c.send channel
   onmessage  ◀─────────────  writePump
```

### 브라우저 → 서버 메시지

| type | 설명 |
|---|---|
| `create_room` | 방 생성 |
| `join_room` | 방 입장 |
| `ready_start` | 게임 시작 준비 완료 |
| `type1_submit_ranking` | 출제자 순위 제출 |
| `type1_submit_guess` | 참여자 추측 제출 |
| `ready_next` | 다음 문제 준비 완료 |
| `type2_submit_answer` | 인물 퀴즈 답 제출 |

### 서버 → 브라우저 메시지

| type | 설명 |
|---|---|
| `room_created` / `room_joined` | 방 생성/입장 완료 |
| `player_joined` / `player_left` | 참여자 입퇴장 알림 |
| `ready_start_update` | 준비 현황 업데이트 |
| `game_started` | 게임 시작 |
| `type1_questioner` | 출제자에게 문제 전달 |
| `type1_waiting` | 출제자 대기 중 알림 |
| `type1_answer_phase` | 참여자 답변 시작 |
| `type1_player_submitted` | 참여자 제출 현황 |
| `type1_results` | 라운드 결과 |
| `ready_next_update` | 다음 문제 준비 현황 |
| `type2_starting` | 인물 퀴즈 카운트다운 |
| `type2_question` | 인물 퀴즈 문제 |
| `type2_wrong` | 오답 처리 |
| `type2_result_correct` / `type2_result_fail` | 정답/실패 결과 |
| `type2_next_countdown` | 다음 문제 카운트다운 |
| `game_over` | 게임 종료 및 최종 점수 |
| `error` | 에러 메시지 |

### 게임1 흐름 예시 (마음 퀴즈)

```
[영석-출제자]         [Go 서버]         [민준-참여자]

                   type1_questioner
영석 화면 표시  ◀──────────────────
                   type1_waiting
                   ─────────────────▶  민준 화면 표시

순위 선택 후 제출
type1_submit_ranking ──────────────▶

                   type1_answer_phase
                   ─────────────────▶  민준 답변 화면
type1_questioner_waiting
영석 대기 화면  ◀──────────────────

                   type2_submit_guess ◀── 민준 제출

                   type1_results
최종 결과      ◀─────────────────────▶  최종 결과
```
