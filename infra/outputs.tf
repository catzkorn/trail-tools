output "url" {
  description = "The public URL of the app."
  value       = local.public_url
}

output "image_repository" {
  description = "The repository to push images to, for KO_DOCKER_REPO."
  value       = local.image_repo
}

output "domain_records" {
  description = "The DNS records to create for the custom domain."
  value       = local.domain != "" && var.service_enabled ? google_cloud_run_domain_mapping.domain[0].status[0].resource_records : []
}
