output "environment" {
  description = "Workspace this stack was applied in"
  value       = local.environment
}

output "chain_id" {
  description = "EIP-155 chain id of the deployed network"
  value       = var.chain_id
}

output "node_public_ips" {
  description = "Public IPv4 address of each node"
  value       = aws_instance.node[*].public_ip
}

output "node_rpc_urls" {
  description = "JSON-RPC endpoint of each node"
  value       = [for ip in aws_instance.node[*].public_ip : "http://${ip}:8545"]
}

output "ssh_targets" {
  description = "SSH targets for debugging"
  value       = [for ip in aws_instance.node[*].public_ip : "ubuntu@${ip}"]
}
