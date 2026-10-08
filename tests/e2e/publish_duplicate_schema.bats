setup_file() {
  if [[ -z "${BIDIREKT_BROKER_URL:-}" ]]; then
    echo "BIDIREKT_BROKER_URL is required" >&2
    return 1
  fi

  export RUN="$(date +%s)$$"
  export API="todo_api_$RUN" PRODUCTION="production_$RUN"

  for fixture in "$BATS_TEST_DIRNAME"/fixtures/publish_duplicate_schema/*.yaml; do
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

@test "publish todo_api v1 with the schemas split across files" {
  run bidirekt publish ./api_rest.yaml ./api_schemas_todo.yaml ./api_schemas_message.yaml \
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

@test "v2 rejected: Todo declared in two schema fragments" {
  run bidirekt publish ./api_rest.yaml ./api_schemas_todo.yaml ./api_schemas_todo_again.yaml \
    --participant "$API" --version v2
  assert_failure
  assert_output --partial "./api_schemas_todo_again.yaml: duplicate schema \"Todo\", also declared in ./api_schemas_todo.yaml"
}

@test "v2 was not stored" {
  run bidirekt can-i-deploy "$API" --version v2 --environment "$PRODUCTION"
  assert_failure
  assert_output --partial "contract not found"
}

@test "v3 rejected: inline Todo in the rest fragment collides with the one in the schema fragment" {
  run bidirekt publish ./api_rest_with_inline_schema.yaml ./api_schemas_todo.yaml \
    --participant "$API" --version v3
  assert_failure
  assert_output --partial "./api_schemas_todo.yaml: duplicate schema \"Todo\", also declared in ./api_rest_with_inline_schema.yaml"
}

@test "publish todo_api v4 without the repeated name" {
  run bidirekt publish ./api_rest.yaml ./api_schemas_todo.yaml ./api_schemas_message.yaml \
    --participant "$API" --version v4
  assert_success
}

@test "todo_api v4 can be deployed to production" {
  run bidirekt can-i-deploy "$API" --version v4 --environment "$PRODUCTION"
  assert_success
}

@test "record todo_api v4 deployed to production" {
  run bidirekt record-deployment "$API" --version v4 --environment "$PRODUCTION"
  assert_success
}
