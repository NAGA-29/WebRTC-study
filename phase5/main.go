// Phase 5: ArkEve応用 - 座標同期サーバー
//
// ルームID付きのシグナリングサーバー。
// 複数のルームを同時に管理できる（Phase 5では1ルームで十分）。
//
// 実行方法:
//   go mod tidy
//   go run main.go
//
// ブラウザで以下を開く:
//   http://localhost:8080/?role=offerer&room=room1
//   http://localhost:8080/?role=answerer&room=room1

package main

import (
	"net/http"
	"sync"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// Room は1つの接続セッションの状態を保持する
type Room struct {
	mu                 sync.RWMutex
	offer              string
	answer             string
	offererCandidates  []string
	answererCandidates []string
}

// RoomStore は複数のルームを管理する
type RoomStore struct {
	mu    sync.RWMutex
	rooms map[string]*Room
}

func (rs *RoomStore) getOrCreate(roomID string) *Room {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	if room, ok := rs.rooms[roomID]; ok {
		return room
	}
	room := &Room{}
	rs.rooms[roomID] = room
	return room
}

var roomStore = &RoomStore{rooms: make(map[string]*Room)}

func main() {
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.CORS())

	// 静的ファイル配信
	e.Static("/", "static")

	// ルームスコープのシグナリングAPI（:roomId パラメータ付き）
	e.POST("/rooms/:roomId/offer", handlePostOffer)
	e.GET("/rooms/:roomId/offer", handleGetOffer)
	e.POST("/rooms/:roomId/answer", handlePostAnswer)
	e.GET("/rooms/:roomId/answer", handleGetAnswer)
	e.POST("/rooms/:roomId/candidates", handlePostCandidate)
	e.GET("/rooms/:roomId/candidates", handleGetCandidates)
	e.DELETE("/rooms/:roomId/reset", handleReset)

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
	room := roomStore.getOrCreate(c.Param("roomId"))
	var body sdpBody
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	room.mu.Lock()
	room.offer = body.SDP
	room.mu.Unlock()
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func handleGetOffer(c echo.Context) error {
	room := roomStore.getOrCreate(c.Param("roomId"))
	room.mu.RLock()
	offer := room.offer
	room.mu.RUnlock()
	if offer == "" {
		return c.NoContent(http.StatusNoContent)
	}
	return c.JSON(http.StatusOK, sdpBody{SDP: offer})
}

func handlePostAnswer(c echo.Context) error {
	room := roomStore.getOrCreate(c.Param("roomId"))
	var body sdpBody
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	room.mu.Lock()
	room.answer = body.SDP
	room.mu.Unlock()
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func handleGetAnswer(c echo.Context) error {
	room := roomStore.getOrCreate(c.Param("roomId"))
	room.mu.RLock()
	answer := room.answer
	room.mu.RUnlock()
	if answer == "" {
		return c.NoContent(http.StatusNoContent)
	}
	return c.JSON(http.StatusOK, sdpBody{SDP: answer})
}

func handlePostCandidate(c echo.Context) error {
	room := roomStore.getOrCreate(c.Param("roomId"))
	side := c.QueryParam("side")
	var body candidateBody
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	room.mu.Lock()
	if side == "answerer" {
		room.answererCandidates = append(room.answererCandidates, body.Candidate)
	} else {
		room.offererCandidates = append(room.offererCandidates, body.Candidate)
	}
	room.mu.Unlock()
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func handleGetCandidates(c echo.Context) error {
	room := roomStore.getOrCreate(c.Param("roomId"))
	side := c.QueryParam("side")
	room.mu.RLock()
	var candidates []string
	if side == "answerer" {
		candidates = make([]string, len(room.answererCandidates))
		copy(candidates, room.answererCandidates)
	} else {
		candidates = make([]string, len(room.offererCandidates))
		copy(candidates, room.offererCandidates)
	}
	room.mu.RUnlock()
	return c.JSON(http.StatusOK, candidatesResponse{Candidates: candidates})
}

func handleReset(c echo.Context) error {
	roomID := c.Param("roomId")
	roomStore.mu.Lock()
	roomStore.rooms[roomID] = &Room{}
	roomStore.mu.Unlock()
	return c.JSON(http.StatusOK, map[string]string{"status": "reset"})
}
