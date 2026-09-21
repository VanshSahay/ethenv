package cli

import (
	"context"
	"flag"
	"fmt"
	"time"

	"ethenv/internal/ops"
	"ethenv/internal/tf"
	"ethenv/internal/ui"
)

func runSendDemo(args []string) error {
	fs := flag.NewFlagSet("send-demo", flag.ExitOnError)
	env := fs.String("env", "dev", "environment")
	amount := fs.String("eth", "0.1", "amount in ETH")
	rpcURL := fs.String("rpc", "", "JSON-RPC URL override")
	dir := fs.String("dir", defaultDir, "terraform module directory")
	if err := fs.Parse(args); err != nil {
		return err
	}

	url, err := ops.NodeURL(tf.Runner{Dir: *dir}, *env, *rpcURL)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	res, err := ops.SendDemo(ctx, url, *amount, func(from, to string) {
		fmt.Printf("sending %s ETH  %s → %s\n", *amount, from, to)
	})
	if res.Hash != "" {
		fmt.Println("tx hash  " + res.Hash)
	}
	if err != nil {
		return err
	}
	fmt.Println(ui.OK(fmt.Sprintf("mined: block %s (tip was %d), gasUsed %s, status %s",
		res.Block, res.Tip, res.GasUsed, res.Status)))
	return nil
}
