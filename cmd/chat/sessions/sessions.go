package sessions

import (
	"log"
    "fmt"
	"time"
	"github.com/google/uuid"
)

func (s *Session) String() string {
    return fmt.Sprintf(
        "Token=%s, Game Instance=%s, Player Name=%s, Is In Game=%t, Expiry=%d",
        s.token,
        s.gameInstance,
        s.playerName,
        s.isInGame,
        s.expiry.Unix(),
    )
}

func (s *Session) GetToken() string { 
    return s.token
}

func (s *Session) GetIsInGame() bool { 
    return s.isInGame
}

func SessionToJson(s *Session) SessionAsJson {
    if s == nil {
        log.Print("Unexpected null session pointer, returning garbage")
        return SessionAsJson{}
    }
    return SessionAsJson{
        s.token,
        &s.gameInstance,
        &s.isInGame,
        &s.playerName,
    }
}

func MakeSessions() *Sessions {
    tokens := make(map[string]*Session)
    s := &Sessions{
        tokens,
        make(chan PatchFromJsonData),
        make(chan FindData),
        make(chan RequestData),
    }
    go s.aggregator()
    return s
}

//////////////////////////////////////////////////

var tokenLifetimeMinutes time.Duration = 1

func (s *Sessions) aggregator() {
    ticker := time.NewTicker(time.Minute)
    for {
        select {
        case patch := <-s.PatchFromJsonChan:
            patch.Cb<-s.patchFromJson(patch.Token, patch.Info)
        case find := <-s.FindChan:
            find.Cb<-s.find(find.Token)
        case request := <-s.RequestChan:
            request.Cb<-s.request()
        case _ = <-ticker.C:
            s.cleanUpExpired()
        }
    }
}

//////////////////////////////////////////////////
// Synchronous methods

func (s *Sessions) request() string {
    session := Session{
        uuid.New().String(),
        false,
        "",
        "",
        time.Now().Add(time.Minute * tokenLifetimeMinutes),
    }
    s.tokens[session.token] = &session
    return session.token
}

func (s *Sessions) find(token string) *Session {
    if session, found := s.tokens[token]; found {
        return session
    } else {
        return nil
    }
}

func (s *Sessions) patchFromJson(token string, req *SessionAsJson) bool {
    if req == nil {
        return false
    }

    // Ignore JSON token, redundant field

    if found := s.find(token); found != nil {
        if req.IsInGame != nil {
            found.isInGame = *req.IsInGame
        }
        if req.GameInstance != nil {
            found.gameInstance = *req.GameInstance
        }
        if req.PlayerName != nil {
            found.playerName = *req.PlayerName
        }
        found.expiry = time.Now().Add(time.Minute * tokenLifetimeMinutes)
        return true
    } else {
        return false
    }
}

func (s *Sessions) cleanUpExpired() {
    if len(s.tokens) <= 0 {
        return
    }
    log.Print("Tick clean up, count=", len(s.tokens))
    now := time.Now()
    for k, v := range s.tokens {
        if now.Compare(v.expiry) >= 0 {
            delete(s.tokens, k)
            log.Print("Deleting: ", v)
        } else {
            log.Print("Keeping: ", v)
        }
    }
}
