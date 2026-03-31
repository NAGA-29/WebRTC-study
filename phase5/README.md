# Phase 5: ArkEve応用 — リアルタイム座標同期

## 目標

- WebRTC DataChannelでリアルタイムに座標データを同期する
- 「信頼性なし・順序なし」DataChannelがリアルタイム通信に適している理由を理解する
- ArkEveなどのゲームへの応用イメージを確立する

---

## ファイル構成

```
phase5/
├── main.go              ← ルームID付きGo+Echoシグナリングサーバー
├── go.mod
└── static/
    ├── index.html       ← Canvas付きUI（ロール・ルームID設定）
    └── app.js           ← WebRTC接続・座標同期ロジック（ES Moduleとして分離）
```

---

## 実行方法

```bash
cd phase5
go mod tidy
go run main.go
```

2つのタブで開く:

```
タブ1: http://localhost:8080/?role=offerer&room=room1
タブ2: http://localhost:8080/?role=answerer&room=room1
```

またはブラウザでフォームから入力してもOK。

接続成功後、片方のタブでCanvas上でマウスを動かすと、
もう片方のタブにカーソルがリアルタイムで表示される。

---

## Phase 4からの変更点

### 1. ルームID付きのシグナリングAPI

```go
// Phase 4まで
e.POST("/offer", handler)

// Phase 5: :roomId パラメータ付き
e.POST("/rooms/:roomId/offer", handler)
e.GET("/rooms/:roomId/offer", handler)
// ...
```

これで複数の接続セッションを同時に管理できる。

### 2. DataChannelを「信頼性なし」に設定

```javascript
// Phase 3 (チャット): 信頼性あり・順序保証あり
pc.createDataChannel('chat', { ordered: true });

// Phase 5 (位置同期): 信頼性なし・順序なし
pc.createDataChannel('position', {
  ordered: false,      // 到着順序を保証しない
  maxRetransmits: 0,   // 再送しない（失われたパケットは捨てる）
});
```

**なぜ「信頼性なし」がリアルタイム通信に良いか？**

```
信頼性あり（TCP的）:
  フレーム1 → フレーム2 → フレーム3 → フレーム4
  フレーム2がロストすると... フレーム2の再送を待つ → フレーム3,4が詰まる → 遅延が増大

信頼性なし（UDP的）:
  フレーム1 → フレーム2 → [ロスト] → フレーム4 → フレーム5
  古いフレームは捨てる → 最新の位置データだけを描画 → 遅延が少ない
```

チャットは「全メッセージが届く必要がある」→ 信頼性あり
ゲームの位置は「最新の位置データだけ届けばOK」→ 信頼性なし

### 3. 送信レートの制限

```javascript
const SEND_INTERVAL_MS = 33; // 約30fps

function sendCoordinates(nx, ny) {
  const now = performance.now();
  if (now - lastSendTime < SEND_INTERVAL_MS) return; // スキップ
  lastSendTime = now;
  dc.send(JSON.stringify({ x: nx, y: ny, ts: Date.now() }));
}
```

`mousemove` は1秒間に100回以上発火することがある。
30fpsに絞ることでCPUとネットワーク帯域を節約する。

### 4. 座標の正規化

```javascript
// 送信: Canvas座標 → 正規化（0.0〜1.0）
const nx = mouseX / canvas.width;   // 例: 300 / 600 = 0.5
const ny = mouseY / canvas.height;  // 例: 200 / 400 = 0.5

// 受信: 正規化 → Canvas座標
const rx = remoteData.x * canvas.width;   // 0.5 * 600 = 300
const ry = remoteData.y * canvas.height;  // 0.5 * 400 = 200
```

正規化することで、解像度が違うデバイス間でも同じ位置に表示できる。

### 5. JSを別ファイルに分離（ES Module）

```html
<script type="module">
  import { connect, sendCoordinates, getRemotePosition } from './app.js';
</script>
```

アプリが複雑になったのでロジックを `app.js` に分離した。
`final/` フェーズではさらにクラスを使って整理する。

---

## データフォーマット

DataChannelで送受信するデータ:

```javascript
// 送信するJSONの形式
{
  x: 0.5,          // Canvas幅に対する相対位置（0.0〜1.0）
  y: 0.5,          // Canvas高さに対する相対位置（0.0〜1.0）
  ts: 1706000000000 // タイムスタンプ（未使用だが遅延計算に使える）
}
```

---

## ArkEveへの応用イメージ

このPhaseのサンプルを拡張すると:

```javascript
// プレイヤー位置の同期
{
  playerId: "player1",
  x: 100.5,
  y: 200.3,
  rotation: 1.57,
  animation: "walk",
  ts: Date.now()
}
```

複数プレイヤー対応には:
- **メッシュ型**: 全員がP2Pで繋がる（N人なら N*(N-1)/2 接続）
- **SFU型**: 専用サーバーが全員のデータを中継（スケーラブル）

---

## 演習

1. 座標に加えて「クリック」イベントも送受信してみる
   ```javascript
   canvas.addEventListener('click', () => {
     dc.send(JSON.stringify({ type: 'click', x: nx, y: ny }));
   });
   ```

2. `ts` フィールドを使って送受信の**遅延時間**を計算して表示する
   ```javascript
   const latency = Date.now() - data.ts;
   ```

3. 3人目のブラウザを追加して、3者接続を試みる
   （ヒント: ルームIDを分けて2つのPeerConnectionを持つ）

---

## 次のステップ

全フェーズの知識を統合して最終課題へ → **[Final: フル実装](../final/README.md)**
