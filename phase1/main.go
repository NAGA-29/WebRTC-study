// Phase 1: シグナリングサーバー
//
// WebRTCの接続情報（SDP Offer/Answer・ICE Candidate）を
// メモリ上で保管・配信するREST APIサーバー。
//
// このフェーズではブラウザのWebRTC接続は発生しない。
// curlで各エンドポイントを叩いて、シグナリングの仕組みを理解する。
//
// 実行方法:
//   go mod tidy
//   go run main.go
//
// 動作確認:
//   curl -X POST http://localhost:8080/offer \
//     -H "Content-Type: application/json" \
//     -d '{"sdp":"test-offer-sdp"}'
//   curl http://localhost:8080/offer

package main

import (
	"net/http"
	"sync"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// SignalingStore はシグナリング情報をメモリ上で管理する。
// 実際のWebRTCアプリではDBやRedisを使うが、学習用途なのでメモリのみ。
type SignalingStore struct {
	mu                 sync.RWMutex // 並行アクセス保護
	offer              string       // SDP Offer（Offerer側が作成）
	answer             string       // SDP Answer（Answerer側が作成）
	offererCandidates  []string     // Offerer側のICE candidates
	answererCandidates []string     // Answerer側のICE candidates
}

var store = &SignalingStore{}

func main() {
	e := echo.New()
	e.Use(middleware.Logger()) // リクエストログ
	e.Use(middleware.CORS())   // ブラウザからのCORSを許可

	// 静的ファイル配信（テスト用UI）
	e.Static("/", "static")

	// --- シグナリングAPI ---
	// SDP Offer
	e.POST("/offer", handlePostOffer)
	e.GET("/offer", handleGetOffer)

	// SDP Answer
	e.POST("/answer", handlePostAnswer)
	e.GET("/answer", handleGetAnswer)

	// ICE Candidates（?side=offerer または ?side=answerer）
	e.POST("/candidates", handlePostCandidate)
	e.GET("/candidates", handleGetCandidates)

	// サーバーリセット（開発用）
	e.DELETE("/reset", handleReset)

	e.Logger.Fatal(e.Start(":8080"))
}

// --- リクエスト/レスポンスの型定義 ---

type sdpBody struct {
	SDP string `json:"sdp"` // SDP文字列（JSON.stringify(offer) の値）
}

type candidateBody struct {
	Candidate string `json:"candidate"` // ICE candidate（JSON.stringify(candidate) の値）
}

type candidatesResponse struct {
	Candidates []string `json:"candidates"`
	Total      int      `json:"total"`
}

// --- ハンドラー ---

// POST /offer - SDP OfferをサーバーにPOSTする
func handlePostOffer(c echo.Context) error {
	var body sdpBody
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	store.mu.Lock()
	store.offer = body.SDP
	store.mu.Unlock()
	c.Logger().Infof("Offer stored (%d bytes)", len(body.SDP))
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// GET /offer - 保存済みSDP Offerを取得する
func handleGetOffer(c echo.Context) error {
	store.mu.RLock()
	offer := store.offer
	store.mu.RUnlock()
	if offer == "" {
		// まだOfferが来ていない → 204 No Content
		return c.NoContent(http.StatusNoContent)
	}
	return c.JSON(http.StatusOK, sdpBody{SDP: offer})
}

// POST /answer - SDP AnswerをサーバーにPOSTする
func handlePostAnswer(c echo.Context) error {
	var body sdpBody
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	store.mu.Lock()
	store.answer = body.SDP
	store.mu.Unlock()
	c.Logger().Infof("Answer stored (%d bytes)", len(body.SDP))
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// GET /answer - 保存済みSDP Answerを取得する
func handleGetAnswer(c echo.Context) error {
	store.mu.RLock()
	answer := store.answer
	store.mu.RUnlock()
	if answer == "" {
		// まだAnswerが来ていない → 204 No Content
		return c.NoContent(http.StatusNoContent)
	}
	return c.JSON(http.StatusOK, sdpBody{SDP: answer})
}

// POST /candidates?side=offerer|answerer - ICE candidateを追加する
func handlePostCandidate(c echo.Context) error {
	side := c.QueryParam("side") // "offerer" または "answerer"
	var body candidateBody
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	store.mu.Lock()
	if side == "answerer" {
		store.answererCandidates = append(store.answererCandidates, body.Candidate)
	} else {
		store.offererCandidates = append(store.offererCandidates, body.Candidate)
	}
	store.mu.Unlock()
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// GET /candidates?side=offerer|answerer - ICE candidatesを取得する
func handleGetCandidates(c echo.Context) error {
	side := c.QueryParam("side")
	store.mu.RLock()
	var candidates []string
	if side == "answerer" {
		candidates = make([]string, len(store.answererCandidates))
		copy(candidates, store.answererCandidates)
	} else {
		candidates = make([]string, len(store.offererCandidates))
		copy(candidates, store.offererCandidates)
	}
	store.mu.RUnlock()
	return c.JSON(http.StatusOK, candidatesResponse{
		Candidates: candidates,
		Total:      len(candidates),
	})
}

// DELETE /reset - ストアをリセットする（開発・テスト用）
func handleReset(c echo.Context) error {
	store.mu.Lock()
	store.offer = ""
	store.answer = ""
	store.offererCandidates = nil
	store.answererCandidates = nil
	store.mu.Unlock()
	return c.JSON(http.StatusOK, map[string]string{"status": "reset"})
}
