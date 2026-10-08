setup_file() {
  if [[ -z "${BIDIREKT_BROKER_URL:-}" ]]; then
    echo "BIDIREKT_BROKER_URL is required" >&2
    return 1
  fi

  export RUN="$(date +%s)$$"
  export API="todo_api_$RUN" APP="todo_app_$RUN" PRODUCTION="production_$RUN"

  for fixture in "$BATS_TEST_DIRNAME"/fixtures/publish_unresolved_schema/*.yaml; do
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

@test "publish todo_api v1 with all names resolving" {
  run bidirekt publish ./api_rest.yaml ./api_schemas.yaml --participant "$API" --version v1
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

@test "v2 rejected: unresolved name in a response" {
  run bidirekt publish ./api_schemas.yaml ./api_rest_typo_response.yaml --participant "$API" --version v2
  assert_failure
  assert_output --partial "./api_rest_typo_response.yaml: unresolved schema \"TodoList\" referenced by provides GET /todos 200"
}

@test "v2 was not stored" {
  run bidirekt can-i-deploy "$API" --version v2 --environment "$PRODUCTION"
  assert_failure
  assert_output --partial "contract not found"
}

@test "v3 rejected: unresolved name in a request" {
  run bidirekt publish ./api_schemas.yaml ./api_rest_typo_request.yaml --participant "$API" --version v3
  assert_failure
  assert_output --partial "./api_rest_typo_request.yaml: unresolved schema \"NewTodo\" referenced by provides POST /todos request"
}

@test "v4 rejected: dangling ref in a schema nobody uses" {
  run bidirekt publish ./api_rest.yaml ./api_schemas.yaml ./api_schemas_dangling.yaml \
    --participant "$API" --version v4
  assert_failure
  assert_output --partial "./api_schemas_dangling.yaml: unresolved ref \"Person\" in Team.owner"
}

@test "v5 rejected: cyclic schema" {
  run bidirekt publish ./api_schemas_cycle.yaml ./api_rest_cycle.yaml --participant "$API" --version v5
  assert_failure
  assert_output --partial "./api_schemas_cycle.yaml: schema \"Loop\" is deeper than 10 levels"
}

@test "todo_app rejected: typo in a single-file publish" {
  run bidirekt publish ./app_single_file_typo.yaml --participant "$APP" --version v1
  assert_failure
  assert_output --partial "./app_single_file_typo.yaml: unresolved schema \"Todso\" referenced by consumes $API GET /todos 200"
}

@test "publish todo_api v6 after all the rejections" {
  run bidirekt publish ./api_rest.yaml ./api_schemas.yaml --participant "$API" --version v6
  assert_success
}

@test "todo_api v6 can be deployed to production" {
  run bidirekt can-i-deploy "$API" --version v6 --environment "$PRODUCTION"
  assert_success
}

@test "record todo_api v6 deployed to production" {
  run bidirekt record-deployment "$API" --version v6 --environment "$PRODUCTION"
  assert_success
}
