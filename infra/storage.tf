# Trip report photos. The app serves them itself, at /photos/<id>/<size>.jpg,
# so the bucket isn't public: only the service account can read and write
# it. us-west1 is in Cloud Storage's always free tier (5 GB).
resource "google_storage_bucket" "photos" {
  name                        = "${var.project}-trip-photos"
  location                    = upper(var.region)
  storage_class               = "STANDARD"
  uniform_bucket_level_access = true
  public_access_prevention    = "enforced"

  depends_on = [google_project_service.services]
}

# The service stores, reads and deletes photos.
resource "google_storage_bucket_iam_member" "run_photos" {
  bucket = google_storage_bucket.photos.name
  role   = "roles/storage.objectUser"
  member = google_service_account.run.member
}

# A cache of the elevation tiles downloaded from the AWS Terrain Tiles
# dataset, as the file system of Cloud Run instances and job executions
# doesn't outlive them. Only the service account, which the service and the
# import job run as, can read and write it.
resource "google_storage_bucket" "dem" {
  name                        = "${var.project}-dem-tiles"
  location                    = upper(var.region)
  storage_class               = "STANDARD"
  uniform_bucket_level_access = true
  public_access_prevention    = "enforced"

  depends_on = [google_project_service.services]
}

# The service and the import job read and add tiles.
resource "google_storage_bucket_iam_member" "run_dem" {
  bucket = google_storage_bucket.dem.name
  role   = "roles/storage.objectUser"
  member = google_service_account.run.member
}
