// Phase 2: PeerConnection確立
//
// Phase 1と同じシグナリングAPIを提供しつつ、
// static/ 配下のHTMLファイルも配信する。
//
// 実行方法:
//   go mod tidy
//   go run main.go
//
// 動作確認:
//   ブラウザで http://localhost:8080 (Offerer) と
//              http://localhost:8080/answer.html (Answerer) を
//   それぞれ別タブで開く

package main

import (
	"net/http"
	"sync"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// SignalingStore はシグナリング情報をメモリ上で管理する
type SignalingStore struct {
	mu                 sync.RWMutex
	offer              string
	answer             string
	offererCandidates  []string
	answererCandidates []string
}

var store = &SignalingStore{}

func main() {
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.CORS())

	// 静的ファイルの配信
	// GET /         → static/index.html   (Offerer側)
	// GET /answer.html → static/answer.html (Answerer側)
	e.Static("/", "static")

	// シグナリングAPI（Phase 1と同じ構成）
	e.POST("/offer", handlePostOffer)
	e.GET("/offer", handleGetOffer)
	e.POST("/answer", handlePostAnswer)
	e.GET("/answer", handleGetAnswer)
	e.POST("/candidates", handlePostCandidate)
	e.GET("/candidates", handleGetCandidates)
	e.DELETE("/reset", handleReset)

	e.Logger.Fatal(e.Start(":8080"))
}

type sdpBody struct {
	SDP string `json:"sdp"`
}

type candidateBody struct {
	Candidate string `json:"candidate"`
}

type candidatesResponse struct {
	Candidates []string `json:"candidates"`
}

func handlePostOffer(c echo.Context) error {
	var body sdpBody
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	store.mu.Lock()
	store.offer = body.SDP
	store.mu.Unlock()
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func handleGetOffer(c echo.Context) error {
	store.mu.RLock()
	offer := store.offer
	store.mu.RUnlock()
	if offer == "" {
		return c.NoContent(http.StatusNoContent)
	}
	return c.JSON(http.StatusOK, sdpBody{SDP: offer})
}

func handlePostAnswer(c echo.Context) error {
	var body sdpBody
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	store.mu.Lock()
	store.answer = body.SDP
	store.mu.Unlock()
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func handleGetAnswer(c echo.Context) error {
	store.mu.RLock()
	answer := store.answer
	store.mu.RUnlock()
	if answer == "" {
		return c.NoContent(http.StatusNoContent)
	}
	return c.JSON(http.StatusOK, sdpBody{SDP: answer})
}

func handlePostCandidate(c echo.Context) error {
	side := c.QueryParam("side")
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
	return c.JSON(http.StatusOK, candidatesResponse{Candidates: candidates})
}

func handleReset(c echo.Context) error {
	store.mu.Lock()
	store.offer = ""
	store.answer = ""
	store.offererCandidates = nil
	store.answererCandidates = nil
	store.mu.Unlock()
	return c.JSON(http.StatusOK, map[string]string{"status": "reset"})
}
