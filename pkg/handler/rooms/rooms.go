package rooms

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
	"unicode"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/floj/scrumpoker/pkg/errresp"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"

	"github.com/r3labs/sse/v2"
)

type RoomsHandler struct {
	mu     *sync.Mutex
	ticker *time.Ticker
	rooms  map[string]*Room
	sseSvr *sse.Server
}

func NewHandler() (*RoomsHandler, func() error, error) {
	sseSvr := sse.New()
	sseSvr.AutoReplay = false
	// sseSvr.AutoStream = true
	sseSvr.SplitData = true

	tickerCleanup := time.NewTicker(5 * time.Minute)
	ctx, cancel := context.WithCancel(context.Background())

	h := &RoomsHandler{
		mu:     &sync.Mutex{},
		rooms:  map[string]*Room{},
		sseSvr: sseSvr,
	}

	go func() {
		for {
			select {
			case <-tickerCleanup.C:
				h.cleanupRooms()
			case <-ctx.Done():
				slog.Info("Stopping background goroutine for rooms handler")
				return
			}
		}

	}()

	stopFn := func() error {
		cancel()
		tickerCleanup.Stop()
		sseSvr.Close()
		return nil
	}

	return h, stopFn, nil
}

func (h *RoomsHandler) cleanupRooms() {
	slog.Info("Running cleanup for inactive rooms")
	h.mu.Lock()
	defer h.mu.Unlock()

	for name, room := range h.rooms {
		room.mu.Lock()
		if time.Since(time.Unix(room.UpdatedAt, 0)) > 4*time.Hour {
			delete(h.rooms, name)
			h.sseSvr.RemoveStream(name)
			slog.Info("Removed inactive room", slog.String("room", name))
			continue
		}

		for playerID, player := range room.Players {
			if time.Since(time.Unix(player.UpdatedAt, 0)) > 15*time.Minute {
				delete(room.Players, playerID)
				slog.Info("Removed inactive player from room", slog.String("room", name), slog.String("player_id", playerID))
			}
		}

		room.mu.Unlock()
	}
}

func (h *RoomsHandler) Register(e *echo.Group) {
	e.GET("/debug", h.DebugInfo)
	e.POST("/", h.CreateRoom)
	e.GET("/:id/", h.GetRoom)
	e.POST("/:id/join", h.Join)
	e.POST("/:id/vote", h.Vote)
	e.POST("/:id/reveal", h.Reveal)
	e.POST("/:id/reset", h.Reset)
	e.GET("/sse", h.EventStream)
}

func (h *RoomsHandler) EventStream(c *echo.Context) error {
	req := c.Request()
	if req.Header.Get("Accept") != "text/event-stream" {
		return c.JSON(http.StatusBadRequest, errresp.GenericResp{
			Error: "Accept header must be 'text/event-stream'",
		})
	}
	slog.Info("New SSE connection established", slog.String("remote_addr", c.RealIP()))
	go func() {
		<-req.Context().Done() // Received Browser Disconnection
		slog.Info("Client disconnected, closing SSE connection", slog.String("remote_addr", c.RealIP()))
	}()

	h.sseSvr.ServeHTTP(c.Response(), req)
	return nil
}

func (h *RoomsHandler) GetRoom(c *echo.Context) error {
	roomName := c.Param("id")

	h.mu.Lock()
	room, ok := h.rooms[roomName]
	h.mu.Unlock()
	if !ok {
		return c.JSON(http.StatusNotFound, errresp.GenericResp{
			Error: "room not found",
		})
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	return c.JSON(http.StatusOK, room.ToResponse())
}

type JoinRoomRequest struct {
	PlayerID string `json:"playerId"`
	Username string `json:"username"`
}

type JoinRoomResponse struct {
	PlayerID     string `json:"playerId"`
	Username     string `json:"username"`
	Room         Room   `json:"room"`
	SelectedCard string `json:"selectedCard"`
}

func (h *RoomsHandler) Join(c *echo.Context) error {
	roomName := c.Param("id")

	req := &JoinRoomRequest{}
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, errresp.GenericResp{
			Error: "invalid request body",
		})
	}

	h.mu.Lock()
	room, ok := h.rooms[roomName]
	if !ok {
		room = NewRoom()
		room.Name = roomName
		h.rooms[room.Name] = room
		h.sseSvr.CreateStream(room.Name)
		slog.Info("room not found, created a new one", slog.String("room", roomName))
	}
	h.mu.Unlock()

	room.mu.Lock()
	defer room.mu.Unlock()

	playerID := req.PlayerID
	if playerID == "" {
		playerID = uuid.Must(uuid.NewV7()).String()
	}

	player, exists := room.Players[playerID]
	if !exists {
		player = &Player{}
		room.Players[playerID] = player
	}
	player.Name = req.Username
	if player.Name == "" {
		player.Name = toTitleCase(petname.Generate(2, " "))
	}
	slog.Info("player joined the room", slog.String("room", roomName), slog.Bool("rejoined", exists), slog.String("player_id", playerID), slog.String("username", player.Name))

	room.UpdatedAt = time.Now().Unix()
	h.sseSvr.Publish(room.Name, &sse.Event{
		Data: mustMarshal(SSEMessage{
			Event: "room_updated",
			Data:  room.ToResponse(),
		}),
	})

	return c.JSON(http.StatusOK, JoinRoomResponse{
		PlayerID:     playerID,
		Username:     player.Name,
		Room:         room.ToResponse(),
		SelectedCard: player.Card,
	})
}

func toTitleCase(s string) string {
	if s == "" {
		return s
	}
	parts := strings.Fields(s)
	for i, part := range parts {
		parts[i] = capitalize(part)
	}
	return strings.Join(parts, " ")
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

type SSEMessage struct {
	Event string `json:"eventName"`
	Data  any    `json:"data"`
}

type CreateRoomResponse struct {
	Name string `json:"name"`
}

func (h *RoomsHandler) CreateRoom(c *echo.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	for range 10 {
		r := NewRoom()
		if _, set := h.rooms[r.Name]; set {
			continue
		}
		h.rooms[r.Name] = r
		h.sseSvr.CreateStream(r.Name)
		return c.JSON(http.StatusOK, CreateRoomResponse{
			Name: r.Name,
		})
	}
	return c.JSON(http.StatusInternalServerError, errresp.GenericResp{
		Error: "failed to create a unique room name, please try again",
	})
}

func (h *RoomsHandler) Reveal(c *echo.Context) error {
	roomName := c.Param("id")

	h.mu.Lock()
	room, ok := h.rooms[roomName]
	h.mu.Unlock()
	if !ok {
		return c.JSON(http.StatusNotFound, errresp.GenericResp{
			Error: "room not found",
		})
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	room.Revealed = true
	room.UpdatedAt = time.Now().Unix()

	playerID := c.Request().Header.Get("x-player-id")
	if player, exist := room.Players[playerID]; exist {
		player.UpdatedAt = time.Now().Unix()
	}

	h.sseSvr.Publish(room.Name, &sse.Event{
		Data: mustMarshal(SSEMessage{
			Event: "room_updated",
			Data:  room.ToResponse(),
		}),
	})

	return c.NoContent(http.StatusNoContent)
}

func (h *RoomsHandler) Reset(c *echo.Context) error {
	roomName := c.Param("id")

	h.mu.Lock()
	room, ok := h.rooms[roomName]
	h.mu.Unlock()
	if !ok {
		return c.JSON(http.StatusNotFound, errresp.GenericResp{
			Error: "room not found",
		})
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	for _, p := range room.Players {
		p.Card = ""
	}
	room.Revealed = false
	room.UpdatedAt = time.Now().Unix()

	playerID := c.Request().Header.Get("x-player-id")
	if player, exist := room.Players[playerID]; exist {
		player.UpdatedAt = time.Now().Unix()
	}

	h.sseSvr.Publish(room.Name, &sse.Event{
		Data: mustMarshal(SSEMessage{
			Event: "room_cleared",
			Data:  room.ToResponse(),
		}),
	})

	return c.NoContent(http.StatusNoContent)
}

type VoteRequest struct {
	PlayerID string `json:"playerId"`
	Card     string `json:"card"`
}

func (h *RoomsHandler) Vote(c *echo.Context) error {
	roomName := c.Param("id")

	req := &VoteRequest{}
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, errresp.GenericResp{
			Error: "invalid request body",
		})
	}

	h.mu.Lock()
	room, ok := h.rooms[roomName]
	h.mu.Unlock()
	if !ok {
		return c.JSON(http.StatusNotFound, errresp.GenericResp{
			Error: "room not found",
		})
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	player, ok := room.Players[req.PlayerID]
	if !ok {
		return c.JSON(http.StatusNotFound, errresp.GenericResp{
			Error: "player not found in the room",
		})
	}

	player.Card = req.Card
	player.UpdatedAt = time.Now().Unix()
	room.UpdatedAt = time.Now().Unix()

	h.sseSvr.Publish(room.Name, &sse.Event{
		Data: mustMarshal(SSEMessage{
			Event: "room_updated",
			Data:  room.ToResponse(),
		}),
	})

	return c.NoContent(http.StatusNoContent)
}

func (h *RoomsHandler) DebugInfo(c *echo.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	return c.JSONPretty(http.StatusOK, h.rooms, "  ")
}

func mustMarshal(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

// Saves the Room to a file in JSON format.
func (h *RoomsHandler) SaveRooms(file string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Lock each room to ensure no updates happen while saving
	for _, r := range h.rooms {
		r.mu.Lock()
		defer r.mu.Unlock()
	}

	os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	f, err := os.Create(file)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(h.rooms)

}

func (h *RoomsHandler) LoadRooms(file string) error {
	f, err := os.Open(file)
	if err != nil {
		if os.IsNotExist(err) {
			slog.Info("Persist file not found, starting with empty rooms", slog.String("file", file))
			return nil
		}
		return err
	}
	defer f.Close()

	rooms := map[string]*Room{}
	if err := json.NewDecoder(f).Decode(&rooms); err != nil {
		return err
	}

	for _, r := range rooms {
		r.mu = &sync.Mutex{}
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.rooms = rooms
	for _, r := range h.rooms {
		h.sseSvr.CreateStream(r.Name)
	}
	slog.Info("Rooms loaded from persist file", slog.String("file", file), slog.Int("room_count", len(h.rooms)))

	return nil
}
