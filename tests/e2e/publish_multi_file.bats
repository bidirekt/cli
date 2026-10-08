setup_file() {
  if [[ -z "${BIDIREKT_BROKER_URL:-}" ]]; then
    echo "BIDIREKT_BROKER_URL is required" >&2
    return 1
  fi

  export RUN="$(date +%s)$$"
  export API="todo_api_$RUN" APP="todo_app_$RUN" PRODUCTION="production_$RUN"

  for fixture in "$BATS_TEST_DIRNAME"/fixtures/publish_multi_file/*.yaml; do
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

@test "publish todo_api v1 in four fragments" {
  run bidirekt publish ./api_rest_todos.yaml ./api_rest_todo_item.yaml ./api_rest_todo_update.yaml ./api_schemas_v1.yaml \
    --participant "$API" --version v1
  assert_success
}

@test "publish todo_app v1 in a single file" {
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

@test "publish todo_api v2 (schema fragment without completed)" {
  run bidirekt publish ./api_rest_todos.yaml ./api_rest_todo_item.yaml ./api_rest_todo_update.yaml ./api_schemas_v2.yaml \
    --participant "$API" --version v2
  assert_success
}

@test "todo_api v2 blocked at the scalar path" {
  run bidirekt can-i-deploy "$API" --version v2 --environment "$PRODUCTION"
  assert_failure
  assert_output --partial "$API doesn't provide \"\$.completed\", but $APP reads it → keep providing it"
}

@test "todo_api v2 blocked at the array item path" {
  run bidirekt can-i-deploy "$API" --version v2 --environment "$PRODUCTION"
  assert_failure
  assert_output --partial "$API doesn't provide \"\$[].completed\", but $APP reads it → keep providing it"
}

@test "publish todo_app v2 (stops requiring completed)" {
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
