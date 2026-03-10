package rooms

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/floj/scrumpoker/pkg/errresp"
	"github.com/labstack/echo/v5"

	"github.com/r3labs/sse/v2"
)

type RoomsHandler struct {
	mu     *sync.Mutex
	ticker *time.Ticker
	rooms  map[string]*Room
	sseSvr *sse.Server
	ctx    context.Context
}

func NewHandler(ctx context.Context) (*RoomsHandler, func() error, error) {
	sseSvr := sse.New()
	sseSvr.AutoReplay = false
	// sseSvr.AutoStream = true
	sseSvr.SplitData = true

	tickerCleanup := time.NewTicker(5 * time.Minute)
	ctx, cancel := context.WithCancel(ctx)

	h := &RoomsHandler{
		mu:     &sync.Mutex{},
		rooms:  map[string]*Room{},
		sseSvr: sseSvr,
		ctx:    ctx,
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
		if time.Since(time.Unix(room.UpdatedAt, 0)) > 4*time.Hour {
			room.Stop()
			delete(h.rooms, name)
			slog.Info("Removed inactive room", slog.String("room", name))
			continue
		}
		room.Cleanup()
	}
}

func (h *RoomsHandler) Register(e *echo.Group) {
	e.POST("", h.CreateRoom)
	e.GET("/:id", h.GetRoom)
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

	room.GetRoom(func(r Room) {
		c.JSON(http.StatusOK, r)
	})

	return nil
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
		room = NewRoom(roomName)
		h.rooms[room.Name] = room
		room.Start(h.ctx, h.sseSvr)
		slog.Info("room not found, created a new one", slog.String("room", roomName))
	}
	h.mu.Unlock()

	room.AddPlayer(req.PlayerID, req.Username, func(room Room, playerID string, player Player) {
		c.JSON(http.StatusOK, JoinRoomResponse{
			Room:         room.toResponse(),
			PlayerID:     playerID,
			Username:     player.Name,
			SelectedCard: player.Card,
		})
	})
	return nil
}

type SSEMessage struct {
	Event RoomEvent `json:"eventName"`
	Data  any       `json:"data"`
}

type CreateRoomResponse struct {
	Name string `json:"name"`
}

func (h *RoomsHandler) CreateRoom(c *echo.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	for range 10 {
		r := NewRoom("")
		if _, set := h.rooms[r.Name]; set {
			continue
		}
		h.rooms[r.Name] = r
		r.Start(h.ctx, h.sseSvr)
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

	playerID := c.Request().Header.Get("x-player-id")
	room.Reveal(playerID)

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

	playerID := c.Request().Header.Get("x-player-id")
	room.Reset(playerID)

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
	room.Vote(req.PlayerID, req.Card, func(result VoteResult) {
		switch result {
		case VoteSuccess:
			// all good
			c.NoContent(http.StatusNoContent)
		case VoteNotFound:
			c.JSON(http.StatusNotFound, errresp.GenericResp{
				Error: "player not found in room",
			})
		case VoteInvalid:
			c.JSON(http.StatusBadRequest, errresp.GenericResp{
				Error: "invalid card selection",
			})
		default:
			c.JSON(http.StatusInternalServerError, errresp.GenericResp{
				Error: "unknown vote result: " + string(result),
			})
		}
	})
	return nil
}

// Saves the Room to a file in JSON format.
func (h *RoomsHandler) SaveRooms(file string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	f, err := os.Create(file)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	errs := []error{}

	for _, r := range h.rooms {
		slog.Info("saving room", slog.String("room", r.Name))
		r.Save(enc, func(encErr error) {
			if encErr != nil {
				errs = append(errs, encErr)
			}
		})
	}

	return errors.Join(errs...)
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
	dec := json.NewDecoder(f)
	errs := []error{}
	for {
		room := &Room{}
		err := dec.Decode(room)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			errs = append(errs, fmt.Errorf("failed to decode room from persist file: %w", err))
		}
		rooms[room.Name] = room
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	h.rooms = rooms
	for _, r := range h.rooms {
		r.Start(h.ctx, h.sseSvr)
	}
	slog.Info("Rooms loaded from persist file", slog.String("file", file), slog.Int("room_count", len(h.rooms)))

	return errors.Join(errs...)
}
