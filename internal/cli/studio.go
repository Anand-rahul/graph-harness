package cli

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// newStudioCmdReal implements `graph-harness studio`. P0.T42: HTTP server
// bound to loopback only with a per-session token + strict Origin check.
// SPEC §9.6 — these protections are non-negotiable.
func newStudioCmdReal() *cobra.Command {
	return &cobra.Command{
		Use:   "studio",
		Short: "Launch the local-first web UI (loopback only)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ws, err := activeWorkspace()
			if err != nil {
				return err
			}
			port, _ := cmd.Flags().GetInt("port")
			oneshot, _ := cmd.Flags().GetBool("oneshot")

			tokenBytes := make([]byte, 16)
			if _, err := rand.Read(tokenBytes); err != nil {
				return err
			}
			token := hex.EncodeToString(tokenBytes)

			mux := http.NewServeMux()
			mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
				if !checkAuth(r, token) {
					http.Error(w, "forbidden", http.StatusForbidden)
					return
				}
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("ok"))
			})
			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if !checkAuth(r, token) {
					http.Error(w, "forbidden", http.StatusForbidden)
					return
				}
				_, _ = fmt.Fprintf(w, "graph-harness studio (P0 skeleton)\nworkspace: %s\n", ws.Root)
			})

			addr := fmt.Sprintf("127.0.0.1:%d", port)
			ln, err := net.Listen("tcp", addr)
			if err != nil {
				return err
			}
			srv := &http.Server{
				Handler:           mux,
				ReadHeaderTimeout: 5 * time.Second,
			}
			out := cmd.OutOrStdout()
			_, _ = fmt.Fprintf(out, "Studio: http://%s/?token=%s\n", ln.Addr().String(), token)
			_, _ = fmt.Fprintln(out, "Bound to loopback; Origin must match http://127.0.0.1:* or http://localhost:*")
			if oneshot {
				_ = ln.Close()
				return nil
			}
			return srv.Serve(ln)
		},
	}
}

func checkAuth(r *http.Request, token string) bool {
	// Token in either query string or X-Studio-Token header.
	q := r.URL.Query().Get("token")
	h := r.Header.Get("X-Studio-Token")
	if q != token && h != token {
		return false
	}
	// Strict origin check (SPEC §9.6) — only allow loopback origins.
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true // no Origin header = same-origin tool/curl: allowed
	}
	if !strings.HasPrefix(origin, "http://127.0.0.1") && !strings.HasPrefix(origin, "http://localhost") {
		return false
	}
	return true
}
