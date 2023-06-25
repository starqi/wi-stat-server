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
        s.Token,
        s.GameInstance,
        s.PlayerName,
        s.IsInGame,
        s.Expiry.Unix(),
    )
}

func SessionToJson(s *Session) SessionAsJson {
    if s == nil {
        log.Print("Unexpected null session pointer, returning garbage")
        return SessionAsJson{}
    }
    return SessionAsJson{
        s.Token,
        &s.GameInstance,
        &s.IsInGame,
        &s.PlayerName,
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
            patch.Cb <- s.patchFromJson(patch.Token, patch.Info)
            close(patch.Cb)
        case find := <-s.FindChan:
            sessionCopy, found := s.findAndCopy(find.Token)
            if !found {
                find.Cb <- nil
            } else {
                find.Cb <- &sessionCopy
            }
            close(find.Cb)
        case request := <-s.RequestChan:
            request.Cb <- s.request()
            close(request.Cb)
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
    s.tokens[session.Token] = &session
    return session.Token
}

func (s *Sessions) findAndCopy(token string) (Session, bool) {
    if session, found := s.tokens[token]; found {
        return *session, true
    } else {
        return Session{}, false
    }
}

func (s *Sessions) patchFromJson(token string, req *SessionAsJson) bool {
    if req == nil {
        return false
    }

    // Ignore JSON token, redundant field

    if found := s.tokens[token]; found != nil {
        if req.IsInGame != nil {
            found.IsInGame = *req.IsInGame
        }
        if req.GameInstance != nil {
            found.GameInstance = *req.GameInstance
        }
        if req.PlayerName != nil {
            found.PlayerName = *req.PlayerName
        }
        found.Expiry = time.Now().Add(time.Minute * tokenLifetimeMinutes)
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
        // Right now it's simple, there is no need to update expiry, when player leaves the game, clean up is immediate
        if !v.IsInGame && now.Compare(v.Expiry) >= 0 {
            log.Print("Deleting: ", v)
            delete(s.tokens, k)
        } else {
            log.Print("Keeping: ", v)
        }
    }
}
