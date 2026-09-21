package cli

import (
	"fmt"

	"ethenv/internal/tf"
)

const defaultDir = "terraform"

func rpcURLFromEnv(r tf.Runner, env string) (string, error) {
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
