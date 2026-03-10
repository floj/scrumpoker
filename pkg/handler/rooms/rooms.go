package rooms

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
	"unicode"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/floj/scrumpoker/pkg/errresp"
	roomt "github.com/floj/scrumpoker/pkg/models/room"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"

	"github.com/r3labs/sse/v2"
)

type RoomsHandler struct {
	mu     *sync.Mutex
	ticker *time.Ticker
	rooms  map[string]*roomt.Room
	sseSvr *sse.Server
}

type JoinRoomRequest struct {
	PlayerID string `json:"playerId"`
	Username string `json:"username"`
}

type JoinRoomResponse struct {
	PlayerID     string     `json:"playerId"`
	Username     string     `json:"username"`
	Room         roomt.Room `json:"room"`
	SelectedCard string     `json:"selectedCard"`
}

type CreateRoomResponse struct {
	Name string `json:"name"`
}

type VoteRequest struct {
	PlayerID string `json:"playerId"`
	Card     string `json:"card"`
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
		rooms:  map[string]*roomt.Room{},
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

func (h *RoomsHandler) Register(e *echo.Group) {
	e.POST("", h.CreateRoom)
	e.GET("/:id", h.GetRoom)
	e.POST("/:id/join", h.Join)
	e.POST("/:id/vote", h.Vote)
	e.POST("/:id/reveal", h.Reveal)
	e.POST("/:id/reset", h.Reset)
	e.GET("/sse", h.EventStream)
}

func (h *RoomsHandler) cleanupRooms() {
	slog.Info("Running cleanup for inactive rooms")
	h.mu.Lock()
	defer h.mu.Unlock()

	rmRooms := []string{}

	for _, r := range h.rooms {
		r.Do("", func(player *roomt.Player, room *roomt.Room) (roomt.PublishEvent, error) {
			if time.Since(time.Unix(room.UpdatedAt, 0)) > 4*time.Hour {
				rmRooms = append(rmRooms, room.Name)
				return roomt.EventRoomNoOp, nil
			}

			rmPlayers := []string{}
			for playerID, player := range room.Players {
				if time.Since(time.Unix(player.UpdatedAt, 0)) > 15*time.Minute {
					rmPlayers = append(rmPlayers, playerID)
				}
			}
			for _, playerID := range rmPlayers {
				delete(room.Players, playerID)
				slog.Info("Removed inactive player from room", slog.String("room", room.Name), slog.String("player_id", playerID))
			}
			return roomt.EventRoomNoOp, nil
		})

	}

	for _, name := range rmRooms {
		h.sseSvr.RemoveStream(name)
		delete(h.rooms, name)
	}
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

func (h *RoomsHandler) WithRoomDo(c *echo.Context, roomName string, playerID string, fn func(player *roomt.Player, room *roomt.Room) (roomt.PublishEvent, error)) error {
	h.mu.Lock()
	r, ok := h.rooms[roomName]
	h.mu.Unlock()

	if !ok {
		return c.JSON(http.StatusNotFound, errresp.GenericResp{
			Error: "room not found",
		})
	}

	if playerID == "" {
		playerID = c.Request().Header.Get("x-player-id")
	}

	return r.Do(playerID, fn)
}

func (h *RoomsHandler) GetRoom(c *echo.Context) error {
	roomName := c.Param("id")

	return h.WithRoomDo(c, roomName, "", func(player *roomt.Player, room *roomt.Room) (roomt.PublishEvent, error) {
		return roomt.EventRoomNoOp, c.JSON(http.StatusOK, room.ToResponse())
	})
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
	r, ok := h.rooms[roomName]
	if !ok {
		r = roomt.NewRoom()
		r.Name = roomName
		r.Init(h.sseSvr)
		h.rooms[r.Name] = r
		slog.Info("room not found, created a new one", slog.String("room", r.Name))
	}
	h.mu.Unlock()

	playerID := req.PlayerID
	if playerID == "" {
		playerID = uuid.New().String()
	}

	return r.Do(playerID, func(player *roomt.Player, room *roomt.Room) (roomt.PublishEvent, error) {
		exists := player != nil
		if !exists {
			player = &roomt.Player{}
			room.Players[playerID] = player
		}
		player.Name = req.Username
		if player.Name == "" {
			player.Name = toTitleCase(petname.Generate(2, " "))
		}
		slog.Info("player joined the room", slog.String("room", room.Name), slog.Bool("rejoined", exists), slog.String("player_id", playerID), slog.String("username", player.Name))
		return roomt.EventRoomUpdated, c.JSON(http.StatusOK, JoinRoomResponse{
			PlayerID:     playerID,
			Username:     player.Name,
			Room:         room.ToResponse(),
			SelectedCard: player.Card,
		})
	})
}

func (h *RoomsHandler) CreateRoom(c *echo.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	for range 10 {
		r := roomt.NewRoom()
		if _, set := h.rooms[r.Name]; set {
			continue
		}
		r.Init(h.sseSvr)
		h.rooms[r.Name] = r

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

	return h.WithRoomDo(c, roomName, "", func(player *roomt.Player, room *roomt.Room) (roomt.PublishEvent, error) {
		room.Revealed = true
		return roomt.EventRoomUpdated, c.NoContent(http.StatusNoContent)
	})
}

func (h *RoomsHandler) Reset(c *echo.Context) error {
	roomName := c.Param("id")

	return h.WithRoomDo(c, roomName, "", func(player *roomt.Player, room *roomt.Room) (roomt.PublishEvent, error) {
		for _, p := range room.Players {
			p.Card = ""
		}
		room.Revealed = false
		return roomt.EventRoomCleared, c.NoContent(http.StatusNoContent)
	})
}

func (h *RoomsHandler) Vote(c *echo.Context) error {

	req := &VoteRequest{}
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, errresp.GenericResp{
			Error: "invalid request body",
		})
	}
	roomName := c.Param("id")

	return h.WithRoomDo(c, roomName, req.PlayerID, func(player *roomt.Player, room *roomt.Room) (roomt.PublishEvent, error) {
		if player == nil {
			return roomt.EventRoomNoOp, c.JSON(http.StatusNotFound, errresp.GenericResp{
				Error: "player not found in the room",
			})
		}
		player.Card = req.Card
		return roomt.EventRoomUpdated, c.NoContent(http.StatusNoContent)
	})
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
		err := r.Do("", func(player *roomt.Player, rr *roomt.Room) (roomt.PublishEvent, error) {
			slog.Info("Saving room", slog.String("room", rr.Name), slog.Int("player_count", len(rr.Players)))
			return roomt.EventRoomNoOp, enc.Encode(rr)
		})
		errs = append(errs, err)
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

	rooms := map[string]*roomt.Room{}

	dec := json.NewDecoder(f)
	errs := []error{}
	for {
		r := &roomt.Room{}
		if err := dec.Decode(r); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			errs = append(errs, err)
			continue
		}
		r.Init(h.sseSvr)
		rooms[r.Name] = r
		slog.Info("Loaded room", slog.String("room", r.Name), slog.Int("player_count", len(r.Players)))
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	h.rooms = rooms
	slog.Info("Rooms loaded from persist file", slog.String("file", file), slog.Int("room_count", len(h.rooms)))
	return nil
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
