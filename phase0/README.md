# Phase 0: WebRTC 前提知識

> このフェーズはコードなし。まず「全体像」を頭に入れる。

---

## WebRTCとは？

WebRTC（Web Real-Time Communication）は、ブラウザ間で **P2P（ピアツーピア）通信** を実現するブラウザ標準API。
サーバーを経由せず、ブラウザ同士が直接データをやり取りできる。

```
  [ブラウザ A]  ←── P2P接続（WebRTC） ──→  [ブラウザ B]
      ↑                                          ↑
      └──────── シグナリングサーバー ────────────┘
                 (接続情報の仲介のみ)
```

**重要**: WebRTCはP2P接続確立後、サーバーを介さない。シグナリングサーバーは接続「準備」のためだけに使う。

---

## 接続までの全体フロー

```
ブラウザA (Offerer)          シグナリングサーバー        ブラウザB (Answerer)
      |                              |                          |
      |--- createOffer() ----------->|                          |
      |--- POST /offer ------------->|                          |
      |    (SDP Offer)               |<-- GET /offer -----------|
      |                              |--- 200 SDP Offer ------->|
      |                              |    setRemoteDescription()|
      |                              |    createAnswer()        |
      |                              |<-- POST /answer ---------|
      |<-- GET /answer ------------->|    (SDP Answer)          |
      |    setRemoteDescription()    |                          |
      |                              |                          |
      |--- POST /candidates -------->|<-- POST /candidates -----|
      |    (ICE candidates)          |    (ICE candidates)      |
      |<-- GET /candidates ----------|--- GET /candidates ------>|
      |    addIceCandidate()         |    addIceCandidate()     |
      |                              |                          |
      |====== P2P接続確立 =====================================>|
      |<====== DataChannel通信（サーバー不要）==================|
```

---

## SDP（Session Description Protocol）

SDPは接続情報を記述するフォーマット。「私はこういう通信ができます」という自己紹介文。

### SDPに含まれる情報

```
v=0                              // バージョン
o=- 123456 2 IN IP4 127.0.0.1   // セッションID
s=-                              // セッション名
t=0 0                            // 有効期間（0=永続）

m=application 9 UDP/DTLS/SCTP webrtc-datachannel  // DataChannel宣言
c=IN IP4 0.0.0.0
a=ice-ufrag:xxxx                 // ICE認証用フラグメント
a=ice-pwd:xxxxxxxxxxxxxxxx       // ICE認証パスワード
a=fingerprint:sha-256 AA:BB:...  // DTLS証明書のフィンガープリント
a=setup:actpass                  // DTLS接続ロール
a=sctp-port:5000                 // SCTPポート（DataChannel用）
```

### Offer/Answerモデル

```
Offerer                    Answerer
   |                          |
   |-- createOffer() -->  SDP Offer
   |   (「こういう通信できます」)
   |                          |
   |<-- createAnswer() -- SDP Answer
   |   (「わかりました、こちらも対応できます」)
```

---

## ICE（Interactive Connectivity Establishment）

ICEは「どうやって相手に到達するか」を見つける仕組み。

### ICE Candidateの種類

```
種類         説明                              例
─────────────────────────────────────────────────────
host         自分のローカルIPアドレス          192.168.1.10:54321
srflx        STUNで発見したグローバルIP        203.0.113.5:54321
             (server reflexive = 反射アドレス)
relay        TURNサーバー経由のアドレス         198.51.100.1:3478
             (最後の手段、必ず繋がる)
```

### ICE接続の優先順位

```
1. host（直接接続） ← 最速・最優先
      ↓ 失敗したら
2. srflx（STUN経由） ← NATを越えて繋がる
      ↓ 失敗したら
3. relay（TURN経由） ← 確実だが遅い
```

---

## NAT（Network Address Translation）とは

家庭やオフィスのルーターは、内部ネットワークのIPアドレスを変換して外部と通信する。
これが「NAT」。WebRTCはこのNATを越える必要がある。

```
  [ブラウザA]                                    [ブラウザB]
  192.168.1.10        NAT                        192.168.2.20
      |               |                              |
      |  ルーター      |       ルーター               |
      |  203.0.113.1  |       198.51.100.1           |
      |               |                              |
      +----インターネット（グローバルIP空間）---------+
```

### NATの問題

- AはBのローカルIP（192.168.2.20）に直接届かない
- BもAのローカルIP（192.168.1.10）に直接届かない
- → STUNサーバーでグローバルIPを調べて、そのIPで接続する

---

## STUN / TURN

### STUN（Session Traversal Utilities for NAT）

「自分のグローバルIPを教えてくれるサーバー」

```
ブラウザA          STUNサーバー
    |--- 「私のIPは何ですか？」 -->|
    |<-- 「203.0.113.5:54321です」|
    |（srflx candidateとして使う）
```

無料の公開STUNサーバー: `stun.l.google.com:19302`

### TURN（Traversal Using Relays around NAT）

「通信を中継してくれるサーバー」。STUNで繋がらない場合の最終手段。

```
ブラウザA  ←→  TURNサーバー（中継）  ←→  ブラウザB
```

- 費用がかかる（帯域を使うため）
- 遅くなる（中継経由のため）
- でも必ず繋がる

---

## シグナリングとは

WebRTCはSDPとICE candidateの交換方法（シグナリングプロトコル）を**あえて定めていない**。
HTTP REST・WebSocket・QRコード・何でもOK。

このリポジトリでは **REST API（ポーリング方式）** を使う。
シンプルで curl でテストできるので学習に最適。

```
POST /offer       → Offerを保存
GET  /offer       → Offerを取得
POST /answer      → Answerを保存
GET  /answer      → Answerを取得
POST /candidates  → ICE candidateを追加
GET  /candidates  → ICE candidateを取得
```

---

## まとめ：接続成立までに必要なもの

| 要素 | 役割 | 誰が担う |
|------|------|---------|
| SDP Offer/Answer | 接続パラメータの合意 | ブラウザ（WebRTC API） |
| ICE Candidate | 到達可能なアドレスの探索 | ブラウザ + STUNサーバー |
| シグナリング | SDP・ICEの交換仲介 | Go+Echoサーバー |
| DTLS | 暗号化 | ブラウザ（自動） |
| P2P通信 | データ送受信 | ブラウザ同士（サーバー不要） |

---

## 次のステップ

Phase 0の内容が理解できたら → **[Phase 1: シグナリングサーバーを作る](../phase1/README.md)**
