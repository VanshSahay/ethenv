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

No AWS credentials at hand? The RPC path still demos locally:

```bash
anvil --port 8545 --chain-id 1337 --block-time 2 &
./ethenv status --rpc http://127.0.0.1:8545
./ethenv send-demo --rpc http://127.0.0.1:8545
```



## Web UI

`./ethenv serve` (binds 127.0.0.1:8080) opens a black-and-white single-page
dashboard served by the same binary — zero JS dependencies, one embedded
HTML file. It shows a block-height odometer that ticks with the chain, a
per-block "tape", the node health table for the selected environment
(reads each workspace's state file directly), and dev-chain actions
(demo transfer + faucet). Deploy/destroy intentionally stay CLI-only.

## Evaluation criteria mapping


| #   | Parameter        | Where / how                                                                                                                                                |
| --- | ---------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | **Workspaces**   | `terraform workspace select dev                                                                                                                            |
| 2   | **Variables**    | `terraform/variables.tf` declares **9 variables**; `terraform.tfvars.dev` vs `.prod` differ in `node_count`, `chain_id`, `ebs_volume_size`                 |
| 3   | **Data blocks**  | `main.tf` has **4**: `aws_ami` (Canonical lookup), `aws_vpc` (default VPC), `aws_subnets`, `aws_availability_zones` — zero hardcoded IDs                   |
| 4   | **Code quality** | `terraform validate` ✓, `terraform fmt` clean, tagged resources (`Project`/`Environment`/`Component`/`ManagedBy`), `go vet` clean, single static Go binary |
| 5   | **Env config**   | dev: **1** node / 8 GiB / chain 1337 · prod: **3** validators / 20 GiB / chain 31337                                                                       |


The examiner's four verification commands all work as-is:

```bash
terraform workspace list
terraform validate
grep -c "^data " main.tf              # → 4
terraform plan -var-file="terraform.tfvars.dev"
```

`ethenv verify` automates all of the above plus the red-flag scan (hardcoded
`vpc-…`/`subnet-…`/`ami-…` ids) and exits non-zero on any failure.

## Candidate checklist

- [x] 2 workspaces (dev & prod) — created by `ethenv deploy`
- [x] `variables.tf` with 5+ variables — 9 declared
- [x] `terraform.tfvars.dev` & `.prod` with different values — 3 keys differ
- [x] 3+ data blocks — 4 (AMI, VPC, subnets, AZs)
- [x] No hardcoded resource IDs in `main.tf` — everything is a data-source lookup
- [x] Dev smaller than prod — both t3.micro **on purpose (free tier)**; prod outweighs dev via 3 nodes + 20 GiB volumes + separate chain id. One-line upgrade to t3.small is available in `terraform.tfvars.prod` when cost allows
- [x] Dev 1 instance, prod 3 instances — exactly
- [x] All resources tagged with environment name — `Environment = terraform.workspace`
- [x] `terraform validate` passes — verified
- [x] Can switch workspaces and apply both — that is literally `ethenv deploy --env dev && ethenv deploy --env prod`



## Red flags — and why none apply


| Red flag                   | This project                                                                  |
| -------------------------- | ----------------------------------------------------------------------------- |
| Hardcoded IDs              | everything from data blocks; `ethenv verify` greps for `vpc-…/subnet-…/ami-…` |
| No workspaces / only 1     | `dev` and `prod` workspaces with isolated state                               |
| Same config everywhere     | different node count, chain id, volume size                                   |
| `terraform validate` fails | passes; checked by `ethenv verify`                                            |
| No data blocks             | 4                                                                             |




## Notes

- **Free tier**: 3 × t3.micro + 2 × 8/20 GiB gp3 for a few hours is free;
still, `ethenv destroy --env prod` + `--env dev` after the demo.
- **Keys** in `terraform/signers.tf` are demo-only (`cast wallet new`),
throwaway, and pre-funded only in the private 31337 genesis. Never reuse.
Each node has two identities: a **signer key** (Clique block signing,
address baked into the genesis `extradata`) and a **nodekey** (DevP2P
transport identity — its pubkey is what appears in the enode URL and
static-nodes.json). geth writes a random nodekey on first start, so
ours is pre-provisioned at `/eth/geth/nodekey` to keep the peer list
deterministic.
- **geth is pinned to v1.13.15**: the classic Clique signer flags
(`--mine --unlock`) were removed from newer releases; v1.13.15 is a
battle-tested LTS for private PoA chains.
- **Boot flow**: user_data only installs docker + awscli and registers a
systemd `ethnode` service that runs `/eth/boot.sh` with
`Restart=on-failure`. Boot logs: `journalctl -u ethnode`. On a good run
nodes are RPC-reachable ~60-90s after launch.
- **Prod peering**: each node discovers its siblings by EC2 tags (IAM role
grants `ec2:DescribeInstances`), builds `/eth/static-nodes.json` from the
baked nodekey pubkeys + discovered private IPs, and points `--bootnodes`
at node-0. Nothing is hardcoded; SSH in with
`ssh -i ~/.ssh/ethenv_demo ubuntu@<ip>` to debug.
- The first `terraform plan` after `init` takes a couple of minutes (provider
download). Subsequent runs are fast. Nodes are RPC-reachable ~60-90s after
launch.



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

