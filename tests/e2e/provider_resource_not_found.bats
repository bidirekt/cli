setup_file() {
  if [[ -z "${BIDIREKT_BROKER_URL:-}" ]]; then
    echo "BIDIREKT_BROKER_URL is required" >&2
    return 1
  fi

  export RUN="$(date +%s)$$"
  export API="todo_api_$RUN" APP="todo_app_$RUN" PRODUCTION="production_$RUN"

  for fixture in "$BATS_TEST_DIRNAME"/fixtures/provider_resource_not_found/*.yaml; do
    envsubst '$API $APP' < "$fixture" > "$BATS_FILE_TMPDIR/${fixture##*/}"
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

@test "publish todo_app v2 (PUT /todos + GET /todos/stats)" {
  run bidirekt publish ./app_v2.yaml --participant "$APP" --version v2
  assert_success
}

@test "todo_app v2 blocked: method and endpoint not provided" {
  run bidirekt can-i-deploy "$APP" --version v2 --environment "$PRODUCTION"
  assert_failure
  assert_output --partial "but $API doesn't provide it → stop calling it, or wait until $API publishes it"
  assert_equal "$(grep -cF -- "but $API doesn't provide it → stop calling it, or wait until $API publishes it" <<< "$output")" 3
}

@test "publish todo_api v2 (provides PUT /todos + GET /todos/stats)" {
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
}

@test "todo_app v2 can be deployed to production" {
  run bidirekt can-i-deploy "$APP" --version v2 --environment "$PRODUCTION"
  assert_success
}

@test "record todo_app v2 deployed to production" {
  run bidirekt record-deployment "$APP" --version v2 --environment "$PRODUCTION"
  assert_success
}

@test "publish todo_app v3 (consumes billing, which doesn't exist)" {
  run bidirekt publish ./app_v3.yaml --participant "$APP" --version v3
  assert_success
}

@test "todo_app v3 blocked: participant billing doesn't exist" {
  run bidirekt can-i-deploy "$APP" --version v3 --environment "$PRODUCTION"
  assert_failure
  assert_output --partial "$APP calls GET /invoices, but billing doesn't provide it → stop calling it, or wait until billing publishes it"
}

@test "publish todo_app v4 (without billing)" {
  run bidirekt publish ./app_v4.yaml --participant "$APP" --version v4
  assert_success
}

@test "todo_app v4 can be deployed to production" {
  run bidirekt can-i-deploy "$APP" --version v4 --environment "$PRODUCTION"
  assert_success
}

@test "record todo_app v4 deployed to production" {
  run bidirekt record-deployment "$APP" --version v4 --environment "$PRODUCTION"
  assert_success
}
