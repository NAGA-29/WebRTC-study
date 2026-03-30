# Final: アーキテクチャ解説

## システム構成

```
┌─────────────────────────────────────────────────────────────────┐
│                      Go + Echo サーバー                          │
│                      (localhost:8080)                            │
│                                                                  │
│  ┌────────────────┐    ┌──────────────────────────────────────┐  │
│  │  静的ファイル   │    │  シグナリングREST API                 │  │
│  │  配信          │    │                                      │  │
│  │  /             │    │  POST /rooms              ルーム作成  │  │
│  │  → index.html  │    │  POST /rooms/:id/offer    Offer保存  │  │
│  │  → app.js      │    │  GET  /rooms/:id/offer    Offer取得  │  │
│  └────────────────┘    │  POST /rooms/:id/answer   Answer保存 │  │
│                        │  GET  /rooms/:id/answer   Answer取得 │  │
│                        │  POST /rooms/:id/candidates  候補追加 │  │
│                        │  GET  /rooms/:id/candidates  候補取得 │  │
│                        │  GET  /rooms/:id/log      ログ取得   │  │
│                        └──────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                 ↑ HTTP REST (シグナリングのみ)
    ─────────────────────────────────────────────────
                 ↕ P2P接続（サーバー不要）
┌────────────────┐              ┌────────────────┐
│  ブラウザ A    │←────────────→│  ブラウザ B    │
│  (Offerer)     │  DataChannel │  (Answerer)    │
│                │  テキストチャット              │
│  RoomClient    │              │  RoomClient    │
│  WebRTCClient  │              │  WebRTCClient  │
│  LogPanel      │              │  LogPanel      │
└────────────────┘              └────────────────┘
```

---

## 接続確立のシーケンス

```
ブラウザA (Offerer)       Goサーバー (シグナリング)     ブラウザB (Answerer)
      |                          |                          |
      |--- POST /rooms ---------->|                          |
      |<-- { roomId: "room-xxx" }-|                          |
      |                          |<-- GET /rooms/xxx/offer --|-- ポーリング開始
      |                          |--- 204 No Content ------->|
      |                          |                          |
      | createOffer()            |                          |
      | setLocalDescription()    |                          |
      |--- POST /rooms/xxx/offer->|                          |
      |                          |--- 200 { sdp: "..." } --->|
      |                          |    setRemoteDescription() |
      |                          |    createAnswer()         |
      |                          |    setLocalDescription()  |
      |                          |<-- POST /rooms/xxx/answer-|
      |<-- GET /rooms/xxx/answer--|                          |
      | setRemoteDescription()   |                          |
      |                          |                          |
      |--- POST /candidates ------>|<-- POST /candidates -----|
      |    (ICE candidates)       |    (ICE candidates)       |
      |<-- GET /candidates (poll)-|--- GET /candidates ------>|
      | addIceCandidate()         |    addIceCandidate()      |
      |                          |                          |
      |================== P2P接続確立 ======================>|
      |<===== DataChannel open ==================================|
      |<===== チャット（サーバー不要）==========================|
      |                          |                          |
      |<-- GET /rooms/xxx/log --->|                          |
      |    接続ログを表示         |   (2秒毎にポーリング)     |
```

---

## クライアントのクラス設計

```
app.js
├── RoomClient           ← シグナリングAPIのラッパー
│   ├── create()         ← POST /rooms でルーム作成
│   ├── postOffer()      ← SDP Offerをサーバーに送る
│   ├── getOffer()       ← SDP Offerをポーリングで取得
│   ├── postAnswer()     ← SDP Answerをサーバーに送る
│   ├── getAnswer()      ← SDP Answerをポーリングで取得
│   ├── postCandidate()  ← ICE candidateを追加
│   ├── getCandidates()  ← ICE candidatesを取得
│   └── getLogs()        ← 接続ログを取得
│
├── WebRTCClient         ← RTCPeerConnectionのラッパー
│   ├── connectAsOfferer()  ← Offer作成→送信→Answer待機
│   ├── connectAsAnswerer() ← Offer待機→Answer作成→送信
│   ├── _setupPeerConnection()  ← onicecandidate等のセットアップ
│   ├── _setupDataChannel()     ← DataChannelのコールバック設定
│   ├── _startCandidatePolling() ← 相手のcandidatesを取得し続ける
│   └── send()           ← DataChannelでメッセージ送信
│
└── LogPanel             ← 接続ログUI
    ├── startPolling()   ← 定期的にログを取得して表示
    ├── stopPolling()    ← ポーリング停止
    ├── addLocal()       ← ブラウザ側のローカルログを追加
    └── clear()          ← ログをクリア
```

---

## Phase 1〜5との対比

| 項目 | Phase 1 | Phase 2 | Phase 3 | Phase 4 | Phase 5 | Final |
|------|---------|---------|---------|---------|---------|-------|
| シグナリング | REST | REST | REST | REST+config | REST+roomId | REST+roomId+log |
| WebRTC | なし | PeerConnection | DataChannel | DataChannel+STUN | DataChannel(UDP) | DataChannel |
| UI | テストUI | 2ファイル | 2ファイル | 1ファイル | Canvas | 統合UI |
| JS構成 | なし | インライン | インライン | インライン | ES Module | クラス設計 |
| ルーム | 単一 | 単一 | 単一 | 単一 | 名前付き | 自動生成 |

---

## 複数人対応への拡張（次のステップ）

現在のFinalは2者間（1対1）のみ対応。複数人対応には2つのアプローチがある。

### メッシュ型（P2Pフルメッシュ）

```
  A ←→ B
  ↕   ↕
  C ←→ D
```

- N人 → N*(N-1)/2 接続が必要
- 4人で6接続、10人で45接続
- シンプルだがスケールしない（4〜6人が限界）

```javascript
// 各ピアに対してWebRTCClientを作成する
const peers = new Map();
for (const peerId of otherPeerIds) {
  const client = new WebRTCClient(room, role, { ... });
  peers.set(peerId, client);
}
```

### SFU型（Selective Forwarding Unit）

```
     A
     ↓
 B ← SFU → C
     ↑
     D
```

- 全員がSFUサーバー1つに接続するだけ
- スケーラブル（100人でも100接続）
- SFUサーバーが必要（mediasoup, ion-sfu など）

ArkEveの本番環境ではSFU型が適切。
