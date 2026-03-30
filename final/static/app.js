// Final: WebRTCクライアント 完全版
//
// RoomClient   → シグナリングAPIの呼び出し・ルーム管理
// WebRTCClient → RTCPeerConnection・DataChannelの管理
// LogPanel     → 接続イベントログの取得・表示

// ===== RoomClient =====

export class RoomClient {
  constructor(roomId) {
    this.roomId = roomId;
    this.base = `/rooms/${roomId}`;
  }

  // ルームを新規作成してRoomClientを返す
  static async create() {
    const res = await fetch('/rooms', { method: 'POST' });
    const data = await res.json();
    return new RoomClient(data.roomId);
  }

  async postOffer(offer) {
    await this._post('/offer', { sdp: JSON.stringify(offer) });
  }

  async getOffer() {
    return this._pollSDP('/offer');
  }

  async postAnswer(answer) {
    await this._post('/answer', { sdp: JSON.stringify(answer) });
  }

  async getAnswer() {
    return this._pollSDP('/answer');
  }

  async postCandidate(side, candidate) {
    await this._post(`/candidates?side=${side}`, { candidate: JSON.stringify(candidate) });
  }

  async getCandidates(side) {
    const res = await fetch(`${this.base}/candidates?side=${side}`);
    const data = await res.json();
    return data.candidates || [];
  }

  async getLogs() {
    const res = await fetch(`${this.base}/log`);
    const data = await res.json();
    return data.logs || [];
  }

  async reset() {
    await fetch(`${this.base}/reset`, { method: 'DELETE' });
  }

  // SDPが届くまでポーリングする
  async _pollSDP(path) {
    while (true) {
      const res = await fetch(this.base + path);
      if (res.status === 200) {
        const data = await res.json();
        return JSON.parse(data.sdp);
      }
      await sleep(1000);
    }
  }

  async _post(path, body) {
    await fetch(this.base + path, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
  }
}

// ===== WebRTCClient =====

export class WebRTCClient {
  constructor(room, role, options = {}) {
    this.room = room;           // RoomClient
    this.role = role;           // 'offerer' | 'answerer'
    this.dc = null;             // DataChannel
    this.onMessage = options.onMessage || (() => {});
    this.onStateChange = options.onStateChange || (() => {});

    const iceServers = options.iceServers || [
      { urls: 'stun:stun.l.google.com:19302' },
    ];
    this.pc = new RTCPeerConnection({ iceServers });
    this._setupPeerConnection();
  }

  _setupPeerConnection() {
    const pc = this.pc;

    // ICE candidateが生成されたらサーバーに送る
    pc.onicecandidate = async (event) => {
      if (event.candidate) {
        await this.room.postCandidate(this.role, event.candidate.toJSON());
      }
    };

    // 接続状態の変化を通知する
    pc.oniceconnectionstatechange = () => {
      this.onStateChange('ice', pc.iceConnectionState);
    };

    pc.onconnectionstatechange = () => {
      this.onStateChange('connection', pc.connectionState);
    };
  }

  // 相手のICE candidatesをポーリングして追加する
  _startCandidatePolling() {
    const theirSide = this.role === 'offerer' ? 'answerer' : 'offerer';
    let idx = 0;
    const poll = async () => {
      const candidates = await this.room.getCandidates(theirSide);
      for (let i = idx; i < candidates.length; i++) {
        await this.pc.addIceCandidate(new RTCIceCandidate(JSON.parse(candidates[i])));
      }
      idx = candidates.length;
      if (!['connected', 'completed'].includes(this.pc.iceConnectionState)) {
        setTimeout(poll, 1000);
      }
    };
    poll();
  }

  // DataChannelのイベントを設定する
  _setupDataChannel(channel) {
    this.dc = channel;
    channel.onopen = () => this.onStateChange('datachannel', 'open');
    channel.onclose = () => this.onStateChange('datachannel', 'closed');
    channel.onmessage = (e) => this.onMessage(e.data);
    return channel;
  }

  // Offererとして接続を開始する
  async connectAsOfferer() {
    this._setupDataChannel(
      this.pc.createDataChannel('chat', { ordered: true })
    );

    const offer = await this.pc.createOffer();
    await this.pc.setLocalDescription(offer);
    await this.room.postOffer(offer);

    this._startCandidatePolling();

    const answer = await this.room.getAnswer();
    await this.pc.setRemoteDescription(new RTCSessionDescription(answer));
  }

  // Answererとして接続を待つ
  async connectAsAnswerer() {
    this.pc.ondatachannel = (event) => {
      this._setupDataChannel(event.channel);
    };

    const offer = await this.room.getOffer();
    await this.pc.setRemoteDescription(new RTCSessionDescription(offer));

    const answer = await this.pc.createAnswer();
    await this.pc.setLocalDescription(answer);
    await this.room.postAnswer(answer);

    this._startCandidatePolling();
  }

  // メッセージを送信する
  send(text) {
    if (!this.dc || this.dc.readyState !== 'open') {
      throw new Error('DataChannelが開いていません');
    }
    this.dc.send(text);
  }

  // 接続を閉じる
  close() {
    if (this.dc) this.dc.close();
    this.pc.close();
  }
}

// ===== LogPanel =====

export class LogPanel {
  constructor(elementId, roomClient) {
    this.el = document.getElementById(elementId);
    this.room = roomClient;
    this.lastCount = 0;
    this._polling = false;
  }

  startPolling(intervalMs = 2000) {
    this._polling = true;
    const poll = async () => {
      if (!this._polling) return;
      try {
        const logs = await this.room.getLogs();
        // 新しいログエントリだけ追加する
        for (let i = this.lastCount; i < logs.length; i++) {
          this._appendLog(logs[i]);
        }
        this.lastCount = logs.length;
      } catch (e) {
        // ネットワークエラーは無視
      }
      setTimeout(poll, intervalMs);
    };
    poll();
  }

  stopPolling() {
    this._polling = false;
  }

  _appendLog(entry) {
    const div = document.createElement('div');
    div.className = 'log-entry';
    div.innerHTML = `<span class="log-time">${entry.time}</span> ` +
      `<span class="log-event">[${entry.event}]</span> ` +
      `<span class="log-msg">${entry.message}</span>`;
    this.el.appendChild(div);
    this.el.scrollTop = this.el.scrollHeight;
  }

  addLocal(msg) {
    const div = document.createElement('div');
    div.className = 'log-entry local';
    const time = new Date().toLocaleTimeString('ja-JP', { hour12: false });
    div.innerHTML = `<span class="log-time">${time}</span> ` +
      `<span class="log-event">[client]</span> ` +
      `<span class="log-msg">${msg}</span>`;
    this.el.appendChild(div);
    this.el.scrollTop = this.el.scrollHeight;
  }

  clear() {
    this.el.innerHTML = '';
    this.lastCount = 0;
  }
}

// ===== ユーティリティ =====
export function sleep(ms) { return new Promise(r => setTimeout(r, ms)); }
