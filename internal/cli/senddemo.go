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

func runSendDemo(args []string) error {
	fs := flag.NewFlagSet("send-demo", flag.ExitOnError)
	env := fs.String("env", "dev", "environment")
	amount := fs.String("eth", "0.1", "amount in ETH")
	rpcURL := fs.String("rpc", "", "JSON-RPC URL override")
	dir := fs.String("dir", defaultDir, "terraform module directory")
	if err := fs.Parse(args); err != nil {
		return err
	}

	url := *rpcURL
	if url == "" {
		var err error
		if url, err = rpcURLFromEnv(tf.Runner{Dir: *dir}, *env); err != nil {
			return err
		}
	}
	c := rpc.New(url)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	accounts, err := c.Accounts(ctx)
	if err != nil {
		return err
	}
	if len(accounts) < 2 {
		return fmt.Errorf("node exposes no unlocked accounts (dev/anvil provides 10; prod signers are not exposed over RPC)")
	}
	from, to := accounts[0], accounts[1]

	wei, err := rpc.EthToWei(*amount)
	if err != nil {
		return err
	}
	before, err := c.BlockNumber(ctx)
	if err != nil {
		return err
	}

	fmt.Printf("sending %s ETH  %s → %s\n", *amount, from, to)
	hash, err := c.SendValue(ctx, from, to, wei)
	if err != nil {
		return err
	}
	fmt.Println("tx hash  " + hash)

	var receipt map[string]any
	for i := 0; i < 30; i++ {
		time.Sleep(time.Second)
		if receipt, err = c.Receipt(ctx, hash); err != nil {
			return err
		}
		if receipt != nil {
			break
		}
	}
	if receipt == nil {
		return fmt.Errorf("transaction %s not mined within 30s", hash)
	}

	blockNum, _ := receipt["blockNumber"].(string)
	gasUsed, _ := receipt["gasUsed"].(string)
	status, _ := receipt["status"].(string)
	fmt.Println(ui.OK(fmt.Sprintf("mined: block %s (tip was %d), gasUsed %s, status %s",
		blockNum, before, gasUsed, status)))
	return nil
}
