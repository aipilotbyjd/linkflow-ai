# LinkFlow AI - Terraform Outputs

# VPC Outputs
output "vpc_id" {
  description = "VPC ID"
  value       = module.vpc.vpc_id
}

output "vpc_cidr_block" {
  description = "VPC CIDR block"
  value       = module.vpc.vpc_cidr_block
}

output "private_subnet_ids" {
  description = "Private subnet IDs"
  value       = module.vpc.private_subnets
}

output "public_subnet_ids" {
  description = "Public subnet IDs"
  value       = module.vpc.public_subnets
}

# EKS Outputs
output "eks_cluster_id" {
  description = "EKS cluster ID"
  value       = module.eks.cluster_id
}

output "eks_cluster_name" {
  description = "EKS cluster name"
  value       = module.eks.cluster_name
}

output "eks_cluster_endpoint" {
  description = "EKS cluster endpoint"
  value       = module.eks.cluster_endpoint
}

output "eks_cluster_security_group_id" {
  description = "EKS cluster security group ID"
  value       = module.eks.cluster_security_group_id
}

output "eks_cluster_certificate_authority_data" {
  description = "EKS cluster CA data"
  value       = module.eks.cluster_certificate_authority_data
  sensitive   = true
}

output "eks_oidc_provider_arn" {
  description = "EKS OIDC provider ARN"
  value       = module.eks.oidc_provider_arn
}

# Database Outputs
output "rds_endpoint" {
  description = "RDS endpoint"
  value       = module.rds.db_instance_endpoint
  sensitive   = true
}

output "rds_port" {
  description = "RDS port"
  value       = module.rds.db_instance_port
}

output "rds_database_name" {
  description = "RDS database name"
  value       = module.rds.db_instance_name
}

# Cache Outputs
output "redis_endpoint" {
  description = "ElastiCache Redis endpoint"
  value       = module.elasticache.cluster_address
}

output "redis_port" {
  description = "ElastiCache Redis port"
  value       = 6379
}

# Kafka Outputs
output "kafka_bootstrap_brokers_tls" {
  description = "Kafka TLS bootstrap brokers"
  value       = aws_msk_cluster.kafka.bootstrap_brokers_tls
  sensitive   = true
}

output "kafka_zookeeper_connect_string" {
  description = "Kafka Zookeeper connection string"
  value       = aws_msk_cluster.kafka.zookeeper_connect_string
  sensitive   = true
}

# Storage Outputs
output "s3_bucket_name" {
  description = "S3 storage bucket name"
  value       = aws_s3_bucket.storage.id
}

output "s3_bucket_arn" {
  description = "S3 storage bucket ARN"
  value       = aws_s3_bucket.storage.arn
}

# Connection Strings (for Kubernetes secrets)
output "database_url" {
  description = "PostgreSQL connection URL"
  value       = "postgresql://${module.rds.db_instance_username}:PASSWORD@${module.rds.db_instance_endpoint}/${module.rds.db_instance_name}?sslmode=require"
  sensitive   = true
}

output "redis_url" {
  description = "Redis connection URL"
  value       = "rediss://${module.elasticache.cluster_address}:6379"
  sensitive   = true
}

# Kubeconfig
output "kubeconfig_command" {
  description = "Command to configure kubectl"
  value       = "aws eks update-kubeconfig --region ${var.aws_region} --name ${module.eks.cluster_name}"
}
