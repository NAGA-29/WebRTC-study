# Phase 3: DataChannelチャット

## 目標

- WebRTC DataChannelでテキストチャットを実現する
- Offererが `createDataChannel()` を呼び、Answererが `ondatachannel` で受け取る違いを理解する
- データがサーバーを経由しないP2P通信を体験する

---

## ファイル構成

```
phase3/
├── main.go               ← Go+Echoシグナリングサーバー（Phase 2と同じ）
├── go.mod
└── static/
    ├── index.html        ← Offerer（DataChannelを作成する側）
    └── answer.html       ← Answerer（DataChannelを受け取る側）
```

---

## 実行方法

```bash
cd phase3
go mod tidy
go run main.go
```

- Offerer: http://localhost:8080
- Answerer: http://localhost:8080/answer.html

**手順**:
1. answer.htmlを開く（自動でOfferを待ち始める）
2. index.htmlを開き「接続開始」を押す
3. 両タブに「チャット」UIが表示されたらメッセージを送れる

---

## Phase 2からの変更点

Phase 2との違いは **DataChannelの追加のみ**。シグナリングサーバーは変更なし。

### Offerer側（index.html）で追加したコード

```javascript
// DataChannelを作成する（Offerer側だけが createDataChannel() を呼ぶ）
const dc = pc.createDataChannel('chat', {
  ordered: true,  // メッセージの順序を保証
});

dc.onopen = () => { /* チャットUIを表示 */ };
dc.onmessage = (event) => { /* 受信メッセージを表示 */ };

// 送信はこれだけ！サーバーを経由しないP2P通信
dc.send('こんにちは！');
```

### Answerer側（answer.html）で追加したコード

```javascript
// Answerer側は ondatachannel イベントで DataChannel を受け取る
// createDataChannel() は呼ばない！
pc.ondatachannel = (event) => {
  const dc = event.channel;

  dc.onopen = () => { /* チャットUIを表示 */ };
  dc.onmessage = (event) => { /* 受信メッセージを表示 */ };
};
```

---

## DataChannelの設定オプション

`createDataChannel(label, options)` の主なオプション:

| オプション | 説明 | デフォルト |
|-----------|------|----------|
| `ordered` | メッセージの順序を保証するか | `true` |
| `maxRetransmits` | 再送回数の上限（`null`=無制限） | `null` |
| `maxPacketLifeTime` | メッセージの有効期限(ms) | `null` |
| `protocol` | サブプロトコル名 | `""` |

### 信頼性の選択

```javascript
// チャット: 順序保証あり・再送あり（デフォルト）
pc.createDataChannel('chat', { ordered: true });

// ゲーム位置同期: 順序保証なし・再送なし（最新データだけ欲しい）
pc.createDataChannel('position', { ordered: false, maxRetransmits: 0 });
```

→ Phase 5 でゲームの位置同期に使う設定を学ぶ

---

## 重要な学習ポイント

1. **DataChannelはサーバーを経由しない**
   - `dc.send()` で送ったデータは直接相手のブラウザに届く
   - サーバーはシグナリング（接続確立）にしか使わない

2. **非対称なAPI**
   - Offerer: `createDataChannel('name')` を呼ぶ
   - Answerer: `ondatachannel` イベントで受け取る

3. **DataChannelはPeerConnectionが確立してから開く**
   - ICE接続 → DTLS暗号化確立 → DataChannel open の順

---

## ブラウザの開発者ツールで確認

- ネットワークタブ: シグナリングAPIのリクエストはある（接続確立まで）
- 接続確立後: メッセージ送受信のネットワーク通信は**存在しない**（P2Pのため）
- `chrome://webrtc-internals`: DataChannelの状態遷移が確認できる

---

## 演習

1. 送信したメッセージに「タイムスタンプ」を含めてみる
   ```javascript
   dc.send(JSON.stringify({ text: msg, ts: Date.now() }));
   ```

2. DataChannelで**ファイル**を送受信してみる（ArrayBufferを使う）
   ```javascript
   dc.send(arrayBuffer); // バイナリも送れる
   ```

3. `ordered: false, maxRetransmits: 0` に変えて、チャットが壊れることを確認する（Phase 5 の伏線）

---

## 次のステップ

STUNを使ってネットワーク越えの接続を試みる → **[Phase 4: NAT越え](../phase4/README.md)**
