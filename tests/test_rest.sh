#!/usr/bin/env bats

source ./tests/commands/delete_object_tagging.sh
source ./tests/commands/get_object.sh
source ./tests/commands/list_buckets.sh
source ./tests/commands/put_object.sh
source ./tests/commands/put_object_tagging.sh
source ./tests/logger.sh
source ./tests/setup.sh
source ./tests/util.sh
source ./tests/util_list_buckets.sh
source ./tests/util_list_objects.sh
source ./tests/util_rest.sh
source ./tests/util_tags.sh

@test "test_rest_list_objects" {
  run setup_bucket "s3api" "$BUCKET_ONE_NAME"
  assert_success

  test_file="test_file"
  run create_test_files "$test_file"
  assert_success

  run put_object "s3api" "$TEST_FILE_FOLDER/$test_file" "$BUCKET_ONE_NAME" "$test_file"
  assert_success

  run list_check_objects_rest "$BUCKET_ONE_NAME"
  assert_success
}

@test "test_rest_list_buckets" {
  run setup_bucket "s3api" "$BUCKET_ONE_NAME"
  assert_success

  run list_check_buckets_rest
  assert_success
}

@test "test_rest_delete_object" {
  run setup_bucket "s3api" "$BUCKET_ONE_NAME"
  assert_success

  test_file="test_file"
  run create_test_files "$test_file"
  assert_success

  run put_object "rest" "$TEST_FILE_FOLDER/$test_file" "$BUCKET_ONE_NAME" "$test_file"
  assert_success

  run get_object "rest" "$BUCKET_ONE_NAME" "$test_file" "$TEST_FILE_FOLDER/$test_file-copy"
  assert_success

  run compare_files "$TEST_FILE_FOLDER/$test_file" "$TEST_FILE_FOLDER/$test_file-copy"
  assert_success

  run delete_object "rest" "$BUCKET_ONE_NAME" "$test_file"
  assert_success

  run get_object "rest" "$BUCKET_ONE_NAME" "$test_file" "$TEST_FILE_FOLDER/$test_file-copy"
  assert_failure
}

@test "test_rest_tagging" {
  test_file="test_file"
  test_key="TestKey"
  test_value="TestValue"

  run setup_bucket "s3api" "$BUCKET_ONE_NAME"
  assert_success

  run create_test_files "$test_file"
  assert_success

  run put_object "rest" "$TEST_FILE_FOLDER/$test_file" "$BUCKET_ONE_NAME" "$test_file"
  assert_success

  run put_object_tagging "rest" "$BUCKET_ONE_NAME" "$test_file" "$test_key" "$test_value"
  assert_success

  run check_verify_object_tags "rest" "$BUCKET_ONE_NAME" "$test_file" "$test_key" "$test_value"
  assert_success

  run delete_object_tagging "rest" "$BUCKET_ONE_NAME" "$test_file"
  assert_success

  run verify_no_object_tags "rest" "$BUCKET_ONE_NAME" "$test_file"
  assert_success
}
