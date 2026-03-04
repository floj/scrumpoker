package rooms

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

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
	tickerPing := time.NewTicker(5 * time.Second)
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
			case <-tickerPing.C:
				h.pingRooms()
			case <-ctx.Done():
				slog.Info("Stopping background goroutine for rooms handler")
				return
			}
		}

	}()

	stopFn := func() error {
		cancel()
		tickerCleanup.Stop()
		tickerPing.Stop()
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
		}
		room.mu.Unlock()
	}
}

func (h *RoomsHandler) pingRooms() {
	slog.Info("Pinging all rooms")
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, room := range h.rooms {
		room.mu.Lock()
		h.sseSvr.Publish(room.Name, &sse.Event{
			Data: mustMarshal(SSEMessage{
				Event: "room_updated",
				Data:  room.ToResponse(),
			}),
		})
		room.mu.Unlock()
	}
}

func (h *RoomsHandler) Register(e *echo.Group) {
	e.POST("/", h.CreateRoom)
	e.POST("/:id/join", h.JoinRoom)
	e.POST("/:id/vote", h.Vote)
	e.POST("/:id/reveal", h.Reveal)
	e.DELETE("/:id/", h.Clear)
	e.GET("/sse", func(c *echo.Context) error {
		slog.Info("New SSE connection established", slog.String("remote_addr", c.RealIP()))
		go func() {
			<-c.Request().Context().Done() // Received Browser Disconnection
			slog.Info("Client disconnected, closing SSE connection", slog.String("remote_addr", c.RealIP()))
		}()

		h.sseSvr.ServeHTTP(c.Response(), c.Request())
		return nil
	})
}

type JoinRoomRequest struct {
	PlayerID string `json:"playerId"`
	Username string `json:"username"`
}

func (h *RoomsHandler) JoinRoom(c *echo.Context) error {
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
	if exists {
		slog.Info("Player re-joined the room", slog.String("room", roomName), slog.String("player_id", playerID), slog.String("username", req.Username))
		player.Name = req.Username
	} else {
		slog.Info("New player joined the room", slog.String("room", roomName), slog.String("player_id", playerID), slog.String("username", req.Username))
		player = &Player{
			Name: req.Username,
		}
		room.Players[playerID] = player
	}

	room.UpdatedAt = time.Now().Unix()
	h.sseSvr.Publish(room.Name, &sse.Event{
		Data: mustMarshal(SSEMessage{
			Event: "room_updated",
			Data:  room.ToResponse(),
		}),
	})

	return c.JSON(http.StatusOK, map[string]any{
		"playerId":     playerID,
		"room":         room.ToResponse(),
		"selectedCard": player.Card,
	})
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

	h.sseSvr.Publish(room.Name, &sse.Event{
		Data: mustMarshal(SSEMessage{
			Event: "room_updated",
			Data:  room.ToResponse(),
		}),
	})

	return c.NoContent(http.StatusNoContent)
}

func (h *RoomsHandler) Clear(c *echo.Context) error {
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

	h.sseSvr.Publish(room.Name, &sse.Event{
		Data: mustMarshal(SSEMessage{
			Event: "room_cleared",
			Data:  room.ToResponse(),
		}),
	})

	return c.NoContent(http.StatusNoContent)
}

func (h *RoomsHandler) Vote(c *echo.Context) error {
	roomName := c.Param("id")

	type VoteRequest struct {
		PlayerID string `json:"playerId"`
		Card     string `json:"card"`
	}

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
	room.Players[req.PlayerID].Card = req.Card
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
	return c.JSON(http.StatusOK, h.rooms)
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
