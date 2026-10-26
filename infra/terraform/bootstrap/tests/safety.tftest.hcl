mock_provider "aws" {
  override_during = plan
  mock_resource "aws_s3_bucket" { defaults = { arn = "arn:aws:s3:::offline-state-test" } }
}

run "state_safety" {
  command = plan
  variables { state_bucket = "offline-state-test" }
  assert {
    condition     = aws_s3_bucket_versioning.state.versioning_configuration[0].status == "Enabled" && !aws_s3_bucket.state.force_destroy && aws_s3_bucket_public_access_block.state.block_public_policy
    error_message = "State must be versioned, private and protected against forced deletion."
  }
  assert {
    condition     = one(aws_s3_bucket_server_side_encryption_configuration.state.rule).apply_server_side_encryption_by_default[0].sse_algorithm == "AES256"
    error_message = "State encryption is mandatory."
  }
}
