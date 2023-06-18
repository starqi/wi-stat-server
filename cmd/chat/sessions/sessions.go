package sessions

import (
    "github.com/google/uuid"
    "time"
)

var tokenLifetimeMinutes time.Duration = 1

type Session struct {
    token string
    isInGame bool
    gameInstance string
    expiry time.Time
}

type SessionAsJson struct {
    Token string `json:"token"`
    GameInstance string `json:"gameInstance"`
    IsInGame bool `json:"isInGame"`
}

func SessionToJson(s *Session) SessionAsJson {
    return SessionAsJson{
        s.token,
        s.gameInstance,
        s.isInGame,
    }
}

func (s *Session) GetToken() string { 
    return s.token
}

func (s *Session) GetIsInGame() bool { 
    return s.isInGame
}

type Sessions struct {
    tokens map[string]Session
}

func MakeSessions() *Sessions {
    tokens := make(map[string]Session)
    return &Sessions{
        tokens,
    }
}

func (s *Sessions) Request() string {
    session := Session{
        uuid.New().String(),
        false,
        "",
        time.Now().Add(time.Minute * tokenLifetimeMinutes),
    }
    s.tokens[session.token] = session
    return session.token
}

func (s *Sessions) Find(id string) *Session {
    if session, found := s.tokens[id]; found {
        return &session
    } else {
        return nil
    }
}

func (s *Sessions) PatchFromJson(req *SessionAsJson) bool {
    if found := s.Find(req.Token); found != nil {
        found.isInGame = req.IsInGame
        found.gameInstance = req.GameInstance
        found.expiry = time.Now().Add(time.Minute * tokenLifetimeMinutes)
        return true
    } else {
        return false
    }
}
