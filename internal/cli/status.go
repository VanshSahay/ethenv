package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"ethenv/internal/rpc"
	"ethenv/internal/tf"
	"ethenv/internal/ui"
)

type urlList []string

func (u *urlList) String() string { return strings.Join(*u, ",") }
func (u *urlList) Set(v string) error {
	*u = append(*u, v)
	return nil
}

func runStatus(args []string) error {
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	env := fs.String("env", "", "environment to inspect (reads terraform outputs)")
	dir := fs.String("dir", defaultDir, "terraform module directory")
	var urls urlList
	fs.Var(&urls, "rpc", "probe this JSON-RPC URL directly (repeatable, skips terraform)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	targets := []string(urls)
	envName := *env
	if len(targets) == 0 {
		if envName == "" {
			return fmt.Errorf("either --env <name> or --rpc <url> is required")
		}
		outs, err := (tf.Runner{Dir: *dir}).Outputs()
		if err != nil {
			return err
		}
		if targets, err = tf.StringList(outs, "node_rpc_urls"); err != nil {
			return err
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	rows := make([][]string, 0, len(targets))
	reachable := 0
	for i, u := range targets {
		row := probe(ctx, u)
		if row.status == "ok" {
			reachable++
		}
		rows = append(rows, []string{
			fmt.Sprintf("node-%d", i),
			u,
			row.client,
			fmt.Sprintf("%d", row.chain),
			fmt.Sprintf("%d", row.block),
			fmt.Sprintf("%d", row.peers),
			row.status,
		})
	}

	fmt.Println(ui.Header("node health:"))
	ui.Table(os.Stdout,
		[]string{"NODE", "ENDPOINT", "CLIENT", "CHAIN", "BLOCK", "PEERS", "STATUS"},
		rows)

	if reachable == 0 {
		fmt.Println()
		fmt.Println(ui.Warn("no node reachable yet - instances need ~3-4 min to install docker and start the chain"))
		fmt.Println(ui.Dim + "retry with: ethenv status" + envFlagHint(envName) + ui.Reset)
	}
	return nil
}

type nodeRow struct {
	client       string
	chain, block uint64
	peers        uint64
	status       string
}

func probe(ctx context.Context, url string) nodeRow {
	c := rpc.New(url)
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		if attempt > 0 {
			time.Sleep(2 * time.Second)
		}
		version, err := c.ClientVersion(ctx)
		if err != nil {
			lastErr = err
			continue
		}
		chain, err := c.ChainID(ctx)
		if err != nil {
			lastErr = err
			continue
		}
		block, err := c.BlockNumber(ctx)
		if err != nil {
			lastErr = err
			continue
		}
		peers, _ := c.PeerCount(ctx)
		return nodeRow{client: version, chain: chain, block: block, peers: peers, status: "ok"}
	}
	status := "down"
	if lastErr != nil {
		status = "down: " + shortErr(lastErr)
	}
	return nodeRow{client: "-", status: status}
}

func shortErr(err error) string {
	s := err.Error()
	if i := strings.Index(s, ": "); i >= 0 {
		s = s[i+2:]
	}
	if len(s) > 40 {
		s = s[:37] + "..."
	}
	return s
}

func envFlagHint(env string) string {
	if env == "" {
		return ""
	}
	return " --env " + env
}
