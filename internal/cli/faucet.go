package cli

import (
	"context"
	"flag"
	"fmt"
	"time"

	"ethenv/internal/rpc"
	"ethenv/internal/tf"
	"ethenv/internal/ui"
)

func runFaucet(args []string) error {
	fs := flag.NewFlagSet("faucet", flag.ExitOnError)
	env := fs.String("env", "dev", "environment (the faucet only exists on dev/anvil)")
	to := fs.String("to", "", "recipient address (0x...)")
	amount := fs.String("eth", "10", "amount in ETH")
	rpcURL := fs.String("rpc", "", "JSON-RPC URL override")
	dir := fs.String("dir", defaultDir, "terraform module directory")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *to == "" {
		return fmt.Errorf("--to <address> is required")
	}
	if *rpcURL == "" && *env != "dev" {
		return fmt.Errorf("anvil_setBalance only exists on the dev node; use --env dev or --rpc <url>")
	}

	url := *rpcURL
	if url == "" {
		var err error
		if url, err = rpcURLFromEnv(tf.Runner{Dir: *dir}, *env); err != nil {
			return err
		}
	}

	wei, err := rpc.EthToWei(*amount)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := rpc.New(url).SetBalance(ctx, *to, wei); err != nil {
		return fmt.Errorf("faucet failed: %w", err)
	}
	fmt.Println(ui.OK(fmt.Sprintf("funded %s with %s ETH (chain of env %q)", *to, *amount, *env)))
	return nil
}
