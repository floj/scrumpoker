package rooms

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/floj/scrumpoker/pkg/errresp"
	"github.com/floj/scrumpoker/pkg/handler/ws"
	roomt "github.com/floj/scrumpoker/pkg/models/room"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"github.com/olahol/melody"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type RoomsHandler struct {
	mu       *sync.Mutex
	rooms    map[string]*roomt.Room
	maxRooms int
	m        *melody.Melody
}

type JoinRoomRequest struct {
	AuthToken string `json:"authToken"`
	Username  string `json:"username"`
}

type JoinRoomResponse struct {
	PlayerID     string     `json:"playerId"`
	AuthToken    string     `json:"authToken"`
	Username     string     `json:"username"`
	Room         roomt.Room `json:"room"`
	SelectedCard string     `json:"selectedCard"`
}

type CreateRoomResponse struct {
	Name string `json:"name"`
}

type VoteRequest struct {
	Card string `json:"card"`
}

func NewHandler(maxRooms int) (*RoomsHandler, func() error, error) {

	tickerCleanup := time.NewTicker(5 * time.Minute)
	ctx, cancel := context.WithCancel(context.Background())
	m := melody.New()

	m.HandleConnect(func(s *melody.Session) {
		sess, _ := ws.FromSession(s)
		slog.Info("websocket connected", slog.String("remote_addr", s.Request.RemoteAddr), slog.Any("room", sess.RoomName))
	})

	h := &RoomsHandler{
		mu:       &sync.Mutex{},
		rooms:    map[string]*roomt.Room{},
		m:        m,
		maxRooms: maxRooms,
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
		m.Close()
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
	e.GET("/:id/ws", h.EventHub)
}

func (h *RoomsHandler) cleanupRooms() {
	slog.Info("Running cleanup for inactive rooms")

	rooms := h.allRooms()
	rmRooms := []*roomt.Room{}
	checkRooms := []*roomt.Room{}

	now := time.Now()

	for _, r := range rooms {
		r.Do(func(room *roomt.Room) (roomt.PublishEvent, error) {
			if now.Sub(time.Unix(room.UpdatedAt, 0)) > 4*time.Hour {
				rmRooms = append(rmRooms, room)
			} else {
				checkRooms = append(checkRooms, room)
			}
			return roomt.EventRoomNoOp, nil
		})
	}

	if len(rmRooms) > 0 {
		h.mu.Lock()
		for _, r := range rmRooms {
			delete(h.rooms, r.Name)
		}
		h.mu.Unlock()
	}

	for _, r := range checkRooms {
		r.Do(func(room *roomt.Room) (roomt.PublishEvent, error) {
			rmPlayers := []string{}
			for playerID, player := range room.Players {
				if now.Sub(time.Unix(player.UpdatedAt, 0)) > 15*time.Minute {
					rmPlayers = append(rmPlayers, playerID)
				}
			}
			if len(rmPlayers) == 0 {
				return roomt.EventRoomNoOp, nil
			}

			for _, playerID := range rmPlayers {
				delete(room.Players, playerID)
				slog.Info("Removed inactive player from room", slog.String("room", room.Name), slog.String("player_id", playerID))
			}
			return roomt.EventRoomUpdated, nil
		})
	}
}

func (h *RoomsHandler) EventHub(c *echo.Context) error {

	roomName := c.Param("id")

	if !isValidRoomName(roomName) {
		return c.JSON(http.StatusBadRequest, errresp.GenericResp{
			Error: "invalid room name",
		})
	}

	slog.Info("new websocket connection", slog.String("room", roomName), slog.String("remote_addr", c.RealIP()))
	return h.m.HandleRequestWithKeys(c.Response(), c.Request(), map[string]any{
		"wsSessionData": ws.WsSessionData{
			RoomName: roomName,
			PlayerID: "",
		},
	})
}

func (h *RoomsHandler) WithRoomDo(c *echo.Context, roomName string, fn func(player *roomt.Player, room *roomt.Room) (roomt.PublishEvent, error)) error {
	if !isValidRoomName(roomName) {
		return c.JSON(http.StatusBadRequest, errresp.GenericResp{
			Error: "invalid room name",
		})
	}

	h.mu.Lock()
	r, ok := h.rooms[roomName]
	h.mu.Unlock()

	if !ok {
		return c.JSON(http.StatusNotFound, errresp.GenericResp{
			Error: "room not found",
		})
	}

	authToken := c.Request().Header.Get("x-auth-token")
	return r.DoWithPlayer(authToken, fn)
}

func (h *RoomsHandler) GetRoom(c *echo.Context) error {
	roomName := c.Param("id")

	return h.WithRoomDo(c, roomName, func(player *roomt.Player, room *roomt.Room) (roomt.PublishEvent, error) {
		return roomt.EventRoomNoOp, c.JSON(http.StatusOK, room.ToResponse())
	})
}

const allowedRoomNameChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_"

func isValidRoomName(name string) bool {
	if len(name) < 2 {
		return false
	}

	if len(name) > 128 {
		return false
	}

	for _, r := range name {
		if !strings.ContainsRune(allowedRoomNameChars, r) {
			return false
		}
	}
	return true
}

func (h *RoomsHandler) Join(c *echo.Context) error {
	roomName := c.Param("id")

	if !isValidRoomName(roomName) {
		return c.JSON(http.StatusBadRequest, errresp.GenericResp{
			Error: "invalid room name",
		})
	}

	req := &JoinRoomRequest{}
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, errresp.GenericResp{
			Error: "invalid request body",
		})
	}

	if len(req.Username) > 64 {
		return c.JSON(http.StatusBadRequest, errresp.GenericResp{
			Error: "username must be at most 64 characters",
		})
	}

	h.mu.Lock()
	r, exists := h.rooms[roomName]
	if !exists {
		if h.maxRooms > 0 && len(h.rooms) >= h.maxRooms {
			h.mu.Unlock()
			return c.JSON(http.StatusTooManyRequests, errresp.GenericResp{
				Error: "maximum number of rooms reached, please try again later",
			})
		}
		r = roomt.NewRoom(roomName, h.m)
		h.rooms[r.Name] = r
		slog.Info("room not found, created a new one", slog.String("room", r.Name))
	}
	h.mu.Unlock()

	return r.Do(func(room *roomt.Room) (roomt.PublishEvent, error) {
		var player *roomt.Player
		rejoined := false
		if req.AuthToken != "" {
			for _, p := range room.Players {
				if p.Token == req.AuthToken {
					player = p
					rejoined = true
					break
				}
			}
		}
		if player == nil {
			player = &roomt.Player{
				ID:    uuid.New().String(),
				Token: uuid.New().String(),
			}
			room.Players[player.ID] = player
		}

		player.Name = req.Username
		if player.Name == "" {
			player.Name = cases.Title(language.English).String(petname.Generate(2, " "))
		}
		slog.Info("player joined the room", slog.String("room", room.Name), slog.Bool("rejoined", rejoined), slog.String("player_id", player.ID), slog.String("username", player.Name))
		return roomt.EventRoomUpdated, c.JSON(http.StatusOK, JoinRoomResponse{
			PlayerID:     player.ID,
			AuthToken:    player.Token,
			Username:     player.Name,
			SelectedCard: player.Card,
			Room:         room.ToResponse(),
		})
	})
}

func (h *RoomsHandler) CreateRoom(c *echo.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.maxRooms > 0 && len(h.rooms) >= h.maxRooms {
		return c.JSON(http.StatusTooManyRequests, errresp.GenericResp{
			Error: "maximum number of rooms reached, please try again later",
		})
	}
	for range 10 {
		name := petname.Generate(3, "-")
		if _, set := h.rooms[name]; set {
			continue
		}
		r := roomt.NewRoom(name, h.m)
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

	return h.WithRoomDo(c, roomName, func(player *roomt.Player, room *roomt.Room) (roomt.PublishEvent, error) {
		if player == nil {
			return roomt.EventRoomNoOp, c.JSON(http.StatusForbidden, errresp.GenericResp{
				Error: "invalid auth token, player not found in the room",
			})
		}

		room.Revealed = true
		return roomt.EventRoomUpdated, c.NoContent(http.StatusNoContent)
	})
}

func (h *RoomsHandler) Reset(c *echo.Context) error {
	roomName := c.Param("id")

	return h.WithRoomDo(c, roomName, func(player *roomt.Player, room *roomt.Room) (roomt.PublishEvent, error) {
		if player == nil {
			return roomt.EventRoomNoOp, c.JSON(http.StatusForbidden, errresp.GenericResp{
				Error: "invalid auth token, player not found in the room",
			})
		}

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

	return h.WithRoomDo(c, roomName, func(player *roomt.Player, room *roomt.Room) (roomt.PublishEvent, error) {
		if player == nil {
			return roomt.EventRoomNoOp, c.JSON(http.StatusForbidden, errresp.GenericResp{
				Error: "invalid auth token, player not found in the room",
			})
		}

		if req.Card != "" && !slices.Contains(room.AllowedCards, req.Card) {
			return roomt.EventRoomNoOp, c.JSON(http.StatusBadRequest, errresp.GenericResp{
				Error: "invalid card",
			})
		}

		player.Card = req.Card
		return roomt.EventRoomUpdated, c.NoContent(http.StatusNoContent)
	})
}

func (h *RoomsHandler) allRooms() []*roomt.Room {
	rooms := []*roomt.Room{}
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, r := range h.rooms {
		rooms = append(rooms, r)
	}
	return rooms
}

// Saves the Room to a file in JSON format.
func (h *RoomsHandler) SaveRooms(file string) error {
	rooms := h.allRooms()
	rMap := map[string]roomt.Room{}

	for _, r := range rooms {
		r.Do(func(room *roomt.Room) (roomt.PublishEvent, error) {
			rMap[r.Name] = r.Copy()
			return roomt.EventRoomNoOp, nil
		})
	}

	tmp := file + ".tmp"
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0640)
	if err != nil {
		return err
	}

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if encErr := enc.Encode(rMap); encErr != nil {
		f.Close()
		os.Remove(tmp)
		return encErr
	}
	f.Close()
	return os.Rename(tmp, file)
}

func (h *RoomsHandler) LoadRooms(file string) error {
	f, err := os.Open(file)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			slog.Info("Persist file not found, starting with empty rooms", slog.String("file", file))
			return nil
		}
		return err
	}
	defer f.Close()

	rooms := map[string]*roomt.Room{}
	if err := json.NewDecoder(f).Decode(&rooms); err != nil {
		return err
	}

	for _, r := range rooms {
		r.Restore(h.m)
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	h.rooms = rooms
	slog.Info("Rooms loaded from persist file", slog.String("file", file), slog.Int("room_count", len(h.rooms)))
	return nil
}
