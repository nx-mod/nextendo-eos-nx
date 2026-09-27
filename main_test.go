package main

import (
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func reset() { store = newEOSStore() }

func TestClientAndUserTokens(t *testing.T) {
	reset()
	// client credentials
	form := url.Values{"grant_type": {"client_credentials"}}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/auth/v1/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	handleAuthToken(rec, req)
	var ct map[string]any
	json.Unmarshal(rec.Body.Bytes(), &ct)
	if ct["access_token"] == "" || ct["token_type"] != "bearer" {
		t.Fatalf("client token %+v", ct)
	}

	// external user auth is stable per external id
	mkUser := func(ext string) map[string]any {
		f := url.Values{"grant_type": {"external_auth"}, "external_auth_token": {ext}}
		rec := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/auth/v1/oauth/token", strings.NewReader(f.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		handleAuthToken(rec, r)
		var m map[string]any
		json.Unmarshal(rec.Body.Bytes(), &m)
		return m
	}
	a := mkUser("nsa-123")
	b := mkUser("nsa-123")
	if a["account_id"] != b["account_id"] || a["account_id"] == "" {
		t.Fatalf("account id not stable: %v vs %v", a["account_id"], b["account_id"])
	}
	if a["access_token"] == b["access_token"] {
		t.Fatal("tokens should differ per issue")
	}
}

func TestConnectPUIDStable(t *testing.T) {
	reset()
	call := func(ext string) map[string]any {
		f := url.Values{"external_auth_token": {ext}}
		rec := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/connect/v1/oauth/token", strings.NewReader(f.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		handleConnect(rec, r)
		var m map[string]any
		json.Unmarshal(rec.Body.Bytes(), &m)
		return m
	}
	if call("x")["product_user_id"] != call("x")["product_user_id"] {
		t.Fatal("PUID not stable per external id")
	}
}

func TestLobbyCreateJoinFind(t *testing.T) {
	reset()
	host := store.connect("host")
	guest := store.connect("guest")

	// create
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/lobby/v1/lobbies", strings.NewReader(`{"bucketId":"fallguys:show","maxMembers":40}`))
	req.Header.Set("Authorization", "Bearer "+host.Value)
	handleLobbies(rec, req)
	var l lobby
	if err := json.Unmarshal(rec.Body.Bytes(), &l); err != nil {
		t.Fatal(err)
	}
	if l.ID == "" || l.OwnerPUID != host.PUID || l.MaxMembers != 40 {
		t.Fatalf("created %+v", l)
	}

	// find by bucket
	rec = httptest.NewRecorder()
	handleLobbies(rec, httptest.NewRequest("GET", "/lobby/v1/lobbies?bucket=fallguys:show", nil))
	var found struct {
		Lobbies []lobby `json:"lobbies"`
	}
	json.Unmarshal(rec.Body.Bytes(), &found)
	if len(found.Lobbies) != 1 {
		t.Fatalf("find got %d", len(found.Lobbies))
	}

	// join
	rec = httptest.NewRecorder()
	jr := httptest.NewRequest("POST", "/lobby/v1/lobbies/"+l.ID+"/join", nil)
	jr.Header.Set("Authorization", "Bearer "+guest.Value)
	handleLobbyJoin(rec, jr)
	if rec.Code != 200 {
		t.Fatalf("join code %d", rec.Code)
	}
	var joined lobby
	json.Unmarshal(rec.Body.Bytes(), &joined)
	if len(joined.Members) != 2 {
		t.Fatalf("members %v", joined.Members)
	}
}

func TestUnhandledRecorded(t *testing.T) {
	reset()
	unhandled = map[string]int64{}
	rec := httptest.NewRecorder()
	handleUnhandled(rec, httptest.NewRequest("POST", "/stats/v1/whatever", nil))
	if rec.Code != 200 || unhandled["POST /stats/v1/whatever"] != 1 {
		t.Fatalf("unhandled not recorded: %v", unhandled)
	}
}
