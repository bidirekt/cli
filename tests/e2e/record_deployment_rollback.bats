setup_file() {
  if [[ -z "${BIDIREKT_BROKER_URL:-}" ]]; then
    echo "BIDIREKT_BROKER_URL is required" >&2
    return 1
  fi

  export RUN="$(date +%s)$$"
  export API="todo_api_$RUN" PRODUCTION="production_$RUN"

  for fixture in "$BATS_TEST_DIRNAME"/fixtures/record_deployment_rollback/*.yaml; do
    envsubst '$API' < "$fixture" > "$BATS_FILE_TMPDIR/${fixture##*/}"
  done
}

setup() {
  bats_load_library bats-support
  bats_load_library bats-assert
  cd "$BATS_FILE_TMPDIR" || return 1
}

@test "create environment production" {
  run bidirekt create-environment "$PRODUCTION"
  assert_success
}

@test "create participant todo_api" {
  run bidirekt create-participant "$API"
  assert_success
}

@test "publish todo_api v1" {
  run bidirekt publish ./api_v1.yaml --participant "$API" --version v1
  assert_success
}

@test "todo_api v1 can be deployed to production" {
  run bidirekt can-i-deploy "$API" --version v1 --environment "$PRODUCTION"
  assert_success
}

@test "record todo_api v1 deployed to production" {
  run bidirekt record-deployment "$API" --version v1 --environment "$PRODUCTION"
  assert_success
  assert_line "$API deployment recorded to $PRODUCTION"
}

@test "publish todo_api v2 (optional description)" {
  run bidirekt publish ./api_v2.yaml --participant "$API" --version v2
  assert_success
}

@test "todo_api v2 can be deployed to production" {
  run bidirekt can-i-deploy "$API" --version v2 --environment "$PRODUCTION"
  assert_success
}

@test "record todo_api v2 deployed to production" {
  run bidirekt record-deployment "$API" --version v2 --environment "$PRODUCTION"
  assert_success
  assert_line "$API deployment recorded to $PRODUCTION"
}

@test "todo_api v1 can be deployed to production again" {
  run bidirekt can-i-deploy "$API" --version v1 --environment "$PRODUCTION"
  assert_success
}

@test "record todo_api v1 deployed to production again (rollback)" {
  run bidirekt record-deployment "$API" --version v1 --environment "$PRODUCTION"
  assert_success
  assert_line "$API rolled back to v1 in $PRODUCTION"
}
