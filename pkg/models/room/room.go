package room

import (
	"encoding/json"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/floj/scrumpoker/pkg/handler/ws"
	"github.com/olahol/melody"
)

func DefaultCards() []string {
	return []string{"0", "1", "2", "3", "5", "8", "13", "20", "40", "100", "❓", "☕"}
}

type Room struct {
	mu  *sync.Mutex    `json:"-"`
	hub *melody.Melody `json:"-"`

	Name         string             `json:"name"`
	CreatedAt    int64              `json:"createdAt,omitempty"`
	UpdatedAt    int64              `json:"updatedAt,omitempty"`
	Players      map[string]*Player `json:"players"`
	AllowedCards []string           `json:"allowedCards"`
	Revealed     bool               `json:"revealed"`
}

type Player struct {
	ID        string `json:"-"`
	Name      string `json:"name"`
	Token     string `json:"token,omitempty"`
	Card      string `json:"card"`
	Voted     bool   `json:"voted"`
	UpdatedAt int64  `json:"updatedAt,omitempty"`
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

func NewRoom(name string, m *melody.Melody) *Room {
	return &Room{
		mu:  &sync.Mutex{},
		hub: m,

		Name:         name,
		CreatedAt:    time.Now().Unix(),
		UpdatedAt:    time.Now().Unix(),
		Players:      map[string]*Player{},
		AllowedCards: DefaultCards(),
		Revealed:     false,
	}
}

func (r *Room) Restore(m *melody.Melody) {
	if r.mu == nil {
		r.mu = &sync.Mutex{}
	}

	if r.Players == nil {
		r.Players = map[string]*Player{}
	}

	for id, p := range r.Players {
		p.ID = id
	}

	if r.AllowedCards == nil {
		r.AllowedCards = DefaultCards()
	}

	r.hub = m
}

func (r *Room) playerByAuth(authToken string) *Player {
	if authToken == "" {
		return nil
	}
	for _, p := range r.Players {
		if p.Token == authToken {
			return p
		}
	}
	return nil
}

func (r *Room) Do(authToken string, f func(player *Player, room *Room) (PublishEvent, error)) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	player := r.playerByAuth(authToken)

	pe, cberr := f(player, r)

	if pe != EventRoomNoOp && cberr == nil {
		now := time.Now().Unix()
		r.UpdatedAt = now
		// re-lookup to catch newly created players
		if player := r.playerByAuth(authToken); player != nil {
			player.UpdatedAt = now
		}

		sseMsg, err := json.Marshal(SSEMessage{
			Event: string(pe),
			Data:  r.ToResponse(),
		})

		if err != nil {
			slog.Error("Failed to marshal SSE message", slog.Any("error", err))
		} else {
			r.hub.BroadcastFilter(sseMsg, func(s *melody.Session) bool {
				sess, ok := ws.FromSession(s)
				if !ok {
					return false
				}
				return sess.RoomName == r.Name
			})
		}
	}

	if cberr != nil {
		return cberr
	}

	return nil
}

// Converts a Room to a response struct, hiding the players' cards if the room is not revealed.
func (r *Room) ToResponse() Room {
	resp := r.Copy()
	resp.UpdatedAt = 0
	resp.CreatedAt = 0
	for _, p := range resp.Players {
		p.Token = ""
		p.Voted = p.Card != ""
		if !r.Revealed {
			p.Card = ""
		}
		p.UpdatedAt = 0
	}
	return resp
}

func (r *Room) Copy() Room {
	cpy := Room{
		Name:         r.Name,
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
		AllowedCards: slices.Clone(r.AllowedCards),
		Revealed:     r.Revealed,
		Players:      map[string]*Player{},
	}
	for id, p := range r.Players {
		cpy.Players[id] = &Player{
			ID:        id,
			Token:     p.Token,
			Name:      p.Name,
			Card:      p.Card,
			Voted:     p.Voted,
			UpdatedAt: p.UpdatedAt,
		}
	}
	return cpy
}
