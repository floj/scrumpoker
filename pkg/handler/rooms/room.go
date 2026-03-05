package rooms

import (
	"sync"
	"time"

	petname "github.com/dustinkirkland/golang-petname"
)

type Room struct {
	mu           *sync.Mutex        `json:"-"`
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

func NewRoom() *Room {
	name := petname.Generate(3, "-")
	return &Room{
		mu:           &sync.Mutex{},
		Name:         name,
		CreatedAt:    time.Now().Unix(),
		UpdatedAt:    time.Now().Unix(),
		Players:      map[string]*Player{},
		AllowedCards: []string{"0", "1", "2", "3", "5", "8", "13", "20", "40", "100", "❓", "☕"},
		Revealed:     false,
	}
}
