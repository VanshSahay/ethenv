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
		outs, err := (tf.Runner{Dir: *dir}).OutputsFor(envName)
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
		st := rpc.Probe(ctx, u, 2)
		if st.State == "ok" {
			reachable++
		}
		rows = append(rows, []string{
			fmt.Sprintf("node-%d", i),
			u,
			st.Client,
			fmt.Sprintf("%d", st.Chain),
			fmt.Sprintf("%d", st.Block),
			fmt.Sprintf("%d", st.Peers),
			st.State,
		})
		if st.Err != "" {
			rows[len(rows)-1][6] = st.State + ": " + st.Err
		}
	}

	fmt.Println(ui.Header("node health:"))
	ui.Table(os.Stdout,
		[]string{"NODE", "ENDPOINT", "CLIENT", "CHAIN", "BLOCK", "PEERS", "STATUS"},
		rows)

	if reachable == 0 {
		fmt.Println()
		fmt.Println(ui.Warn("no node reachable yet - instances need ~90s to install docker and start the chain"))
		fmt.Println(ui.Dim + "retry with: ethenv status" + envFlagHint(envName) + ui.Reset)
	}
	return nil
}

func envFlagHint(env string) string {
	if env == "" {
		return ""
	}
	return " --env " + env
}
