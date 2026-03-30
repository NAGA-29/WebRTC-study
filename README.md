# WebRTC 学習リポジトリ

WebRTCをフェーズ別に段階的に学習するための教材リポジトリです。
各フェーズは独立したフォルダになっており、順番に進めることで体系的に理解できます。

---

## 学習フェーズ一覧

| フェーズ | フォルダ | 内容 | ゴール |
|---------|---------|------|--------|
| Phase 0 | [phase0/](./phase0/) | WebRTC前提知識（ドキュメントのみ） | SDP/ICE/NATを説明できる |
| Phase 1 | [phase1/](./phase1/) | Go+Echoシグナリングサーバー | curlでAPI動作を確認できる |
| Phase 2 | [phase2/](./phase2/) | PeerConnection確立 | 2タブ間でWebRTC接続が成立する |
| Phase 3 | [phase3/](./phase3/) | DataChannelチャット | 2タブ間でテキストチャットができる |
| Phase 4 | [phase4/](./phase4/) | NAT越え（STUN/TURN） | candidateの種類を説明できる |
| Phase 5 | [phase5/](./phase5/) | 座標同期（ArkEve応用） | リアルタイムで座標データを同期できる |
| Final   | [final/](./final/)   | 最終課題（フル実装） | 2人が通信できるWebRTCアプリを完成させる |

---

## 前提環境

- **Go 1.21以上**
- **モダンブラウザ**（Chrome / Firefox / Safari 最新版）

## 各フェーズの実行方法

```bash
# 例: phase2を動かす
cd phase2
go mod tidy
go run main.go
# ブラウザで http://localhost:8080 を開く
```

---

## 技術スタック

- **バックエンド**: Go + [Echo](https://echo.labstack.com/) (全フェーズ共通)
- **フロントエンド**: ブラウザネイティブ WebRTC API（ライブラリなし）
- **シグナリング**: REST API（WebSocketなし・シンプルポーリング方式）

---

## 重要な視点

> WebRTCは目的ではなく **リアルタイム通信の手段**

ゴールはArkEveなどのアプリケーションへの応用。
本質（P2P通信・NAT越え・シグナリング）を理解することを優先する。

---

## やらないこと

- 映像配信 / 音声通話
- SFU構築
- MediaStream API
