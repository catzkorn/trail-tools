output "url" {
  description = "The public URL of the app."
  value       = local.public_url
}

output "image_repository" {
  description = "The repository to push images to, for KO_DOCKER_REPO."
  value       = local.image_repo
}

output "workload_identity_provider" {
  description = "The Workload Identity Federation provider for GitHub Actions."
  value       = google_iam_workload_identity_pool_provider.github.name
}

output "terraform_service_account" {
  description = "The service account GitHub Actions applies Terraform as."
  value       = google_service_account.terraform.email
}

output "domain_records" {
  description = "The DNS records to create for the custom domain."
  value       = local.domain != "" && var.service_enabled ? google_cloud_run_domain_mapping.domain[0].status[0].resource_records : []
}

output "terraform_plan_service_account" {
  description = "The read-only service account GitHub Actions plans pull requests as."
  value       = google_service_account.terraform_plan.email
}
