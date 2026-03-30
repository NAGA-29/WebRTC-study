// Phase 4: NAT越え（STUN/TURN設定付きシグナリングサーバー）
//
// Phase 3と同じシグナリング機能に加えて、
// GET /config エンドポイントでICEサーバー設定をブラウザに提供する。
//
// TURN設定は環境変数で注入できる:
//   TURN_URL=turn:your-turn-server:3478
//   TURN_USER=username
//   TURN_PASS=password
//
// 実行例（STUNのみ）:
//   go run main.go
//
// 実行例（TURN付き）:
//   TURN_URL=turn:coturn.example.com:3478 TURN_USER=user TURN_PASS=pass go run main.go

package main

import (
	"net/http"
	"os"
	"sync"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type SignalingStore struct {
	mu                 sync.RWMutex
	offer              string
	answer             string
	offererCandidates  []string
	answererCandidates []string
}

var store = &SignalingStore{}

// ICEServerConfig はブラウザに渡すICEサーバーの設定
type ICEServerConfig struct {
	URLs       string `json:"urls"`
	Username   string `json:"username,omitempty"`   // TURNのみ必要
	Credential string `json:"credential,omitempty"` // TURNのみ必要
}

func main() {
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.CORS())

	e.Static("/", "static")

	// ICEサーバー設定をブラウザに提供するエンドポイント
	e.GET("/config", handleGetConfig)

	e.POST("/offer", handlePostOffer)
	e.GET("/offer", handleGetOffer)
	e.POST("/answer", handlePostAnswer)
	e.GET("/answer", handleGetAnswer)
	e.POST("/candidates", handlePostCandidate)
	e.GET("/candidates", handleGetCandidates)
	e.DELETE("/reset", handleReset)

	e.Logger.Fatal(e.Start(":8080"))
}

// GET /config - ブラウザに RTCPeerConnection 用のICEサーバー設定を返す
func handleGetConfig(c echo.Context) error {
	// 必ず無料の公開STUNサーバーを含める
	servers := []ICEServerConfig{
		{URLs: "stun:stun.l.google.com:19302"},
	}

	// 環境変数にTURN設定があれば追加する
	turnURL := os.Getenv("TURN_URL")
	turnUser := os.Getenv("TURN_USER")
	turnPass := os.Getenv("TURN_PASS")
	if turnURL != "" {
		servers = append(servers, ICEServerConfig{
			URLs:       turnURL,
			Username:   turnUser,
			Credential: turnPass,
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"iceServers": servers,
	})
}

// 以下 Phase 3 と同じ実装

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
