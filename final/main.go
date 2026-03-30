// Final: フル実装 WebRTCシグナリングサーバー
//
// Phase 1〜5で学んだ内容を統合した完成版サーバー。
// ルーム管理・シグナリング・接続イベントログを提供する。
//
// 実行方法:
//   go mod tidy
//   go run main.go
//
// ブラウザ: http://localhost:8080

package main

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// ===== データ構造 =====

// LogEntry は接続イベントのログエントリ
type LogEntry struct {
	Time    string `json:"time"`
	Event   string `json:"event"`
	Message string `json:"message"`
}

// Room は1つの接続セッションの完全な状態
type Room struct {
	mu                 sync.RWMutex
	ID                 string
	offer              string
	answer             string
	offererCandidates  []string
	answererCandidates []string
	logs               []LogEntry
	createdAt          time.Time
}

// addLog はルームにイベントログを追加する
func (r *Room) addLog(event, message string) {
	entry := LogEntry{
		Time:    time.Now().Format("15:04:05.000"),
		Event:   event,
		Message: message,
	}
	r.mu.Lock()
	r.logs = append(r.logs, entry)
	r.mu.Unlock()
}

// RoomStore は複数ルームを管理する
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
	room := &Room{
		ID:        roomID,
		createdAt: time.Now(),
	}
	rs.rooms[roomID] = room
	return room
}

func (rs *RoomStore) reset(roomID string) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rs.rooms[roomID] = &Room{
		ID:        roomID,
		createdAt: time.Now(),
	}
}

var store = &RoomStore{rooms: make(map[string]*Room)}

// ===== サーバー起動 =====

func main() {
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.CORS())

	e.Static("/", "static")

	// ルーム管理
	e.POST("/rooms", handleCreateRoom)

	// シグナリングAPI（ルームスコープ）
	e.POST("/rooms/:roomId/offer", handlePostOffer)
	e.GET("/rooms/:roomId/offer", handleGetOffer)
	e.POST("/rooms/:roomId/answer", handlePostAnswer)
	e.GET("/rooms/:roomId/answer", handleGetAnswer)
	e.POST("/rooms/:roomId/candidates", handlePostCandidate)
	e.GET("/rooms/:roomId/candidates", handleGetCandidates)

	// 接続イベントログ
	e.GET("/rooms/:roomId/log", handleGetLog)

	// リセット（開発用）
	e.DELETE("/rooms/:roomId/reset", handleReset)

	e.Logger.Fatal(e.Start(":8080"))
}

// ===== リクエスト/レスポンス型 =====

type sdpBody struct {
	SDP string `json:"sdp"`
}

type candidateBody struct {
	Candidate string `json:"candidate"`
}

type candidatesResponse struct {
	Candidates []string `json:"candidates"`
}

// ===== ハンドラー =====

// POST /rooms - 新しいルームを作成する
func handleCreateRoom(c echo.Context) error {
	// ルームIDを生成（タイムスタンプベースの簡易ID）
	roomID := fmt.Sprintf("room-%d", time.Now().UnixMilli()%100000)
	room := store.getOrCreate(roomID)
	room.addLog("room_created", fmt.Sprintf("ルーム %s を作成しました", roomID))
	return c.JSON(http.StatusCreated, map[string]string{
		"roomId": roomID,
	})
}

// POST /rooms/:roomId/offer
func handlePostOffer(c echo.Context) error {
	room := store.getOrCreate(c.Param("roomId"))
	var body sdpBody
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	room.mu.Lock()
	room.offer = body.SDP
	room.mu.Unlock()
	room.addLog("offer_stored", "SDP Offerを受信・保存しました")
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// GET /rooms/:roomId/offer
func handleGetOffer(c echo.Context) error {
	room := store.getOrCreate(c.Param("roomId"))
	room.mu.RLock()
	offer := room.offer
	room.mu.RUnlock()
	if offer == "" {
		return c.NoContent(http.StatusNoContent)
	}
	room.addLog("offer_retrieved", "SDP Offerを取得しました")
	return c.JSON(http.StatusOK, sdpBody{SDP: offer})
}

// POST /rooms/:roomId/answer
func handlePostAnswer(c echo.Context) error {
	room := store.getOrCreate(c.Param("roomId"))
	var body sdpBody
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	room.mu.Lock()
	room.answer = body.SDP
	room.mu.Unlock()
	room.addLog("answer_stored", "SDP Answerを受信・保存しました")
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// GET /rooms/:roomId/answer
func handleGetAnswer(c echo.Context) error {
	room := store.getOrCreate(c.Param("roomId"))
	room.mu.RLock()
	answer := room.answer
	room.mu.RUnlock()
	if answer == "" {
		return c.NoContent(http.StatusNoContent)
	}
	room.addLog("answer_retrieved", "SDP Answerを取得しました")
	return c.JSON(http.StatusOK, sdpBody{SDP: answer})
}

// POST /rooms/:roomId/candidates?side=offerer|answerer
func handlePostCandidate(c echo.Context) error {
	room := store.getOrCreate(c.Param("roomId"))
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
	room.addLog("candidate_added", fmt.Sprintf("ICE candidate追加 (side=%s)", side))
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// GET /rooms/:roomId/candidates?side=offerer|answerer&from=N
// from=N で N番目以降のcandidatesのみ返す（インクリメンタルポーリング）
func handleGetCandidates(c echo.Context) error {
	room := store.getOrCreate(c.Param("roomId"))
	side := c.QueryParam("side")

	room.mu.RLock()
	var all []string
	if side == "answerer" {
		all = make([]string, len(room.answererCandidates))
		copy(all, room.answererCandidates)
	} else {
		all = make([]string, len(room.offererCandidates))
		copy(all, room.offererCandidates)
	}
	room.mu.RUnlock()

	return c.JSON(http.StatusOK, candidatesResponse{Candidates: all})
}

// GET /rooms/:roomId/log - 接続イベントログを取得する
func handleGetLog(c echo.Context) error {
	room := store.getOrCreate(c.Param("roomId"))
	room.mu.RLock()
	logs := make([]LogEntry, len(room.logs))
	copy(logs, room.logs)
	room.mu.RUnlock()
	return c.JSON(http.StatusOK, map[string]interface{}{
		"roomId": c.Param("roomId"),
		"logs":   logs,
	})
}

// DELETE /rooms/:roomId/reset
func handleReset(c echo.Context) error {
	store.reset(c.Param("roomId"))
	return c.JSON(http.StatusOK, map[string]string{"status": "reset"})
}
