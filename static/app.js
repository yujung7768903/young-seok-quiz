/* ═══════════════════════════════════════════════════════════
   마음 퀴즈 — 프론트엔드 메인 로직
═══════════════════════════════════════════════════════════ */

const App = (() => {
  // ── State ──────────────────────────────────────────────
  let ws = null;
  let state = {
    roomId: null,
    playerId: null,
    isHost: false,
    players: [],
    // Type 1
    t1SelectedRanks: [],   // [optionIndex, ...] in order of selection (index0 = 1순위)
    t1Submitted: false,
    t1TimerInterval: null,
    t1TimerSeconds: 0,
    // Type 2
    t2TimerInterval: null,
    t2CountdownInterval: null,
    t2Submitted: false,
    // ready
    readyStartClicked: false,
    readyNextClicked: false,
  };

  // ── URL helpers ────────────────────────────────────────
  const getUrlRoomId = () => new URLSearchParams(location.search).get('room');

  // ── Screen management ──────────────────────────────────
  function showScreen(name) {
    document.querySelectorAll('.screen').forEach(s => s.classList.remove('active'));
    const el = document.getElementById('screen-' + name);
    if (el) el.classList.add('active');
  }

  // ── Toast ──────────────────────────────────────────────
  let toastTimer = null;
  function toast(msg) {
    const el = document.getElementById('toast');
    el.textContent = msg;
    el.style.transform = 'translateX(-50%) translateY(0)';
    clearTimeout(toastTimer);
    toastTimer = setTimeout(() => {
      el.style.transform = 'translateX(-50%) translateY(80px)';
    }, 2500);
  }

  // ── WebSocket ──────────────────────────────────────────
  function connect(onOpen) {
    const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
    ws = new WebSocket(`${protocol}//${location.host}/ws`);
    ws.onopen = () => onOpen && onOpen();
    ws.onmessage = (e) => {
      // Messages might be newline-separated
      e.data.split('\n').forEach(line => {
        line = line.trim();
        if (!line) return;
        try {
          const msg = JSON.parse(line);
          handleServerMessage(msg);
        } catch (err) {
          console.error('parse error', err, line);
        }
      });
    };
    ws.onclose = () => toast('연결이 끊어졌습니다. 새로고침해주세요.');
    ws.onerror = () => toast('연결 오류가 발생했습니다.');
  }

  function send(type, data) {
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ type, data: data || {} }));
    }
  }

  // ── Server message router ──────────────────────────────
  function handleServerMessage({ type, data }) {
    const handlers = {
      room_created:              onRoomCreated,
      room_joined:               onRoomJoined,
      player_joined:             onPlayerJoined,
      player_left:               onPlayerLeft,
      error:                     onError,
      game_started:              onGameStarted,
      // Waiting Room
      ready_start_update:        onReadyStartUpdate,
      // Type 1
      type1_questioner:          onType1Questioner,
      type1_waiting:             onType1Waiting,
      type1_questioner_waiting:  onType1QuestionerWaiting,
      type1_answer_phase:        onType1AnswerPhase,
      type1_player_submitted:    onType1PlayerSubmitted,
      type1_results:             onType1Results,
      ready_next_update:         onReadyNextUpdate,
      // Type 2
      type2_starting:            onType2Starting,
      type2_question:            onType2Question,
      type2_wrong:               onType2Wrong,
      type2_result_correct:      onType2ResultCorrect,
      type2_result_fail:         onType2ResultFail,
      type2_next_countdown:      onType2NextCountdown,
      // Common
      game_over:                 onGameOver,
    };
    const fn = handlers[type];
    console.log(`handleServerMessage type: ${type}`);
    if (fn) fn(data);
    else console.log('unhandled:', type, data);
  }

  // ── Home ───────────────────────────────────────────────
  function goToNickname() {
    const roomId = getUrlRoomId();
    const btn = document.getElementById('nickname-btn');
    const title = document.getElementById('nickname-title');
    if (roomId) {
      title.textContent = '닉네임을 입력해주세요';
      btn.textContent = '입장';
    } else {
      title.textContent = '닉네임을 입력해주세요';
      btn.textContent = '방 만들기';
    }
    showScreen('nickname');
    document.getElementById('nickname-input').focus();
  }

  function submitNickname() {
    const nickname = document.getElementById('nickname-input').value.trim();
    if (!nickname) { toast('닉네임을 입력해주세요.'); return; }
    const roomId = getUrlRoomId();
    connect(() => {
      if (roomId) {
        send('join_room', { room_id: roomId, nickname });
      } else {
        send('create_room', { nickname });
      }
    });
  }

  // ── Waiting room ───────────────────────────────────────
  function onRoomCreated({ room_id, player_id, players }) {
    state.roomId = room_id;
    state.playerId = player_id;
    state.players = players;
    state.isHost = true;
    renderWaitingRoom();
    showScreen('waiting');
  }

  function onRoomJoined({ room_id, player_id, players, is_host }) {
    state.roomId = room_id;
    state.playerId = player_id;
    state.players = players;
    state.isHost = is_host;
    renderWaitingRoom();
    showScreen('waiting');
  }

  function onPlayerJoined({ players }) {
    state.players = players;
    renderWaitingRoom();
  }

  function onPlayerLeft({ players }) {
    state.players = players;
    renderWaitingRoom();
  }

  function renderWaitingRoom() {
    const url = `${location.origin}/?room=${state.roomId}`;
    document.getElementById('invite-url').textContent = url;
    document.getElementById('player-count-badge').textContent = state.players.length;
    document.getElementById('start-total-count').textContent = state.players.length;

    const list = document.getElementById('player-list');
    list.innerHTML = state.players.map(p => `
      <li>
        <span class="player-dot"></span>
        <span>${escHtml(p.nickname)}</span>
        ${p.is_host ? '<span class="host-badge">👑 호스트</span>' : ''}
      </li>
    `).join('');

    const canStart = state.players.length >= 2;
    const waitingStatus = document.getElementById('waiting-status');
    const readyStatus = document.getElementById('ready-status');
    const startBtn = document.getElementById('start-btn');
    waitingStatus.style.display = canStart ? 'none' : 'block';
    readyStatus.style.display = canStart ? 'block' : 'none';
    startBtn.style.display = canStart ? 'block' : 'none';
  }

  function copyInvite() {
    const url = `${location.origin}/?room=${state.roomId}`;
    navigator.clipboard.writeText(url).then(() => toast('링크 복사됨!')).catch(() => toast('복사 실패'));
  }

  function readyStart() {
    if (state.readyStartClicked) return;
    state.readyStartClicked = true;
    document.getElementById("start-btn").disabled = true;
    send('ready_start', { room_id: state.roomId });
  }

  function showQuizTypeSelect() {
    if (!state.isHost) return;
    showScreen('quiz-type');
  }

  // function startGame(quizType) {
  //   send('start_game', { quiz_type: quizType });
  // }

  function onReadyStartUpdate({ ready_count, total_count }) {
    document.getElementById('start-ready-count').textContent = ready_count;
    document.getElementById('start-total-count').textContent = total_count;
    if (ready_count >= total_count) {
      document.getElementById('start-btn').disabled = true;
    }
  }

  function onGameStarted() {
    // waiting for type1_questioner or type1_waiting
  }

  function onError({ message }) {
    toast('⚠️ ' + message);
  }

  // ══════════════════════════════════════════════════════
  // TYPE 1
  // ══════════════════════════════════════════════════════

  function onType1Questioner({ round, total_rounds, question, time_limit }) {
    state.t1SelectedRanks = [];
    state.t1Submitted = false;

    document.getElementById('t1q-round-label').textContent = `ROUND ${round} / ${total_rounds}`;
    document.getElementById('t1q-question').textContent = question.question;
    document.getElementById('t1q-submit-btn').disabled = true;

    renderType1Options('t1q-options', question.options, 'questioner');
    startTimer('t1q', time_limit);
    showScreen('type1-questioner');
  }

  function onType1Waiting({ round, total_rounds, questioner_nickname }) {
    document.getElementById('t1w-round-label').textContent = `ROUND ${round} / ${total_rounds}`;
    document.getElementById('t1w-questioner').textContent = questioner_nickname;
    showScreen('type1-waiting');
  }

  function onType1QuestionerWaiting() {
    document.getElementById('t1qw-submitted-list').innerHTML = '';
    stopTimer('t1q');
    showScreen('type1-questioner-wait');
  }

  function onType1AnswerPhase({ questioner_nickname, question, time_limit }) {
    state.t1SelectedRanks = [];
    state.t1Submitted = false;

    document.getElementById('t1a-questioner').textContent = questioner_nickname;
    document.getElementById('t1a-question').textContent = question.question;
    document.getElementById('t1a-submit-btn').disabled = true;

    renderType1Options('t1a-options', question.options, 'answer');
    startTimer('t1a', time_limit);
    showScreen('type1-answer');
  }

  function onType1PlayerSubmitted({ nickname, submitted_count, total_count }) {
    // Update questioner wait screen
    const qwList = document.getElementById('t1qw-submitted-list');
    const chip = document.createElement('span');
    chip.className = 'submitted-chip';
    chip.textContent = nickname;
    qwList.appendChild(chip);

    // Update submitted screen
    const sList = document.getElementById('t1s-submitted-list');
    const chip2 = document.createElement('span');
    chip2.className = 'submitted-chip';
    chip2.textContent = nickname;
    sList.appendChild(chip2);
  }

  function onType1Results({ round, total_rounds, questioner_nickname, question, questioner_ranking, player_results, accuracy, is_last_round }) {
    stopTimer('t1a');
    stopTimer('t1s');
    state.readyNextClicked = false;

    document.getElementById('t1r-round-label').textContent = `ROUND ${round} / ${total_rounds}`;
    document.getElementById('t1r-title').textContent = `${questioner_nickname}님의 마음 📊`;
    document.getElementById('t1r-accuracy').textContent = `${Math.round(accuracy)}%`;
    document.getElementById('t1r-next-btn').disabled = false;
    document.getElementById('t1r-ready-count').textContent = '0';
    document.getElementById('t1r-total-count').textContent = state.players.length;

    // 정답 보여주기
    // questioner_ranking[i] = rank assigned to option[i]
    const answerList = document.getElementById('t1r-answer-list');
    // Sort options by rank
    const sorted = question.options.map((opt, i) => ({ opt, rank: questioner_ranking[i] }))
      .sort((a, b) => a.rank - b.rank);
    answerList.innerHTML = sorted.map(({ opt, rank }) => `
      <div class="result-option-item">
        <span class="result-rank">${rank}순위</span>
        <span class="result-option-text">${escHtml(opt)}</span>
      </div>
    `).join('');

    // Player results
    const playerResultsEl = document.getElementById('t1r-player-results');
    playerResultsEl.innerHTML = (player_results || []).map(pr => {
      const chips = question.options.map((_, i) => {
        const correct = pr.ranking[i] === questioner_ranking[i];
        return `<span class="rank-chip ${correct ? 'correct' : 'wrong'}">${pr.ranking[i] || '?'}</span>`;
      }).join('');
      return `
        <div class="player-result-item">
          <span class="player-result-name">${escHtml(pr.nickname)}</span>
          <span class="player-result-ranks">${chips}</span>
          <span class="player-result-score">${pr.score}/4</span>
        </div>
      `;
    }).join('');

    showScreen('type1-results');
  }

  function onReadyNextUpdate({ ready_count, total_count }) {
    document.getElementById('t1r-ready-count').textContent = ready_count;
    document.getElementById('t1r-total-count').textContent = total_count;
    if (ready_count >= total_count) {
      document.getElementById('t1r-next-btn').disabled = true;
    }
  }

  // Render option cards for ranking
  function renderType1Options(containerId, options, mode) {
    const container = document.getElementById(containerId);
    container.innerHTML = options.map((opt, i) => `
      <div class="option-item" data-idx="${i}" onclick="App.selectOption('${containerId}', ${i}, '${mode}')">
        <span class="option-rank" id="${containerId}-rank-${i}">–</span>
        <span class="option-text">${escHtml(opt)}</span>
      </div>
    `).join('');
  }

  function selectOption(containerId, idx, mode) {
    if (state.t1Submitted) return;

    // Check if already selected
    const alreadyAt = state.t1SelectedRanks.indexOf(idx);
    if (alreadyAt !== -1) {
      // Deselect: remove it and all after it
      state.t1SelectedRanks = state.t1SelectedRanks.slice(0, alreadyAt);
    } else {
      if (state.t1SelectedRanks.length < 4) {
        state.t1SelectedRanks.push(idx);
      }
    }
    refreshOptionDisplay(containerId, mode);
  }

  function refreshOptionDisplay(containerId, mode) {
    const items = document.querySelectorAll(`#${containerId} .option-item`);
    items.forEach(item => {
      const idx = parseInt(item.dataset.idx);
      const rank = state.t1SelectedRanks.indexOf(idx);
      const rankEl = item.querySelector('.option-rank');
      if (rank !== -1) {
        rankEl.textContent = rank + 1;
        item.classList.add('selected');
      } else {
        rankEl.textContent = '–';
        item.classList.remove('selected');
      }
    });

    const done = state.t1SelectedRanks.length === 4;
    if (mode === 'questioner') {
      document.getElementById('t1q-submit-btn').disabled = !done;
    } else {
      document.getElementById('t1a-submit-btn').disabled = !done;
    }
  }

  function submitType1Ranking() {
    if (state.t1SelectedRanks.length !== 4 || state.t1Submitted) return;
    state.t1Submitted = true;
    // Convert: selectedRanks[rankIndex] = optionIdx
    // → ranking[optionIdx] = rank (1-based)
    const ranking = new Array(4).fill(0);
    state.t1SelectedRanks.forEach((optIdx, rankIdx) => {
      ranking[optIdx] = rankIdx + 1;
    });
    send('type1_submit_ranking', { ranking });
    document.getElementById('t1q-submit-btn').disabled = true;
    stopTimer('t1q');
  }

  function submitType1Guess() {
    if (state.t1SelectedRanks.length !== 4 || state.t1Submitted) return;
    state.t1Submitted = true;
    const ranking = new Array(4).fill(0);
    state.t1SelectedRanks.forEach((optIdx, rankIdx) => {
      ranking[optIdx] = rankIdx + 1;
    });
    send('type1_submit_guess', { ranking });
    stopTimer('t1a');

    // Go to submitted waiting screen
    document.getElementById('t1s-submitted-list').innerHTML = '';
    const currentTimer = state.t1TimerSeconds;
    startTimer('t1s', currentTimer);
    showScreen('type1-submitted');
  }

  function readyNext() {
    if (state.readyNextClicked) return;
    state.readyNextClicked = true;
    document.getElementById('t1r-next-btn').disabled = true;
    send('ready_next', {});
  }

  // ══════════════════════════════════════════════════════
  // TYPE 2
  // ══════════════════════════════════════════════════════

  function onType2Starting({ countdown }) {
    showScreen('type2-starting');
    let count = countdown;
    document.getElementById('t2s-countdown').textContent = count;
    clearInterval(state.t2CountdownInterval);
    state.t2CountdownInterval = setInterval(() => {
      count--;
      const el = document.getElementById('t2s-countdown');
      if (el) el.textContent = count;
      if (count <= 0) {
        clearInterval(state.t2CountdownInterval);
      }
    }, 1000);
  }

  function onType2Question({ question_number, total_questions, image_url, time_limit }) {
    state.t2Submitted = false;
    clearInterval(state.t2CountdownInterval);

    document.getElementById('t2q-label').textContent = `${question_number} / ${total_questions}`;
    document.getElementById('t2q-image').src = image_url;
    document.getElementById('t2q-answer-input').value = '';
    document.getElementById('t2q-answer-input').disabled = false;

    startTimer('t2q', time_limit);
    showScreen('type2-question');
    document.getElementById('t2q-answer-input').focus();
  }

  function submitType2Answer() {
    if (state.t2Submitted) return;
    const answer = document.getElementById('t2q-answer-input').value.trim();
    if (!answer) { toast('이름을 입력해주세요.'); return; }
    state.t2Submitted = true;
    document.getElementById('t2q-answer-input').disabled = true;
    send('type2_submit_answer', { answer });
  }

  function onType2Wrong({ image_url}) {
    const banner = document.getElementById('t2r-banner');
    banner.className = 'result-banner fail';
    document.getElementById('t2r-text').textContent = '땡!';
    document.getElementById('t2r-image').src = image_url;
    document.getElementById('t2r-answer').textContent = "?";
    document.getElementById('t2r-guide-msg').textContent = '참여자들이 답변 중입니다.';
    showScreen('type2-result');
  }

  function onType2ResultCorrect({ winner_nickname, answer, image_url }) {
    console.log("onType2ResultCorrect!!");
    stopTimer('t2q');
    const banner = document.getElementById('t2r-banner');
    banner.className = 'result-banner correct';
    document.getElementById('t2r-text').textContent = `${winner_nickname} 정답!`;
    document.getElementById('t2r-image').src = image_url;
    document.getElementById('t2r-answer').textContent = answer;
    document.getElementById('t2r-guide-msg').textContent = '';
    showScreen('type2-result');
  }

  function onType2ResultFail({ answer, image_url }) {
    stopTimer('t2q');
    const banner = document.getElementById('t2r-banner');
    banner.className = 'result-banner fail';
    document.getElementById('t2r-text').textContent = '땡!';
    document.getElementById('t2r-image').src = image_url;
    document.getElementById('t2r-answer').textContent = answer;
    document.getElementById('t2r-guide-msg').textContent = '';
    showScreen('type2-result');
  }

  function onType2NextCountdown({ countdown }) {
    let count = countdown;
    const el = document.getElementById('t2r-guide-msg');
    if (el) el.textContent = `다음 문제까지 ${count}초...`;
    clearInterval(state.t2CountdownInterval);
    state.t2CountdownInterval = setInterval(() => {
      count--;
      if (el) el.textContent = count > 0 ? `다음 문제까지 ${count}초...` : '';
      if (count <= 0) clearInterval(state.t2CountdownInterval);
    }, 1000);
  }

  // ══════════════════════════════════════════════════════
  // GAME OVER
  // ══════════════════════════════════════════════════════

  function onGameOver({ quiz_type, scores }) {
    stopTimer('t1q');
    stopTimer('t1a');
    stopTimer('t1s');
    stopTimer('t2q');
    clearInterval(state.t2CountdownInterval);

    const scoresWrap = document.getElementById('go-scores-wrap');
    const scoreboard = document.getElementById('go-scoreboard');
    const message = document.getElementById('go-message');

    if (quiz_type === 'type2' && scores && scores.length > 0) {
      scoresWrap.style.display = 'block';
      const sorted = [...scores].sort((a, b) => b.score - a.score);
      const medals = ['🥇', '🥈', '🥉'];
      scoreboard.innerHTML = sorted.map((s, i) => `
        <div class="score-item">
          <span class="score-rank-badge">${medals[i] || (i+1)+'위'}</span>
          <span class="score-name">${escHtml(s.nickname)}</span>
          <span class="score-num">${s.score}점</span>
        </div>
      `).join('');
      message.textContent = '';
    } else {
      scoresWrap.style.display = 'none';
      message.textContent = '모든 라운드가 끝났습니다. 수고하셨습니다!';
    }

    showScreen('game-over');
  }

  // ══════════════════════════════════════════════════════
  // TIMER HELPERS
  // ══════════════════════════════════════════════════════
  const timers = {};

  function startTimer(key, seconds) {
    stopTimer(key);
    let remaining = seconds;
    state.t1TimerSeconds = seconds;

    const textEl = document.getElementById(`${key}-timer-text`);
    const barEl  = document.getElementById(`${key}-timer-bar`);

    const update = () => {
      if (textEl) textEl.textContent = remaining;
      if (barEl) {
        const pct = (remaining / seconds) * 100;
        barEl.style.width = `${pct}%`;
        barEl.classList.toggle('warn',   pct <= 50 && pct > 25);
        barEl.classList.toggle('danger', pct <= 25);
      }
    };
    update();

    timers[key] = setInterval(() => {
      remaining--;
      state.t1TimerSeconds = remaining;
      update();
      if (remaining <= 0) stopTimer(key);
    }, 1000);
  }

  function stopTimer(key) {
    clearInterval(timers[key]);
    delete timers[key];
  }

  // ── Utility ────────────────────────────────────────────
  function escHtml(str) {
    return String(str)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;');
  }

  // ── Init ───────────────────────────────────────────────
  function init() {
    // Enter with invite link → show home with join intent
    const roomId = getUrlRoomId();
    if (roomId) {
      document.querySelector('#screen-home .btn-primary').textContent = '🎮 참여하기';
    }
    // Enter key on nickname input
    document.getElementById('nickname-input').addEventListener('keydown', e => {
      if (e.key === 'Enter') submitNickname();
    });
    // Enter key on type2 answer
    document.getElementById('t2q-answer-input').addEventListener('keydown', e => {
      if (e.key === 'Enter') submitType2Answer();
    });
  }

  init();

  // ── Public API ─────────────────────────────────────────
  return {
    showScreen,
    goToNickname,
    submitNickname,
    copyInvite,
    showQuizTypeSelect,
    selectOption,
    submitType1Ranking,
    submitType1Guess,
    readyStart,
    readyNext,
    submitType2Answer,
  };
})();
