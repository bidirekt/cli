setup_file() {
  if [[ -z "${BIDIREKT_BROKER_URL:-}" ]]; then
    echo "BIDIREKT_BROKER_URL is required" >&2
    return 1
  fi

  export RUN="$(date +%s)$$"
  export API="todo_api_$RUN" APP="todo_app_$RUN" STAGING="staging_$RUN" PRODUCTION="production_$RUN"

  for fixture in "$BATS_TEST_DIRNAME"/fixtures/provider_resource_not_deployed_in_environment/*.yaml; do
    envsubst '$API $APP' < "$fixture" > "$BATS_FILE_TMPDIR/${fixture##*/}"
  done
}

setup() {
  bats_load_library bats-support
  bats_load_library bats-assert
  cd "$BATS_FILE_TMPDIR" || return 1
}

@test "create environment staging" {
  run bidirekt create-environment "$STAGING"
  assert_success
}

@test "create environment production" {
  run bidirekt create-environment "$PRODUCTION"
  assert_success
}

@test "create participant todo_api" {
  run bidirekt create-participant "$API"
  assert_success
}

@test "create participant todo_app" {
  run bidirekt create-participant "$APP"
  assert_success
}

@test "publish todo_api v1" {
  run bidirekt publish ./api_v1.yaml --participant "$API" --version v1
  assert_success
}

@test "publish todo_app v1" {
  run bidirekt publish ./app_v1.yaml --participant "$APP" --version v1
  assert_success
}

@test "todo_app v1 blocked: todo_api not deployed anywhere" {
  run bidirekt can-i-deploy "$APP" --version v1 --environment "$PRODUCTION"
  assert_failure
  assert_output --partial "$APP calls GET /todos, but $API is not deployed in $PRODUCTION → deploy $API first"
  refute_output --partial "deployed in:"
}

@test "todo_api v1 can be deployed to staging" {
  run bidirekt can-i-deploy "$API" --version v1 --environment "$STAGING"
  assert_success
}

@test "record todo_api v1 deployed to staging" {
  run bidirekt record-deployment "$API" --version v1 --environment "$STAGING"
  assert_success
}

@test "todo_app v1 blocked: todo_api only in staging" {
  run bidirekt can-i-deploy "$APP" --version v1 --environment "$PRODUCTION"
  assert_failure
  assert_output --partial "$APP calls GET /todos, but $API is not deployed in $PRODUCTION (deployed in: $STAGING) → deploy $API first"
}

@test "todo_api v1 can be deployed to production" {
  run bidirekt can-i-deploy "$API" --version v1 --environment "$PRODUCTION"
  assert_success
}

@test "record todo_api v1 deployed to production" {
  run bidirekt record-deployment "$API" --version v1 --environment "$PRODUCTION"
  assert_success
}

@test "todo_app v1 can be deployed to production" {
  run bidirekt can-i-deploy "$APP" --version v1 --environment "$PRODUCTION"
  assert_success
}

@test "record todo_app v1 deployed to production" {
  run bidirekt record-deployment "$APP" --version v1 --environment "$PRODUCTION"
  assert_success
}
