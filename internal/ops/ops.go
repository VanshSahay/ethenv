package ops

import (
	"context"
	"fmt"
	"time"

	"ethenv/internal/rpc"
	"ethenv/internal/tf"
)

func NodeURL(r tf.Runner, env, override string) (string, error) {
	if override != "" {
		return override, nil
	}
	outs, err := r.OutputsFor(env)
	if err != nil {
		return "", err
	}
	urls, err := tf.StringList(outs, "node_rpc_urls")
	if err != nil {
		return "", err
	}
	if len(urls) == 0 {
		return "", fmt.Errorf("no nodes in environment %q", env)
	}
	return urls[0], nil
}

type TxResult struct {
	From    string
	To      string
	Hash    string
	Block   string
	GasUsed string
	Status  string
	Tip     uint64
}

func SendDemo(ctx context.Context, url, amount string, onPrepared func(from, to string)) (TxResult, error) {
	c := rpc.New(url)
	accounts, err := c.Accounts(ctx)
	if err != nil {
		return TxResult{}, err
	}
	if len(accounts) < 2 {
		return TxResult{}, fmt.Errorf("node exposes no unlocked accounts (dev/anvil provides 10; prod signers are not exposed over RPC)")
	}
	from, to := accounts[0], accounts[1]
	if onPrepared != nil {
		onPrepared(from, to)
	}

	tip, err := c.BlockNumber(ctx)
	if err != nil {
		return TxResult{}, err
	}
	wei, err := rpc.EthToWei(amount)
	if err != nil {
		return TxResult{}, err
	}
	hash, err := c.SendValue(ctx, from, to, wei)
	if err != nil {
		return TxResult{}, err
	}
	res := TxResult{From: from, To: to, Hash: hash, Tip: tip}

	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(time.Second)
		receipt, err := c.Receipt(ctx, hash)
		if err != nil {
			return res, err
		}
		if receipt != nil {
			res.Block, _ = receipt["blockNumber"].(string)
			res.GasUsed, _ = receipt["gasUsed"].(string)
			res.Status, _ = receipt["status"].(string)
			return res, nil
		}
	}
	return res, fmt.Errorf("transaction %s not mined within 30s", hash)
}

func Faucet(ctx context.Context, url, to, amount string) error {
	wei, err := rpc.EthToWei(amount)
	if err != nil {
		return err
	}
	return rpc.New(url).SetBalance(ctx, to, wei)
}
