# Phase 4: NAT越え（STUN/TURN）

## 目標

- STUNとTURNの違いを理解する
- ICE candidateの種類（host/srflx/relay）を実際に観察する
- なぜ異なるネットワーク間では接続が失敗するかを説明できる

---

## ファイル構成

```
phase4/
├── main.go               ← GET /config エンドポイントを追加したサーバー
├── go.mod
└── static/
    └── index.html        ← 1ページ完結型（ロールをラジオボタンで選択）
```

---

## 実行方法

```bash
cd phase4
go mod tidy

# STUNのみ（デフォルト）
go run main.go

# TURN付き
TURN_URL=turn:coturn.example.com:3478 TURN_USER=user TURN_PASS=pass go run main.go
```

2つのブラウザタブで http://localhost:8080 を開き:
- タブ1: ロール = **Offerer** → 「接続開始」を押す
- タブ2: ロール = **Answerer** → 「接続開始」を押す

接続後、**ICE Candidateテーブル**でcandidateの種類を確認する。

---

## Phase 3からの変更点

### サーバー側: `GET /config` を追加

```go
func handleGetConfig(c echo.Context) error {
    servers := []ICEServerConfig{
        {URLs: "stun:stun.l.google.com:19302"},
    }
    // 環境変数にTURN設定があれば追加
    if turnURL := os.Getenv("TURN_URL"); turnURL != "" {
        servers = append(servers, ICEServerConfig{
            URLs:       turnURL,
            Username:   os.Getenv("TURN_USER"),
            Credential: os.Getenv("TURN_PASS"),
        })
    }
    return c.JSON(http.StatusOK, map[string]interface{}{"iceServers": servers})
}
```

### ブラウザ側: iceServers を設定

```javascript
// 起動時にサーバーからICE設定を取得
const res = await fetch('/config');
const config = await res.json();
const iceServers = config.iceServers;

// iceServers を RTCPeerConnection に渡す
const pc = new RTCPeerConnection({ iceServers });
//                                ^^^^^^^^^^^^
// Phase 2/3 では {} だったが、ここで STUN/TURN を指定する
```

---

## ICE Candidateの種類と見方

```
candidate:1 1 UDP 2122252543 192.168.1.10 54321 typ host
                              ^^^^^^^^^^^^^^^^ ^^^
                              IPアドレス       ポート
                                                         ^^^^^^^^
                                                         type=host
```

| type | 意味 | いつ現れる |
|------|------|---------|
| `host` | ローカルIP | 常に（STUNなしでも） |
| `srflx` | NATの外側IP（グローバルIP） | STUNサーバーが必要 |
| `relay` | TURNサーバー経由のアドレス | TURNサーバーが必要 |
| `prflx` | 通信中に発見されたアドレス | まれに現れる |

---

## NAT越えの仕組み

### 同じネットワーク内（Phase 2/3と同じ）

```
ブラウザA ──── (host candidate) ──── ブラウザB
              直接接続できる
```

### 異なるネットワーク（インターネット越え）

```
ブラウザA (192.168.1.10)               ブラウザB (192.168.2.20)
     |                                      |
     |  ルーターA (203.0.113.1)     ルーターB (198.51.100.1)
     |                                      |
     +────────── インターネット ─────────────+

問題: AはBの 192.168.2.20 に届かない（プライベートIPはルーティング不可）
解決: STUNで 203.0.113.1:54321 のようなグローバルIPを発見
     → srflx candidateとして使う
```

### STUNでも繋がらないとき

対称型NAT（Symmetric NAT）では、STUNで発見したIPがSTUNサーバーとのみ有効。
→ **TURN**でリレーするしかない。

```
ブラウザA ──→ TURNサーバー ←── ブラウザB
              (中継転送)
```

---

## 接続失敗のデバッグ方法

1. **`chrome://webrtc-internals`** を開く
2. ICE candidateの種類を確認
3. `host` しかない → STUNサーバーに到達できない
4. `srflx` はあるが接続できない → 対称型NAT。TURNが必要
5. `relay` があるが接続できない → TURNの認証情報が間違っている

---

## 無料で使えるSTUN/TURNサーバー

### STUNサーバー（無料）

```javascript
{ urls: 'stun:stun.l.google.com:19302' }
{ urls: 'stun:stun1.l.google.com:19302' }
{ urls: 'stun:stun.cloudflare.com:3478' }
```

### TURNサーバー（有料または自前構築）

**自前構築**: [coturn](https://github.com/coturn/coturn) をVPSにインストール
**有料クラウド**: Twilio, Daily.co, Metered.ca など

---

## 演習

1. `iceServers: []`（STUNなし）と `iceServers: [{ urls: 'stun:...' }]` で
   収集されるcandidateの種類の違いを観察する

2. `iceTransportPolicy: 'relay'` を設定して、relay candidateのみ使う接続を試みる
   ```javascript
   const pc = new RTCPeerConnection({
     iceServers,
     iceTransportPolicy: 'relay', // TURNが設定されている場合のみ接続可能
   });
   ```

3. `chrome://webrtc-internals` で実際の接続に使われたcandidateペアを確認する

---

## 次のステップ

座標データをリアルタイムに同期する → **[Phase 5: ArkEve応用](../phase5/README.md)**
