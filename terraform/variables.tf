variable "aws_region" {
  description = "AWS region to deploy the Ethereum network in"
  type        = string
  default     = "ap-south-1"
}

variable "project_name" {
  description = "Project name used as a tag on every resource"
  type        = string
  default     = "ethenv"
}

variable "instance_type" {
  description = "EC2 instance type per Ethereum node (t3.micro keeps us in the free tier)"
  type        = string
  default     = "t3.micro"
}

variable "node_count" {
  description = "Number of nodes: dev runs 1 Anvil node, prod runs 3 Geth PoA signers"
  type        = number
  default     = 1

  validation {
    condition     = var.node_count >= 1 && var.node_count <= 3
    error_message = "The Clique PoA network is provisioned for 1-3 signer nodes."
  }
}

variable "chain_id" {
  description = "EIP-155 chain id of the private network (differs per environment to prevent replay)"
  type        = number
  default     = 1337
}

variable "clique_period" {
  description = "Clique PoA block time in seconds (prod only; Anvil uses its own block-time)"
  type        = number
  default     = 2
}

variable "ebs_volume_size" {
  description = "Root EBS volume (gp3) size in GiB per node"
  type        = number
  default     = 8
}

variable "ssh_allowed_cidr" {
  description = "CIDR allowed to SSH (port 22). Restrict to your IP outside of demos."
  type        = string
  default     = "0.0.0.0/0"
}

variable "rpc_allowed_cidr" {
  description = "CIDR allowed to reach the JSON-RPC port 8545"
  type        = string
  default     = "0.0.0.0/0"
}

variable "ssh_public_key_path" {
  description = "Public key registered for SSH access to the nodes"
  type        = string
  default     = "~/.ssh/ethenv_demo.pub"
}
