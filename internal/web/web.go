package web

import (
	"context"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"ethenv/internal/config"
	"ethenv/internal/ops"
	"ethenv/internal/rpc"
	"ethenv/internal/tf"
)

//go:embed index.html
var indexHTML []byte

func Run(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	addr := fs.String("addr", "127.0.0.1:8080", "listen address")
	dir := fs.String("dir", "terraform", "terraform module directory")
	if err := fs.Parse(args); err != nil {
		return err
	}
	fmt.Printf("ethenv ui → http://%s\n", *addr)
	return http.ListenAndServe(*addr, Handler(*dir))
}

func Handler(dir string) http.Handler {
	r := tf.Runner{Dir: dir}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(indexHTML)
	})
	mux.HandleFunc("GET /api/envs", func(w http.ResponseWriter, _ *http.Request) {
		envs, err := config.List(dir)
		if err != nil {
			writeErr(w, 500, err.Error())
			return
		}
		type envInfo struct {
			Name         string `json:"name"`
			NodeCount    int    `json:"nodeCount"`
			InstanceType string `json:"instanceType"`
			ChainID      int    `json:"chainId"`
			Region       string `json:"region"`
			Deployed     bool   `json:"deployed"`
		}
		out := make([]envInfo, 0, len(envs))
		for _, e := range envs {
			out = append(out, envInfo{
				Name: e.Name, NodeCount: e.NodeCount, InstanceType: e.InstanceType,
				ChainID: e.ChainID, Region: e.Region, Deployed: r.WorkspaceStateExists(e.Name),
			})
		}
		writeJSON(w, out)
	})
	mux.HandleFunc("GET /api/status", func(w http.ResponseWriter, req *http.Request) {
		env := req.URL.Query().Get("env")
		if env != "dev" && env != "prod" {
			writeErr(w, 400, "unknown environment")
			return
		}
		outs, err := r.OutputsFor(env)
		if err != nil {
			writeErr(w, 409, err.Error())
			return
		}
		urls, err := tf.StringList(outs, "node_rpc_urls")
		if err != nil {
			writeErr(w, 409, err.Error())
			return
		}
		chainStr, _ := tf.String(outs, "chain_id")
		chainID, _ := strconv.ParseUint(chainStr, 10, 64)

		ctx, cancel := context.WithTimeout(req.Context(), 8*time.Second)
		defer cancel()

		type node struct {
			Name     string `json:"name"`
			Endpoint string `json:"endpoint"`
			Client   string `json:"client"`
			Chain    uint64 `json:"chain"`
			Block    uint64 `json:"block"`
			Peers    uint64 `json:"peers"`
			State    string `json:"state"`
			Err      string `json:"err,omitempty"`
		}
		nodes := make([]node, 0, len(urls))
		height := uint64(0)
		for i, u := range urls {
			st := rpc.Probe(ctx, u, 1)
			if st.Block > height {
				height = st.Block
			}
			nodes = append(nodes, node{
				Name: fmt.Sprintf("node-%d", i), Endpoint: u, Client: st.Client,
				Chain: st.Chain, Block: st.Block, Peers: st.Peers, State: st.State, Err: st.Err,
			})
		}
		writeJSON(w, map[string]any{"env": env, "chainId": chainID, "height": height, "nodes": nodes})
	})
	mux.HandleFunc("POST /api/send-demo", func(w http.ResponseWriter, req *http.Request) {
		var body struct {
			Amount string `json:"amount"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil || body.Amount == "" {
			body.Amount = "0.1"
		}
		url, err := ops.NodeURL(r, "dev", "")
		if err != nil {
			writeErr(w, 409, err.Error())
			return
		}
		ctx, cancel := context.WithTimeout(req.Context(), 45*time.Second)
		defer cancel()
		res, err := ops.SendDemo(ctx, url, body.Amount, nil)
		if err != nil {
			writeErr(w, 500, err.Error())
			return
		}
		writeJSON(w, res)
	})
	mux.HandleFunc("POST /api/faucet", func(w http.ResponseWriter, req *http.Request) {
		var body struct {
			To     string `json:"to"`
			Amount string `json:"amount"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil || body.To == "" {
			writeErr(w, 400, "address required")
			return
		}
		if body.Amount == "" {
			body.Amount = "10"
		}
		url, err := ops.NodeURL(r, "dev", "")
		if err != nil {
			writeErr(w, 409, err.Error())
			return
		}
		ctx, cancel := context.WithTimeout(req.Context(), 30*time.Second)
		defer cancel()
		if err := ops.Faucet(ctx, url, body.To, body.Amount); err != nil {
			writeErr(w, 500, err.Error())
			return
		}
		writeJSON(w, map[string]string{"ok": "funded " + body.To + " with " + body.Amount + " ETH"})
	})
	return mux
}

func trimJSON(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
