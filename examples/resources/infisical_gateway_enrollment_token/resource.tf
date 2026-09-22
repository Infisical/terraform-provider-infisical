terraform {
  required_providers {
    infisical = {
      # version = <latest version>
      source = "infisical/infisical"
    }
    aws = {
      source  = "hashicorp/aws"
      version = ">= 5.0"
    }
  }
}

provider "infisical" {
  host = "https://app.infisical.com" # Only required if using self hosted instance of Infisical, default is https://app.infisical.com
  auth = {
    universal = {
      client_id     = "<machine-identity-client-id>"
      client_secret = "<machine-identity-client-secret>"
    }
  }
}

resource "infisical_gateway" "datacenter" {
  name = "datacenter-01"

  token_auth = {}
}

# A new AMI replaces the instance, which re-runs user data and needs an unused token.
data "aws_ami" "ubuntu" {
  most_recent = true
  owners      = ["099720109477"]

  filter {
    name   = "name"
    values = ["ubuntu/images/hvm-ssd/ubuntu-jammy-22.04-amd64-server-*"]
  }
}

# keepers is the only thing that re-mints. List every input that rebuilds the machine; the
# instance itself cannot be referenced, since it consumes the token.
resource "infisical_gateway_enrollment_token" "datacenter" {
  gateway_id = infisical_gateway.datacenter.id

  keepers = {
    ami_id    = data.aws_ami.ubuntu.id
    subnet_id = var.subnet_id
  }
}

module "gateway" {
  source  = "Infisical/infisical-gateway/aws"
  version = "~> 0.1"

  gateway_name     = infisical_gateway.datacenter.name
  infisical_domain = "https://app.infisical.com"
  vpc_id           = var.vpc_id
  subnet_id        = var.subnet_id

  auth = {
    method = "token"
    token  = infisical_gateway_enrollment_token.datacenter.token
  }
}
