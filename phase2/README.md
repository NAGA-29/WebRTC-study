# Phase 2: PeerConnection確立

## 目標

- ブラウザ2タブ間でWebRTC PeerConnectionを確立する
- `createOffer` → `setLocalDescription` → `setRemoteDescription` の流れを体験する
- ICE candidateの交換方法を理解する

---

## ファイル構成

```
phase2/
├── main.go               ← Go+Echo シグナリングサーバー（Phase 1と同じ構成）
├── go.mod
└── static/
    ├── index.html        ← Offerer（提供側）タブ
    └── answer.html       ← Answerer（応答側）タブ
```

---

## 実行方法

```bash
cd phase2
go mod tidy
go run main.go
```

ブラウザで以下の2つを**別タブ**で開く:
- Offerer: http://localhost:8080
- Answerer: http://localhost:8080/answer.html

**順番**: まずAnswererのタブを開き、次にOffererで「接続開始」ボタンを押す。

---

## 接続の流れ

```
index.html (Offerer)          サーバー              answer.html (Answerer)
      |                          |                          |
      |-- createOffer() -------->|                          |
      |-- POST /offer ---------->|                          |
      |   setLocalDescription()  |<-- GET /offer (poll) ----|
      |                          |--- 200 offer ----------->|
      |                          |    setRemoteDescription()|
      |                          |    createAnswer()        |
      |                          |    setLocalDescription() |
      |                          |<-- POST /answer ----------|
      |<-- GET /answer (poll) ---|                          |
      |   setRemoteDescription() |                          |
      |                          |                          |
      |-- POST /candidates ------>|<-- POST /candidates -----|
      |   (ICE candidates)        |   (ICE candidates)       |
      |<-- GET /candidates (poll)-|--- GET /candidates ----->|
      |   addIceCandidate()       |   addIceCandidate()      |
      |                          |                          |
      |====== PeerConnection 確立 =========================>|
```

---

## コードのポイント

### Offerer側の重要な処理（index.html）

```javascript
// 1. Offerを作成
const offer = await pc.createOffer();

// 2. 自分にセット → ICE candidate収集が始まる
await pc.setLocalDescription(offer);

// 3. シグナリングサーバーに送信
await postOffer(offer);

// 4. Answerを待ってセット
const answer = await waitForAnswer();
await pc.setRemoteDescription(new RTCSessionDescription(answer));
```

### Answerer側の重要な処理（answer.html）

```javascript
// 1. Offerをポーリングで取得
const offer = await waitForOffer();

// 2. 相手のOfferをセット
await pc.setRemoteDescription(new RTCSessionDescription(offer));

// 3. Answerを作成・自分にセット → ICE candidate収集が始まる
const answer = await pc.createAnswer();
await pc.setLocalDescription(answer);

// 4. シグナリングサーバーに送信
await postAnswer(answer);
```

### ICE candidate交換

```javascript
// candidateが生成されたら → サーバーに送る
pc.onicecandidate = async (event) => {
  if (event.candidate) {
    await fetch('/candidates?side=offerer', {
      method: 'POST',
      body: JSON.stringify({ candidate: JSON.stringify(event.candidate.toJSON()) })
    });
  }
};

// 相手のcandidatesをポーリングで取得 → addIceCandidate()
const data = await fetch('/candidates?side=answerer').then(r => r.json());
for (const c of data.candidates) {
  await pc.addIceCandidate(new RTCIceCandidate(JSON.parse(c)));
}
```

---

## 学習ポイント

1. **setLocalDescription → ICE収集開始** の順番が重要
2. **Offererが先にcreateOffer、AnswererがそれにcreateAnswer** する
3. ICE candidateは非同期に複数届く → **ポーリングで逐次追加**する
4. STUNなしでもローカルネットワーク内なら接続できる（`host` candidateのみ使用）

---

## ブラウザの開発者ツールで確認できること

- `chrome://webrtc-internals` でICE candidateの種類と接続ログが確認できる
- ネットワークタブでシグナリングAPIの通信が確認できる

---

## 演習

1. 接続成立後に `pc.getStats()` を呼んで接続情報を確認する
2. `iceServers: []` を `iceServers: [{ urls: 'stun:stun.l.google.com:19302' }]` に変えてみる（Phase 4の準備）
3. `oniceconnectionstatechange` イベントをすべてログに出力して、状態遷移を観察する

---

## 次のステップ

接続確立後にデータを送受信する → **[Phase 3: DataChannel](../phase3/README.md)**
