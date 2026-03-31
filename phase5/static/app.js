// Phase 5: 座標同期 WebRTCクライアント
//
// このファイルはWebRTC接続のセットアップとDataChannelによる座標同期を担当する。
// index.html から読み込まれる。

// ===== URLパラメータからロールとルームIDを取得 =====
const params = new URLSearchParams(location.search);
const ROLE    = params.get('role') || 'offerer';   // 'offerer' or 'answerer'
const ROOM_ID = params.get('room') || 'room1';      // デフォルトルームID

// ===== シグナリングAPIのベースURL =====
const API = `/rooms/${ROOM_ID}`;

// ===== WebRTC設定 =====
const ICE_SERVERS = [
  { urls: 'stun:stun.l.google.com:19302' },
];

// ===== DataChannel設定 =====
// ordered: false, maxRetransmits: 0 → 信頼性なし・順序なし（UDP的挙動）
// リアルタイム位置同期では「古いデータを捨てる」ほうが遅延が少ない
const DC_OPTIONS = {
  ordered: false,       // 順序保証しない
  maxRetransmits: 0,    // 再送しない（最新データのみ優先）
};

// ===== 送信レート制御 =====
// mousemoveは1秒間に数十回発火するが、送信は30fps（33ms間隔）に制限する
const SEND_INTERVAL_MS = 33; // 約30fps

// ===== グローバル変数 =====
let dc = null;
let lastSendTime = 0;
let remoteX = -100; // 相手のカーソル位置（画面外に初期化）
let remoteY = -100;

// ===== ユーティリティ =====
function sleep(ms) { return new Promise(r => setTimeout(r, ms)); }

function log(msg, type = '') {
  const el = document.getElementById('log');
  const time = new Date().toLocaleTimeString('ja-JP', { hour12: false });
  el.innerHTML += `<span class="${type}">[${time}] ${msg}</span>\n`;
  el.scrollTop = el.scrollHeight;
}

function setStatus(msg, cls) {
  const el = document.getElementById('statusDiv');
  el.textContent = '状態: ' + msg;
  el.className = 'status ' + cls;
}

// ===== シグナリング =====
async function postOffer(offer) {
  await fetch(`${API}/offer`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ sdp: JSON.stringify(offer) }),
  });
}

async function waitForOffer() {
  while (true) {
    const res = await fetch(`${API}/offer`);
    if (res.status === 200) { const d = await res.json(); return JSON.parse(d.sdp); }
    await sleep(1000);
  }
}

async function postAnswer(answer) {
  await fetch(`${API}/answer`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ sdp: JSON.stringify(answer) }),
  });
}

async function waitForAnswer() {
  while (true) {
    const res = await fetch(`${API}/answer`);
    if (res.status === 200) { const d = await res.json(); return JSON.parse(d.sdp); }
    await sleep(1000);
  }
}

async function postCandidate(side, candidate) {
  await fetch(`${API}/candidates?side=${side}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ candidate: JSON.stringify(candidate) }),
  });
}

function pollCandidates(pc, theirSide) {
  let idx = 0;
  const poll = async () => {
    const res = await fetch(`${API}/candidates?side=${theirSide}`);
    const data = await res.json();
    const candidates = data.candidates || [];
    for (let i = idx; i < candidates.length; i++) {
      await pc.addIceCandidate(new RTCIceCandidate(JSON.parse(candidates[i])));
    }
    idx = candidates.length;
    if (!['connected', 'completed'].includes(pc.iceConnectionState)) {
      setTimeout(poll, 1000);
    }
  };
  poll();
}

// ===== DataChannelのセットアップ =====
function setupDataChannel(channel) {
  channel.onopen = () => {
    log('DataChannel オープン！座標同期を開始します', 'ok');
    setStatus('接続成功！マウスを動かしてください', 'connected');
    document.getElementById('hint').style.display = 'block';
  };

  channel.onclose = () => {
    log('DataChannel クローズ', 'warn');
    setStatus('接続切断', 'waiting');
  };

  // 相手からの座標データを受信する
  channel.onmessage = (event) => {
    try {
      const data = JSON.parse(event.data);
      // data = { x: number, y: number, ts: number }
      remoteX = data.x;
      remoteY = data.y;
      // 受信レートをログに出す（重すぎるので100回に1回だけ）
      if (Math.random() < 0.01) {
        log(`受信: x=${data.x.toFixed(0)}, y=${data.y.toFixed(0)}`);
      }
    } catch (e) {
      log(`受信データのパースエラー: ${e}`, 'err');
    }
  };

  return channel;
}

// ===== 座標を送信する =====
// Canvas上のマウス座標を相対値（0.0〜1.0）で送る → 解像度に依存しない
export function sendCoordinates(normalizedX, normalizedY) {
  if (!dc || dc.readyState !== 'open') return;

  const now = performance.now();
  // 送信レートを30fpsに制限（古いデータを捨てる）
  if (now - lastSendTime < SEND_INTERVAL_MS) return;
  lastSendTime = now;

  const payload = JSON.stringify({
    x: normalizedX,
    y: normalizedY,
    ts: Date.now(),
  });
  dc.send(payload);
}

// ===== 相手のカーソル位置を取得する（描画用）=====
export function getRemotePosition() {
  return { x: remoteX, y: remoteY };
}

// ===== メイン処理: WebRTC接続を確立する =====
export async function connect() {
  log(`ロール: ${ROLE}, ルーム: ${ROOM_ID}`);
  setStatus('接続中...', 'connecting');

  const pc = new RTCPeerConnection({ iceServers: ICE_SERVERS });

  pc.onicecandidate = async (event) => {
    if (event.candidate) {
      await postCandidate(ROLE, event.candidate.toJSON());
    }
  };

  pc.oniceconnectionstatechange = () => {
    log(`ICE状態: ${pc.iceConnectionState}`, 'warn');
    if (pc.iceConnectionState === 'failed') {
      setStatus('ICE接続失敗', 'failed');
    }
  };

  if (ROLE === 'offerer') {
    // Offerer: DataChannelを作成
    dc = setupDataChannel(
      pc.createDataChannel('position', DC_OPTIONS)
    );

    const offer = await pc.createOffer();
    await pc.setLocalDescription(offer);
    await postOffer(offer);
    log('Offer 送信完了。Answererを待っています...');

    pollCandidates(pc, 'answerer');

    const answer = await waitForAnswer();
    await pc.setRemoteDescription(new RTCSessionDescription(answer));
    log('Answer 受信・セット完了', 'ok');

  } else {
    // Answerer: ondatachannel でDataChannelを受け取る
    pc.ondatachannel = (event) => {
      dc = setupDataChannel(event.channel);
    };

    log('Offerを待機中...');
    const offer = await waitForOffer();
    await pc.setRemoteDescription(new RTCSessionDescription(offer));

    const answer = await pc.createAnswer();
    await pc.setLocalDescription(answer);
    await postAnswer(answer);
    log('Answer 送信完了', 'ok');

    pollCandidates(pc, 'offerer');
  }
}

// ===== リセット =====
export async function resetConnection() {
  await fetch(`${API}/reset`, { method: 'DELETE' });
  dc = null;
  remoteX = -100;
  remoteY = -100;
  lastSendTime = 0;
  setStatus('待機中（リセット済み）', 'waiting');
  log('--- リセット ---', 'warn');
}
