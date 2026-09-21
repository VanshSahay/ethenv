locals {
  environment = terraform.workspace

  common_tags = {
    Project     = var.project_name
    Environment = local.environment
    Component   = "ethereum-node"
    ManagedBy   = "terraform"
  }
}

data "aws_ami" "ubuntu" {
  most_recent = true
  owners      = ["099720109477"]

  filter {
    name   = "name"
    values = ["ubuntu/images/hvm-ssd/ubuntu-jammy-22.04-amd64-server-*"]
  }

  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }
}

data "aws_vpc" "default" {
  default = true
}

data "aws_subnets" "default" {
  filter {
    name   = "vpc-id"
    values = [data.aws_vpc.default.id]
  }
}

data "aws_availability_zones" "available" {
  state = "available"
}

resource "aws_security_group" "nodes" {
  name        = "${var.project_name}-${local.environment}-nodes"
  description = "Ethereum node traffic for ${var.project_name}/${local.environment}"
  vpc_id      = data.aws_vpc.default.id

  ingress {
    description = "SSH"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = [var.ssh_allowed_cidr]
  }

  ingress {
    description = "JSON-RPC"
    from_port   = 8545
    to_port     = 8545
    protocol    = "tcp"
    cidr_blocks = [var.rpc_allowed_cidr]
  }

  ingress {
    description = "DevP2P peering (TCP)"
    from_port   = 30303
    to_port     = 30303
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    description = "DevP2P discovery (UDP)"
    from_port   = 30303
    to_port     = 30303
    protocol    = "udp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    description = "All outbound"
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = merge(local.common_tags, {
    Name = "${var.project_name}-${local.environment}-nodes"
  })
}

resource "aws_iam_role" "node" {
  count = local.environment == "prod" ? 1 : 0
  name  = "${var.project_name}-${local.environment}-node-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action    = "sts:AssumeRole"
      Effect    = "Allow"
      Principal = { Service = "ec2.amazonaws.com" }
    }]
  })

  tags = local.common_tags
}

resource "aws_iam_role_policy" "discover_peers" {
  count = local.environment == "prod" ? 1 : 0
  name  = "discover-peers"
  role  = aws_iam_role.node[0].id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow"
      Action   = ["ec2:DescribeInstances"]
      Resource = "*"
    }]
  })
}

resource "aws_iam_instance_profile" "node" {
  count = local.environment == "prod" ? 1 : 0
  name  = "${var.project_name}-${local.environment}-node-profile"
  role  = aws_iam_role.node[0].name
}

resource "aws_key_pair" "deployer" {
  key_name   = "${var.project_name}-${local.environment}-key"
  public_key = file(pathexpand(var.ssh_public_key_path))
  tags       = local.common_tags
}

resource "aws_instance" "node" {
  count = var.node_count

  ami                         = data.aws_ami.ubuntu.id
  instance_type               = var.instance_type
  subnet_id                   = data.aws_subnets.default.ids[count.index % length(data.aws_subnets.default.ids)]
  associate_public_ip_address = true
  vpc_security_group_ids      = [aws_security_group.nodes.id]
  iam_instance_profile        = local.environment == "prod" ? aws_iam_instance_profile.node[0].name : null
  key_name                    = aws_key_pair.deployer.key_name

  user_data_replace_on_change = true

  user_data = local.environment == "prod" ? templatefile("${path.module}/templates/geth.sh.tftpl", {
    chain_id            = var.chain_id
    clique_period       = var.clique_period
    region              = var.aws_region
    project             = var.project_name
    environment         = local.environment
    node_count          = var.node_count
    extradata           = local.extradata
    alloc_entries       = local.alloc_entries
    node_index          = count.index
    signer_address      = local.signers[count.index].address
    signer_private_key  = local.signers[count.index].private_key
    nodekey_private_key = local.signers[count.index].nodekey
    nodekey_enode_0     = local.signers[0].nodekey_enode
    nodekey_enode_1     = local.signers[1].nodekey_enode
    nodekey_enode_2     = local.signers[2].nodekey_enode
    }) : templatefile("${path.module}/templates/anvil.sh.tftpl", {
    chain_id = var.chain_id
  })

  root_block_device {
    volume_type = "gp3"
    volume_size = var.ebs_volume_size
  }

  tags = merge(local.common_tags, {
    Name = "${var.project_name}-${local.environment}-node-${count.index}"
    Role = local.environment == "prod" ? (count.index == 0 ? "bootnode" : "signer") : "dev"
  })

}
