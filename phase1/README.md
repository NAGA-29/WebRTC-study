# Phase 1: シグナリングサーバー

## 目標

- Go + Echo でWebRTC用シグナリングサーバーを作る
- `curl` でAPIの動作を確認する
- シグナリングの役割（SDP・ICE candidateの仲介）を理解する

---

## ファイル構成

```
phase1/
├── main.go           ← Go+Echo REST APIサーバー
├── go.mod
└── static/
    └── index.html    ← APIテスト用UI（本物のWebRTCではない）
```

---

## 実行方法

```bash
cd phase1
go mod tidy
go run main.go
```

サーバーが `:8080` で起動する。
ブラウザで http://localhost:8080 を開くとテストUIが表示される。

---

## APIエンドポイント一覧

| メソッド | パス | 説明 |
|---------|-----|------|
| `POST` | `/offer` | SDP Offerを保存 |
| `GET`  | `/offer` | SDP Offerを取得 |
| `POST` | `/answer` | SDP Answerを保存 |
| `GET`  | `/answer` | SDP Answerを取得 |
| `POST` | `/candidates?side=offerer\|answerer` | ICE candidateを追加 |
| `GET`  | `/candidates?side=offerer\|answerer` | ICE candidatesを取得 |
| `DELETE` | `/reset` | ストアをリセット（開発用） |

---

## curl での動作確認

### 1. Offer を保存する

```bash
curl -X POST http://localhost:8080/offer \
  -H "Content-Type: application/json" \
  -d '{"sdp": "v=0\r\no=- 12345 2 IN IP4 127.0.0.1\r\n..."}'
```

レスポンス:
```json
{"status": "ok"}
```

### 2. Offer を取得する

```bash
curl http://localhost:8080/offer
```

レスポンス:
```json
{"sdp": "v=0\r\no=- 12345 2 IN IP4 127.0.0.1\r\n..."}
```

まだOfferがない場合: `204 No Content`

### 3. Answer を保存する

```bash
curl -X POST http://localhost:8080/answer \
  -H "Content-Type: application/json" \
  -d '{"sdp": "v=0\r\no=- 67890 2 IN IP4 127.0.0.1\r\n..."}'
```

### 4. ICE Candidateを追加する

```bash
# Offerer側のcandidateを追加
curl -X POST "http://localhost:8080/candidates?side=offerer" \
  -H "Content-Type: application/json" \
  -d '{"candidate": "candidate:1 1 UDP 2122252543 192.168.1.10 54321 typ host"}'

# Answerer側のcandidateを追加
curl -X POST "http://localhost:8080/candidates?side=answerer" \
  -H "Content-Type: application/json" \
  -d '{"candidate": "candidate:1 1 UDP 2122252543 192.168.2.20 54322 typ host"}'
```

### 5. ICE Candidatesを取得する

```bash
curl "http://localhost:8080/candidates?side=offerer"
```

レスポンス:
```json
{
  "candidates": ["candidate:1 1 UDP ..."],
  "total": 1
}
```

### 6. ストアをリセット

```bash
curl -X DELETE http://localhost:8080/reset
```

---

## コードのポイント

### sync.RWMutex で並行アクセスを保護

```go
store.mu.Lock()    // 書き込み時はロック
store.offer = body.SDP
store.mu.Unlock()

store.mu.RLock()   // 読み取り時はRLock（複数同時OK）
offer := store.offer
store.mu.RUnlock()
```

### 204 No Content でポーリングに対応

```go
if offer == "" {
    return c.NoContent(http.StatusNoContent) // まだデータなし
}
return c.JSON(http.StatusOK, sdpBody{SDP: offer})
```

ブラウザ側は204が返ってきたら1秒待って再度GETする（ポーリング）。

---

## 学習ポイント

1. **シグナリングプロトコルはWebRTCが定めない** → REST・WebSocket・何でもOK
2. **SDP = 接続パラメータの文字列** → 中身を読むと自分のIPや暗号化情報が入っている
3. **ICE candidate = 到達可能なアドレス** → 複数種類（host/srflx/relay）がある
4. **このサーバーはWebRTCに直接参加しない** → ただの「情報の中継所」

---

## 演習

1. curlでOffer・Answerを手動で保存・取得してみる
2. candidates を `side=offerer` と `side=answerer` 両方で追加して取得してみる
3. `main.go` の `SignalingStore` に**ルームID**を追加して複数ルームに対応してみる（Phase 5の準備）

---

## 次のステップ

このAPIを使って実際にWebRTC接続を確立する → **[Phase 2: PeerConnection](../phase2/README.md)**
