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
        s.GameInstance,
        s.IsInGame,
        s.PlayerName,
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

// [Timed out tokens]
// Currently use crude approach after timing out,
// where the next action which needs a token will fail,
// but won't actively kick people out of chat for example.
// Game server must slow but constantly ping the sessions server.
var tokenLifetimeMinutesFromRequest time.Duration = 1
var tokenLifetimeMinutesFromPatch time.Duration = 5

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
            token, success := s.request()
            if success {
                request.Cb <- &token
            } else {
                request.Cb <- nil
            }
            close(request.Cb)
        case _ = <-ticker.C:
            s.cleanUpExpired()
        }
    }
}

//////////////////////////////////////////////////
// Synchronous methods

func (s *Sessions) request() (string, bool) {
    u := uuid.New().String()
    if s.tokens[u] != nil {
        log.Print("UUID collision, rejecting ", u)
        return "", false
    }
    session := Session{
        u,
        false,
        "",
        "",
        time.Now().Add(time.Minute * tokenLifetimeMinutesFromRequest),
    }
    s.tokens[session.Token] = &session
    return session.Token, true
}

func (s *Sessions) findAndCopy(token string) (Session, bool) {
    if session, found := s.tokens[token]; found {
        return *session, true
    } else {
        return Session{}, false
    }
}

func (s *Sessions) patchFromJson(token string, req *PatchSessionRequest) bool {
    if req == nil {
        return false
    }

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
        // Can send nothing to continue refreshing the expiry
        found.Expiry = time.Now().Add(time.Minute * tokenLifetimeMinutesFromPatch)
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
        if now.Compare(v.Expiry) >= 0 {
            log.Print("Deleting: ", v)
            delete(s.tokens, k)
        } else {
            log.Print("Keeping: ", v)
        }
    }
}
