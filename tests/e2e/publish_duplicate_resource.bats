setup_file() {
  if [[ -z "${BIDIREKT_BROKER_URL:-}" ]]; then
    echo "BIDIREKT_BROKER_URL is required" >&2
    return 1
  fi

  export RUN="$(date +%s)$$"
  export API="todo_api_$RUN" APP="todo_app_$RUN" PRODUCTION="production_$RUN"

  for fixture in "$BATS_TEST_DIRNAME"/fixtures/publish_duplicate_resource/*.yaml; do
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

@test "publish todo_api v1 with the POST statuses split across files" {
  run bidirekt publish ./api_schemas.yaml ./api_rest_todos.yaml ./api_rest_todos_404.yaml \
    --participant "$API" --version v1
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

@test "v2 rejected: GET /todos 200 declared twice" {
  run bidirekt publish ./api_schemas.yaml ./api_rest_todos.yaml ./api_rest_todos_get_again.yaml \
    --participant "$API" --version v2
  assert_failure
  assert_output --partial "./api_rest_todos_get_again.yaml: duplicate resource \"provides GET /todos 200\", also declared in ./api_rest_todos.yaml"
}

@test "v2 was not stored" {
  run bidirekt can-i-deploy "$API" --version v2 --environment "$PRODUCTION"
  assert_failure
  assert_output --partial "contract not found"
}

@test "v3 rejected: POST /todos request declared twice" {
  run bidirekt publish ./api_schemas.yaml ./api_rest_todos.yaml ./api_rest_todos_post_request_again.yaml \
    --participant "$API" --version v3
  assert_failure
  assert_output --partial "./api_rest_todos_post_request_again.yaml: duplicate resource \"provides POST /todos request\", also declared in ./api_rest_todos.yaml"
}

@test "v4 rejected: /todos and /todos/ are the same endpoint" {
  run bidirekt publish ./api_schemas.yaml ./api_rest_todos.yaml ./api_rest_todos_trailing_slash.yaml \
    --participant "$API" --version v4
  assert_failure
  assert_output --partial "./api_rest_todos_trailing_slash.yaml: duplicate resource \"provides GET /todos 200\", also declared in ./api_rest_todos.yaml"
}

@test "v5 rejected: the same file passed twice" {
  run bidirekt publish ./api_schemas.yaml ./api_rest_todos_404.yaml ./api_rest_todos_404.yaml \
    --participant "$API" --version v5
  assert_failure
  assert_output --partial "./api_rest_todos_404.yaml: duplicate resource \"provides POST /todos 404\", declared twice"
}

@test "publish todo_app v1 merging the repeated consumed resource by union" {
  run bidirekt publish ./app_consumes_todos.yaml ./app_consumes_todos_again.yaml \
    --participant "$APP" --version v1
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
