package cli

import (
	"fmt"
	"os"
)

const usageText = `ethenv - multi-environment Ethereum networks on AWS

Usage:
  ethenv env list                     show environments and workspace state
  ethenv deploy --env <name>          init + workspace + plan + apply
  ethenv destroy --env <name>         tear an environment down (free tier!)
  ethenv status --env <name>          live RPC health per node
  ethenv faucet --env dev --to 0x..   fund a dev account (anvil_setBalance)
  ethenv send-demo [--env dev]        send and mine a value transfer
  ethenv verify                       run the evaluation checklist end to end
  ethenv version                      print version

Global flags: --dir <path>  terraform module directory (default "terraform")`

func Run(args []string) error {
	if len(args) == 0 {
		fmt.Println(usageText)
		return nil
	}
	switch args[0] {
	case "env":
		return runEnv(args[1:])
	case "deploy":
		return runDeploy(args[1:])
	case "destroy":
		return runDestroy(args[1:])
	case "status":
		return runStatus(args[1:])
	case "faucet":
		return runFaucet(args[1:])
	case "send-demo":
		return runSendDemo(args[1:])
	case "verify":
		return runVerify(args[1:])
	case "version", "--version", "-v":
		fmt.Println("ethenv v1.0.0")
		return nil
	case "help", "--help", "-h":
		fmt.Println(usageText)
		return nil
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s\n", args[0], usageText)
		os.Exit(2)
		return nil
	}
}
