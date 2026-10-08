setup_file() {
  if [[ -z "${BIDIREKT_BROKER_URL:-}" ]]; then
    echo "BIDIREKT_BROKER_URL is required" >&2
    return 1
  fi

  export RUN="$(date +%s)$$"
  export API="todo_api_$RUN" APP="todo_app_$RUN" PRODUCTION="production_$RUN"

  for fixture in "$BATS_TEST_DIRNAME"/fixtures/provider_resource_removed_but_still_consumed/*.yaml; do
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

@test "publish todo_api v2 (without DELETE /todos/* and without /todos/search)" {
  run bidirekt publish ./api_v2.yaml --participant "$API" --version v2
  assert_success
}

@test "todo_api v2 blocked: two removed resources still consumed by todo_app" {
  run bidirekt can-i-deploy "$API" --version v2 --environment "$PRODUCTION"
  assert_failure
  assert_output --partial "but $APP still calls it → keep it until $APP stops calling it"
  assert_equal "$(grep -cF -- "but $APP still calls it → keep it until $APP stops calling it" <<< "$output")" 2
}

@test "publish todo_app v2 (stops consuming DELETE /todos/* and /todos/search)" {
  run bidirekt publish ./app_v2.yaml --participant "$APP" --version v2
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

@test "todo_api v2 can now be deployed to production" {
  run bidirekt can-i-deploy "$API" --version v2 --environment "$PRODUCTION"
  assert_success
}

@test "record todo_api v2 deployed to production" {
  run bidirekt record-deployment "$API" --version v2 --environment "$PRODUCTION"
  assert_success
}
