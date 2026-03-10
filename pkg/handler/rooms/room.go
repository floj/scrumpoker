package rooms

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/google/uuid"
	"github.com/r3labs/sse/v2"
)

type RoomEvent string
type VoteResult string

const (
	RoomUpdated RoomEvent = "room_updated"
	RoomCleared RoomEvent = "room_cleared"
	RoomNoOp    RoomEvent = "noop"

	VoteSuccess  VoteResult = "vote_success"
	VoteFailed   VoteResult = "vote_failed"
	VoteInvalid  VoteResult = "vote_invalid"
	VoteNotFound VoteResult = "player_not_found"
)

type roomCallback func(*Room) RoomEvent

type Room struct {
	Name         string             `json:"name"`
	CreatedAt    int64              `json:"createdAt"`
	UpdatedAt    int64              `json:"updatedAt"`
	Players      map[string]*Player `json:"players"`
	AllowedCards []string           `json:"allowedCards"`
	Revealed     bool               `json:"revealed"`

	c      chan roomCallback
	cancel context.CancelFunc
}

type Player struct {
	Name      string `json:"name"`
	Card      string `json:"card"`
	Voted     bool   `json:"voted"`
	UpdatedAt int64  `json:"-"`
}

func (p *Player) updateTimestamp() {
	p.UpdatedAt = time.Now().Unix()
}

func (r *Room) Start(ctx context.Context, hub *sse.Server) error {
	if r.cancel != nil {
		return fmt.Errorf("room %s is already running", r.Name)
	}

	cancelCtx, cancel := context.WithCancel(ctx)
	r.cancel = cancel

	hub.CreateStream(r.Name)
	r.c = make(chan roomCallback, 20)

	go func() {
		for {
			select {
			case <-cancelCtx.Done():
				hub.RemoveStream(r.Name)
				return
			case f := <-r.c:
				evt := f(r)
				r.UpdatedAt = time.Now().Unix()
				if evt == RoomNoOp {
					continue
				}
				hub.Publish(r.Name, &sse.Event{
					Data: mustMarshal(SSEMessage{
						Event: evt,
						Data:  r.toResponse(),
					}),
				})
			}
		}
	}()
	return nil
}

func (r *Room) Stop() {
	r.c <- func(room *Room) RoomEvent {
		if r.cancel != nil {
			r.cancel()
			r.cancel = nil
		}
		return RoomNoOp
	}
}

func mustMarshal(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

func (r *Room) AddPlayer(playerID, playerName string, callback func(room Room, playerID string, player Player)) {
	done := make(chan struct{})
	r.c <- func(room *Room) RoomEvent {
		defer close(done)
		if playerID == "" {
			playerID = uuid.Must(uuid.NewV7()).String()
		}
		player, exists := room.Players[playerID]
		if !exists {
			player = &Player{}
			room.Players[playerID] = player
		}
		player.Name = playerName
		if player.Name == "" {
			player.Name = toTitleCase(petname.Generate(2, " "))
		}
		slog.Info("player joined the room", slog.String("room", room.Name), slog.Bool("rejoined", exists), slog.String("player_id", playerID), slog.String("username", player.Name))
		callback(*room, playerID, *player)
		return RoomUpdated
	}
	<-done
}

func (r *Room) Cleanup() {
	r.c <- func(room *Room) RoomEvent {
		now := time.Now().Unix()
		inactive := []string{}
		for id, p := range room.Players {
			if now-p.UpdatedAt > 15*60 { // 15 minutes
				inactive = append(inactive, id)
			}
		}

		for _, id := range inactive {
			p := room.Players[id]
			delete(room.Players, id)
			slog.Info("player removed due to inactivity", slog.String("room", room.Name), slog.String("player_id", id), slog.String("username", p.Name))
		}
		return RoomUpdated
	}
}

func (r *Room) GetRoom(callback func(Room)) {
	done := make(chan struct{})
	r.c <- func(room *Room) RoomEvent {
		defer close(done)
		callback(room.toResponse())
		return RoomNoOp
	}
	<-done
}

func (r *Room) Reveal(playerID string) {
	done := make(chan struct{})
	r.c <- func(room *Room) RoomEvent {
		defer close(done)
		room.Revealed = true
		room.updatePlayerTimestamp(playerID)
		return RoomUpdated
	}
	<-done
}

func (r *Room) Reset(playerID string) {
	done := make(chan struct{})
	r.c <- func(room *Room) RoomEvent {
		defer close(done)
		for _, p := range room.Players {
			p.Card = ""
		}
		room.Revealed = false
		room.updatePlayerTimestamp(playerID)
		return RoomCleared
	}
	<-done
}

func (r *Room) Vote(playerID, card string, callback func(VoteResult)) {
	done := make(chan struct{})
	r.c <- func(room *Room) RoomEvent {
		defer close(done)
		player, ok := room.Players[playerID]
		if !ok {
			callback(VoteNotFound)
			return RoomNoOp
		}

		validCard := false
		for _, c := range room.AllowedCards {
			if c == card {
				validCard = true
				break
			}
		}
		if !validCard {
			callback(VoteInvalid)
			return RoomNoOp
		}
		player.Card = card
		player.updateTimestamp()
		callback(VoteSuccess)
		return RoomUpdated
	}
	<-done
}

// called only internally, no need to go through the channel
func (r *Room) updatePlayerTimestamp(playerID string) {
	if playerID == "" {
		return
	}
	player, exist := r.Players[playerID]
	if !exist {
		return
	}
	player.updateTimestamp()
}

func (r *Room) Save(enc *json.Encoder, callback func(error)) {
	done := make(chan struct{})
	r.c <- func(room *Room) RoomEvent {
		defer close(done)
		err := enc.Encode(room)
		callback(err)
		return RoomNoOp
	}
	<-done
}

// Converts a Room to a response struct, hiding the players' cards if the room is not revealed.
func (r *Room) toResponse() Room {
	resp := Room{
		Name:         r.Name,
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
		AllowedCards: r.AllowedCards,
		Revealed:     r.Revealed,
		Players:      map[string]*Player{},
	}
	for id, p := range r.Players {
		c := ""
		if r.Revealed {
			c = p.Card
		}
		resp.Players[id] = &Player{
			Name:  p.Name,
			Card:  c,
			Voted: p.Card != "",
		}
	}
	return resp
}

func NewRoom(roomName string) *Room {
	if roomName == "" {
		roomName = petname.Generate(3, "-")
	}
	return &Room{
		Name:         roomName,
		CreatedAt:    time.Now().Unix(),
		UpdatedAt:    time.Now().Unix(),
		Players:      map[string]*Player{},
		AllowedCards: []string{"0", "1", "2", "3", "5", "8", "13", "20", "40", "100", "❓", "☕"},
		Revealed:     false,
	}
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
