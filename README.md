# ethenv : multi-environment Ethereum networks on AWS

A Terraform multi-environment deployment of Ethereum nodes, driven by a Go CLI.
Two environments, two different networks:


| env    | network                       | nodes        | why                                                                                             |
| ------ | ----------------------------- | ------------ | ----------------------------------------------------------------------------------------------- |
| `dev`  | **Anvil** (Foundry dev chain) | 1 × t3.micro | instant finality, 10 prefunded accounts, one block per 2s                                       |
| `prod` | **Geth Clique PoA**           | 3 × t3.micro | a real 3-validator proof-of-authority network with peering, bootnode + tag-based peer discovery |


Both are deployed by the **same Terraform module**  
only the workspace (and its tfvars) differ. The `ethenv` CLI wraps terraform, reads the per-env
outputs, and talks JSON-RPC to the nodes (stdlib-only, zero dependencies).

```
                    ┌──────────────────────────────┐
                    │           ethenv (Go)        │
                    │  deploy / status / faucet /  │
                    │  send-demo / verify          │
                    └──────────────┬───────────────┘
                                   │ terraform CLI
        ┌──────────────────────────┴──────────────────────────┐
        │            terraform workspaces (isolated state)    │
        │     dev: terraform.tfstate.d/dev   (1 instance)     │
        │    prod: terraform.tfstate.d/prod  (3 instances)    │
        └──────────────────────────┬──────────────────────────┘
                                   │ EC2 + user_data
                 ┌─────────────────┴─────────────────┐
                 │ dev: 1× t3.micro                  │ prod: 3× t3.micro
                 │ ┌───────────────────────────┐     │ ┌──────────┐ ┌──────────┐ ┌──────────┐
                 │ │ docker: anvil  chain 1337 │     │ │ geth     │←│ geth     │←│ geth     │
                 │ └───────────────────────────┘     │ │ chain    │ │ chain    │ │ chain    │
                 │        JSON-RPC :8545             │ │ 31337 PoA│ │ 31337 PoA│ │ 31337 PoA│
                 └───────────────────────────────────┘ └──────────┘ └──────────┘ └──────────┘
```



## Exam-day runbook

Prereqs: `terraform` ✓, `go` ✓, AWS credentials.

```bash
aws configure                      # access key + secret + ap-south-1
go build -o ethenv ./cmd/ethenv

./ethenv env list                  # shows both envs + workspace state
./ethenv deploy --env dev          # init → workspace dev → plan → apply
./ethenv status --env dev          # nodes need ~3-4 min to boot
./ethenv send-demo --env dev       # 0.1 ETH transfer, mined end to end
./ethenv faucet --env dev --to 0xabc… --eth 5

./ethenv deploy --env prod         # same module, 3-node PoA network
./ethenv status --env prod         # PEERS = 2 on every node proves consensus mesh
./ethenv verify                    # runs the full evaluation checklist

./ethenv serve                     # live dashboard → http://127.0.0.1:8080

./ethenv destroy --env prod        # IMPORTANT: stays in the free tier
./ethenv destroy --env dev
```

local:

```bash
anvil --port 8545 --chain-id 1337 --block-time 2 &
./ethenv status --rpc http://127.0.0.1:8545
./ethenv send-demo --rpc http://127.0.0.1:8545
```


## Troubleshooting (learned the hard way)


| Symptom                                               | Cause / fix                                                                                                                                                                                                                                             |
| ----------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `PEERS = 0` on prod                                   | enode pubkeys must be the nodes' **P2P nodekey** identity, not the Clique signer account. The nodekey is pre-provisioned at `/eth/geth/nodekey`; if you regenerate keys in `signers.tf`, update `nodekey_enode` (`cast wallet pubkey --private-key …`). |
| `unauthorized signer` in geth logs, blocks stuck      | wrong Clique `extradata` layout: it is `32B vanity + 20-byte signer addresses back-to-back + 65B signature` (314 hex chars for 3 signers), nothing per-signer. Check with `clique_getSigners`.                                                          |
| Instance runs stale `boot.sh` after a template change | AWS updates `user_data` **in place** and it only runs on first boot; `user_data_replace_on_change = true` (already set) forces instance replacement. To force manually: `terraform apply -replace=aws_instance.node[0] …`                               |
| cloud-init "skips" user_data (boot log too short)     | clock-sync race; the systemd `ethnode` unit + `journalctl -u ethnode` exist for exactly this. Fallback: `sudo bash /var/lib/cloud/instance/scripts/part-001`.                                                                                           |
| Node seems dead right after `deploy`                  | it isn't — boot takes ~60-90s; `ethenv status` retries and prints the hint.                                                                                                                                                                             |




## Repo layout

```
terraform/
  main.tf                  data blocks, SG, IAM, EC2 (count-based, tagged)
  variables.tf             9 variables
  outputs.tf               RPC urls / public IPs consumed by the CLI
  signers.tf               demo PoA signer keys + extradata construction
  terraform.tfvars.dev     1 × t3.micro, chain 1337
  terraform.tfvars.prod    3 × t3.micro, chain 31337
  templates/anvil.sh.tftpl dev user_data (docker + anvil)
  templates/geth.sh.tftpl  prod user_data (genesis, key import, peer discovery, geth)
internal/
  tf/        terraform wrapper (workspaces, plan/apply, outputs)
  rpc/       JSON-RPC client (stdlib net/http only)
  config/    tfvars parser
  cli/       command implementations
cmd/ethenv/  entrypoint
```

