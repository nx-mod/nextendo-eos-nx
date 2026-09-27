// Command nextendo-eos-nx serves an Epic Online Services (EOS) backend for
// Nextendo Network, so EOS-based cross-play titles (Fall Guys and other Unreal
// games) work on the stack instead of Epic's servers.
//
// EOS is a REST backend: a game gets a client token, authenticates a user, is
// issued a Product User Id (PUID), then creates/joins lobbies and sessions.
// This implements that core in memory and logs everything else as a capture
// surface. The Fall Guys deployment specifics live on the `fall-guys` branch.
//
//	POST /auth/v1/oauth/token       client-credentials / external -> access token
//	POST /epic/oauth/v2/token       Epic account services user token
//	POST /connect/v1/oauth/token    external identity -> PUID + token
//	POST /lobby/v1/lobbies          create a lobby (bearer PUID)
//	GET  /lobby/v1/lobbies?bucket=  find lobbies
//	POST /lobby/v1/lobbies/{id}/join
//
// The exact EOS request/response bodies vary by SDK version; these are the
// sensible shapes. See NOTES.md.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	httpPort   = envOrInt("EOS_PORT", 8476)
	deployment = envOr("EOS_DEPLOYMENT_ID", "")
	certFile   = envOr("CERT_FILE", "")
	keyFile    = envOr("KEY_FILE", "")
	dashPort   = envOr("DASH_PORT", "8105")
	dashToken  = envOr("DASH_TOKEN", "")

	store     = newEOSStore()
	dashStart = time.Now()

	unhMu     sync.Mutex
	unhandled = map[string]int64{} // "METHOD path" -> count
	unhTotal  atomic.Int64
)

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
func envOrInt(k string, d int) int {
	if v := os.Getenv(k); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return d
}

func main() {
	log.SetOutput(os.Stdout)
	go startDashboard()

	mux := http.NewServeMux()
	mux.HandleFunc("/auth/v1/oauth/token", handleAuthToken)
	mux.HandleFunc("/epic/oauth/v2/token", handleAuthToken)
	mux.HandleFunc("/connect/v1/oauth/token", handleConnect)
	mux.HandleFunc("/lobby/v1/lobbies", handleLobbies)
	mux.HandleFunc("/lobby/v1/lobbies/", handleLobbyJoin)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { fmt.Fprintln(w, "ok") })
	mux.HandleFunc("/", handleUnhandled)

	addr := fmt.Sprintf(":%d", httpPort)
	srv := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 15 * time.Second}
	if certFile != "" && keyFile != "" {
		log.Printf("[EOS] listening HTTPS %s (deployment=%q)", addr, deployment)
		log.Fatal(srv.ListenAndServeTLS(certFile, keyFile))
	}
	log.Printf("[EOS] listening HTTP %s (TLS via sni-router; deployment=%q)", addr, deployment)
	log.Fatal(srv.ListenAndServe())
}

func handleAuthToken(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	grant := r.FormValue("grant_type")
	if grant == "client_credentials" {
		t := store.issueClientToken()
		writeToken(w, t, map[string]any{"kind": "client_credentials"})
		return
	}
	// external / password / exchange_code: mint a user token for the external id
	ext := firstNonEmpty(r.FormValue("external_auth_token"), r.FormValue("exchange_code"), r.FormValue("account_id"), r.FormValue("username"), grant)
	t := store.issueUserToken(ext)
	writeToken(w, t, map[string]any{"account_id": t.AccountID})
	log.Printf("[EOS] auth grant=%s -> account=%s", grant, t.AccountID)
}

func handleConnect(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	ext := firstNonEmpty(r.FormValue("external_auth_token"), r.FormValue("nsa_id_token"), r.FormValue("id_token"), r.FormValue("user_id"))
	t := store.connect(ext)
	writeToken(w, t, map[string]any{"product_user_id": t.PUID})
	log.Printf("[EOS] connect -> puid=%s", t.PUID)
}

// puidFromAuth resolves the bearer token to a PUID (or "").
func puidFromAuth(r *http.Request) string {
	auth := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if t := store.lookup(strings.TrimSpace(auth)); t != nil {
		if t.PUID != "" {
			return t.PUID
		}
		return t.AccountID
	}
	return ""
}

func handleLobbies(w http.ResponseWriter, r *http.Request) {
	puid := puidFromAuth(r)
	if r.Method == http.MethodGet {
		max := 50
		found := store.findLobbies(r.URL.Query().Get("bucket"), max)
		writeJSON(w, map[string]any{"lobbies": found})
		return
	}
	if puid == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var req struct {
		BucketID   string         `json:"bucketId"`
		MaxMembers int            `json:"maxMembers"`
		Attributes map[string]any `json:"attributes"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.MaxMembers == 0 {
		req.MaxMembers = 4
	}
	l := store.createLobby(puid, req.BucketID, req.MaxMembers, req.Attributes)
	log.Printf("[EOS] lobby create %s owner=%s bucket=%s", l.ID, puid, l.BucketID)
	writeJSON(w, l)
}

func handleLobbyJoin(w http.ResponseWriter, r *http.Request) {
	puid := puidFromAuth(r)
	if puid == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	// /lobby/v1/lobbies/{id}/join
	rest := strings.TrimPrefix(r.URL.Path, "/lobby/v1/lobbies/")
	lobbyID := strings.TrimSuffix(rest, "/join")
	l := store.joinLobby(lobbyID, puid)
	if l == nil {
		http.Error(w, "lobby full or missing", http.StatusConflict)
		return
	}
	log.Printf("[EOS] lobby join %s puid=%s (%d members)", lobbyID, puid, len(l.Members))
	writeJSON(w, l)
}

// handleUnhandled records and 200s any EOS endpoint not implemented, so the game
// keeps going and the missing surface is visible on the dashboard.
func handleUnhandled(w http.ResponseWriter, r *http.Request) {
	key := r.Method + " " + r.URL.Path
	unhMu.Lock()
	unhandled[key]++
	unhMu.Unlock()
	unhTotal.Add(1)
	log.Printf("[EOS] UNHANDLED %s", key)
	writeJSON(w, map[string]any{})
}

func writeToken(w http.ResponseWriter, t *token, extra map[string]any) {
	out := map[string]any{
		"access_token": t.Value,
		"token_type":   "bearer",
		"expires_in":   int(time.Until(t.Expires).Seconds()),
		"expires_at":   t.Expires.UTC().Format(time.RFC3339),
	}
	for k, v := range extra {
		out[k] = v
	}
	writeJSON(w, out)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func startDashboard() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/stats", func(w http.ResponseWriter, r *http.Request) {
		if dashToken != "" && r.URL.Query().Get("key") != dashToken {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		store.mu.Lock()
		ct, ut, pu, lm, lobbies := store.clientTokens, store.userTokens, store.puids, store.lobbiesMade, len(store.lobbies)
		store.mu.Unlock()
		unhMu.Lock()
		unh := make(map[string]int64, len(unhandled))
		for k, v := range unhandled {
			unh[k] = v
		}
		unhMu.Unlock()
		writeJSON(w, map[string]any{
			"uptimeSeconds": int(time.Since(dashStart).Seconds()),
			"clientTokens":  ct, "userTokens": ut, "puids": pu,
			"lobbiesMade": lm, "activeLobbies": lobbies,
			"unhandled": unh, "deployment": deployment, "stack": "eos",
		})
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { fmt.Fprintln(w, "ok") })
	log.Printf("[EOS Dashboard] :%s", dashPort)
	if err := http.ListenAndServe(":"+dashPort, mux); err != nil {
		log.Printf("[EOS Dashboard] %v", err)
	}
}
