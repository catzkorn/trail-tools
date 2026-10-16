# GitHub Actions applies this configuration and deploys from the main
# branch, and plans pull requests, authenticating with Workload Identity
# Federation rather than stored credentials.

resource "google_iam_workload_identity_pool" "github" {
  workload_identity_pool_id = "github"
  display_name              = "GitHub Actions"
  depends_on                = [google_project_service.services]
}

resource "google_iam_workload_identity_pool_provider" "github" {
  workload_identity_pool_id          = google_iam_workload_identity_pool.github.workload_identity_pool_id
  workload_identity_pool_provider_id = "github"
  display_name                       = "GitHub"
  attribute_mapping = {
    "google.subject"          = "assertion.sub"
    "attribute.repository"    = "assertion.repository"
    "attribute.repository_id" = "assertion.repository_id"
    "attribute.ref"           = "assertion.ref"
  }
  # Only tokens for the main branch and pull requests of the repository are
  # accepted. Pull requests from forks don't get tokens. Pull requests can
  # only use the read-only plan service account. The repository is matched by
  # its ID, which, unlike its name, survives renames and transfers.
  attribute_condition = "assertion.repository_id == '${var.github_repository_id}' && (assertion.ref == 'refs/heads/main' || assertion.event_name == 'pull_request')"
  oidc {
    issuer_uri = "https://token.actions.githubusercontent.com"
  }
}

resource "google_service_account" "terraform" {
  account_id   = "github-terraform"
  display_name = "Applies Trail Tools Terraform from GitHub Actions"
}

# Only the main branch can apply. The provider only accepts tokens from the
# repository, so this is the repository's main branch.
resource "google_service_account_iam_member" "terraform_github" {
  service_account_id = google_service_account.terraform.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "principalSet://iam.googleapis.com/${google_iam_workload_identity_pool.github.name}/attribute.ref/refs/heads/main"
}

# The roles needed to manage everything in this configuration. Managing the
# budget additionally needs roles/billing.costsManager on the billing
# account, which is granted by hand, see the README.
resource "google_project_iam_member" "terraform" {
  for_each = toset([
    "roles/artifactregistry.admin",
    "roles/cloudscheduler.admin",
    "roles/iam.serviceAccountAdmin",
    "roles/iam.serviceAccountUser",
    "roles/iam.workloadIdentityPoolAdmin",
    "roles/resourcemanager.projectIamAdmin",
    "roles/run.admin",
    "roles/secretmanager.admin",
    "roles/serviceusage.serviceUsageAdmin",
    "roles/storage.admin",
  ])
  project = var.project
  role    = each.value
  member  = google_service_account.terraform.member
}

resource "google_service_account" "terraform_plan" {
  account_id   = "github-terraform-plan"
  display_name = "Plans Trail Tools Terraform changes in GitHub Actions pull requests"
}

resource "google_service_account_iam_member" "terraform_plan_github" {
  service_account_id = google_service_account.terraform_plan.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "principalSet://iam.googleapis.com/${google_iam_workload_identity_pool.github.name}/attribute.repository_id/${var.github_repository_id}"
}

# Read-only roles for planning. Reading the budget additionally needs
# roles/billing.viewer on the billing account, which is granted by hand, see
# the README. Secret values can't be read.
resource "google_project_iam_member" "terraform_plan" {
  for_each = toset([
    "roles/viewer",
    "roles/iam.securityReviewer",
    "roles/iam.workloadIdentityPoolViewer",
    "roles/serviceusage.serviceUsageConsumer",
  ])
  project = var.project
  role    = each.value
  member  = google_service_account.terraform_plan.member
}

# Plans read the state without locking it.
resource "google_storage_bucket_iam_member" "terraform_plan_state" {
  bucket = "${var.project}-tfstate"
  role   = "roles/storage.objectViewer"
  member = google_service_account.terraform_plan.member
}
