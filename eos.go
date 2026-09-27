package main

// EOS auth + connect + lobby model.
//
// Epic Online Services is a REST backend used by cross-play titles (Fall Guys,
// Rocket League, many Unreal games). A game authenticates, gets a Product User
// Id (PUID), then creates/joins lobbies and sessions. This models that in
// memory: client + user tokens, PUIDs, and a lobby store. The exact EOS request
// bodies vary by SDK version and deployment; the tractable, testable core is
// here and the rest is a logged capture surface (see NOTES.md).

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

const tokenTTL = time.Hour

// token is an issued bearer token (client or user).
type token struct {
	Value     string
	Kind      string // "client" or "user"
	AccountID string // Epic account id (user tokens)
	PUID      string // product user id (connect)
	Expires   time.Time
}

// lobby is one EOS lobby.
type lobby struct {
	ID         string         `json:"lobbyId"`
	OwnerPUID  string         `json:"ownerPuid"`
	BucketID   string         `json:"bucketId"`
	MaxMembers int            `json:"maxMembers"`
	Members    []string       `json:"members"`
	Attributes map[string]any `json:"attributes"`
	Created    int64          `json:"created"`
}

type eosStore struct {
	mu      sync.Mutex
	tokens  map[string]*token
	lobbies map[string]*lobby
	puidFor map[string]string // externalId -> PUID (stable per external identity)

	clientTokens int64
	userTokens   int64
	puids        int64
	lobbiesMade  int64
}

func newEOSStore() *eosStore {
	return &eosStore{tokens: map[string]*token{}, lobbies: map[string]*lobby{}, puidFor: map[string]string{}}
}

func id(prefix string, n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return prefix + hex.EncodeToString(b)
}

// issueClientToken issues a client-credentials access token.
func (s *eosStore) issueClientToken() *token {
	t := &token{Value: id("cl", 24), Kind: "client", Expires: time.Now().Add(tokenTTL)}
	s.mu.Lock()
	s.tokens[t.Value] = t
	s.clientTokens++
	s.mu.Unlock()
	return t
}

// issueUserToken issues a user access token for an external identity, minting a
// stable Epic account id.
func (s *eosStore) issueUserToken(externalID string) *token {
	s.mu.Lock()
	defer s.mu.Unlock()
	acct := s.puidFor["acct:"+externalID]
	if acct == "" {
		acct = id("", 16)
		s.puidFor["acct:"+externalID] = acct
	}
	t := &token{Value: id("us", 24), Kind: "user", AccountID: acct, Expires: time.Now().Add(tokenTTL)}
	s.tokens[t.Value] = t
	s.userTokens++
	return t
}

// connect exchanges an external identity for a Product User Id + token.
func (s *eosStore) connect(externalID string) *token {
	s.mu.Lock()
	defer s.mu.Unlock()
	puid := s.puidFor["puid:"+externalID]
	if puid == "" {
		puid = id("", 16)
		s.puidFor["puid:"+externalID] = puid
		s.puids++
	}
	t := &token{Value: id("pu", 24), Kind: "user", PUID: puid, Expires: time.Now().Add(tokenTTL)}
	s.tokens[t.Value] = t
	return t
}

// lookup returns a live token, or nil.
func (s *eosStore) lookup(value string) *token {
	s.mu.Lock()
	defer s.mu.Unlock()
	t := s.tokens[value]
	if t == nil || time.Now().After(t.Expires) {
		return nil
	}
	return t
}

// createLobby makes a lobby owned by puid.
func (s *eosStore) createLobby(puid, bucket string, maxMembers int, attrs map[string]any) *lobby {
	l := &lobby{
		ID: id("lb", 12), OwnerPUID: puid, BucketID: bucket, MaxMembers: maxMembers,
		Members: []string{puid}, Attributes: attrs, Created: time.Now().Unix(),
	}
	s.mu.Lock()
	s.lobbies[l.ID] = l
	s.lobbiesMade++
	s.mu.Unlock()
	return l
}

// joinLobby adds a member; returns the lobby or nil if full/missing.
func (s *eosStore) joinLobby(lobbyID, puid string) *lobby {
	s.mu.Lock()
	defer s.mu.Unlock()
	l := s.lobbies[lobbyID]
	if l == nil {
		return nil
	}
	for _, m := range l.Members {
		if m == puid {
			return l
		}
	}
	if l.MaxMembers > 0 && len(l.Members) >= l.MaxMembers {
		return nil
	}
	l.Members = append(l.Members, puid)
	return l
}

// findLobbies returns lobbies in a bucket with room, up to max.
func (s *eosStore) findLobbies(bucket string, max int) []*lobby {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []*lobby
	for _, l := range s.lobbies {
		if (bucket == "" || l.BucketID == bucket) && (l.MaxMembers == 0 || len(l.Members) < l.MaxMembers) {
			out = append(out, l)
			if len(out) >= max {
				break
			}
		}
	}
	return out
}
