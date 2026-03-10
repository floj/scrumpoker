package room

import (
	"encoding/json"
	"sync"
	"time"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/r3labs/sse/v2"
)

type Room struct {
	mu  *sync.Mutex `json:"-"`
	hub *sse.Server `json:"-"`

	Name         string             `json:"name"`
	CreatedAt    int64              `json:"createdAt"`
	UpdatedAt    int64              `json:"updatedAt"`
	Players      map[string]*Player `json:"players"`
	AllowedCards []string           `json:"allowedCards"`
	Revealed     bool               `json:"revealed"`
}

type Player struct {
	Name      string `json:"name"`
	Card      string `json:"card"`
	Voted     bool   `json:"voted"`
	UpdatedAt int64  `json:"-"`
}

type SSEMessage struct {
	Event string `json:"eventName"`
	Data  any    `json:"data"`
}

type PublishEvent string

const (
	EventRoomUpdated PublishEvent = "room_updated"
	EventRoomCleared PublishEvent = "room_cleared"
	EventRoomNoOp    PublishEvent = "room_no_op"
)

func NewRoom() *Room {
	name := petname.Generate(3, "-")
	return &Room{
		Name:         name,
		CreatedAt:    time.Now().Unix(),
		UpdatedAt:    time.Now().Unix(),
		Players:      map[string]*Player{},
		AllowedCards: []string{"0", "1", "2", "3", "5", "8", "13", "20", "40", "100", "❓", "☕"},
		Revealed:     false,
	}
}

func (r *Room) Init(hub *sse.Server) {
	r.mu = &sync.Mutex{}
	r.hub = hub
	hub.CreateStream(r.Name)
}

func (r *Room) Do(playerID string, f func(player *Player, room *Room) (PublishEvent, error)) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.UpdatedAt = time.Now().Unix()

	player, exist := r.Players[playerID]
	if exist {
		player.UpdatedAt = time.Now().Unix()
	}

	pe, err := f(player, r)
	if err != nil {
		return err
	}
	if pe == EventRoomNoOp {
		return nil
	}

	r.hub.Publish(r.Name, &sse.Event{
		Data: mustMarshal(SSEMessage{
			Event: string(pe),
			Data:  r.ToResponse(),
		}),
	})

	return nil
}

func (r *Room) updatePlayerTimestamp(playerID string) {
	if p, exists := r.Players[playerID]; exists {
		p.UpdatedAt = time.Now().Unix()
	}
}

// Converts a Room to a response struct, hiding the players' cards if the room is not revealed.
func (r *Room) ToResponse() Room {
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

func mustMarshal(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
