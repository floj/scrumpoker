package ws

import "github.com/olahol/melody"

type WsSessionData struct {
	RoomName string
	PlayerID string
}

func FromSession(s *melody.Session) (WsSessionData, bool) {
	v, ok := s.Get("wsSessionData")
	if !ok {
		return WsSessionData{}, false
	}
	data, ok := v.(WsSessionData)
	if !ok {
		return WsSessionData{}, false
	}
	return data, true
}
