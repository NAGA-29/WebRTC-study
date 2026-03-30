# Final: 最終課題 — WebRTCフル実装

## 目標

Phase 1〜5の知識を統合して、2者がリアルタイムで通信できるWebRTCアプリを完成させる。

---

## ファイル構成

```
final/
├── main.go                ← Go+Echoサーバー（ルーム管理・シグナリング・ログAPI）
├── go.mod
├── static/
│   ├── index.html         ← 統合UI（チャット+接続ログパネル）
│   └── app.js             ← クラスベースのWebRTCクライアント
└── docs/
    └── architecture.md    ← システム構成図・設計解説
```

---

## 実行方法

```bash
cd final
go mod tidy
go run main.go
```

ブラウザで2つのタブを開く:
1. **タブ1（Offerer）**: http://localhost:8080 → ロール=「Offerer」→「接続開始」
2. **タブ2（Answerer）**: http://localhost:8080 → ルームID（タブ1に表示されたID）を入力 → ロール=「Answerer」→「接続開始」

接続成功後、DataChannelでテキストチャットができる。

---

## 使い方

### 接続手順

1. タブ1でロールを「Offerer」にして「接続開始」を押す
2. タブ1の「ルームID」欄にIDが表示される（例: `room-12345`）
3. タブ2でそのルームIDを入力し、ロールを「Answerer」にして「接続開始」
4. 両タブのステータスが「DC: open」になったらチャット可能

```
タブ1                                  タブ2
ロール: Offerer                        ロール: Answerer
ルームID: (空欄 → 自動生成)            ルームID: room-12345  ← タブ1のIDを入力
[接続開始]                             [接続開始]
```

---

## APIエンドポイント

| メソッド | パス | 説明 |
|---------|-----|------|
| `POST` | `/rooms` | 新規ルーム作成 |
| `POST` | `/rooms/:id/offer` | Offer保存 |
| `GET`  | `/rooms/:id/offer` | Offer取得 |
| `POST` | `/rooms/:id/answer` | Answer保存 |
| `GET`  | `/rooms/:id/answer` | Answer取得 |
| `POST` | `/rooms/:id/candidates?side=` | candidate追加 |
| `GET`  | `/rooms/:id/candidates?side=` | candidates取得 |
| `GET`  | `/rooms/:id/log` | 接続イベントログ取得 |
| `DELETE` | `/rooms/:id/reset` | ルームリセット |

---

## Phase 5からの変更点

### 1. クラスベースのJS設計

```javascript
// RoomClient: シグナリングAPIのラッパー
const room = await RoomClient.create(); // POST /rooms
await room.postOffer(offer);
const answer = await room.getAnswer();  // ポーリング込み

// WebRTCClient: RTCPeerConnectionのラッパー
const wrtc = new WebRTCClient(room, 'offerer', {
  onMessage: (text) => { /* 受信処理 */ },
  onStateChange: (type, state) => { /* 状態変化 */ },
});
await wrtc.connectAsOfferer();

// LogPanel: ログUIのラッパー
const logPanel = new LogPanel('logPanel', room);
logPanel.startPolling(1500); // 1.5秒ごとにサーバーのログを取得
```

### 2. 接続ログパネル

サーバー側でシグナリングイベントをタイムスタンプ付きでログに記録し、
`GET /rooms/:id/log` で取得してUIに表示する。

```go
room.addLog("offer_stored", "SDP Offerを受信・保存しました")
room.addLog("answer_stored", "SDP Answerを受信・保存しました")
room.addLog("candidate_added", "ICE candidate追加 (side=offerer)")
```

### 3. ルームの自動生成

Offererが接続開始するとサーバーが自動的にルームIDを生成。
AnswererはそのIDを入力して同じルームに参加する。

---

## Phase 1〜5の学習内容まとめ

| Phase | 学んだこと | Finalでの活用 |
|-------|-----------|-------------|
| Phase 0 | SDP・ICE・NATの概念 | アーキテクチャ理解の土台 |
| Phase 1 | Go+Echoシグナリング | `main.go` の設計 |
| Phase 2 | createOffer/Answer のフロー | `WebRTCClient.connectAsOfferer/Answerer` |
| Phase 3 | DataChannel | チャット機能 |
| Phase 4 | STUN・ICE設定 | `iceServers` 設定 |
| Phase 5 | ルームID・クラス分離 | `RoomClient`・`WebRTCClient` クラス |

---

## 次のステップ（ArkEveへの応用）

1. **複数人接続** → メッシュ型またはSFU型（`docs/architecture.md` 参照）
2. **状態同期** → Phase 5の座標同期をプレイヤーオブジェクトに拡張
3. **ゲームサーバー連携** → 権威サーバーでゲーム状態を管理、WebRTCでUI同期

詳細は [`docs/architecture.md`](./docs/architecture.md) を参照。
