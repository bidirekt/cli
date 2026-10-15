setup_file() {
  if [[ -z "${BIDIREKT_BROKER_URL:-}" ]]; then
    echo "BIDIREKT_BROKER_URL is required" >&2
    return 1
  fi

  export RUN="$(date +%s)$$"
  export API="todo_api_$RUN" APP="todo_app_$RUN" PRODUCTION="production_$RUN"

  for fixture in "$BATS_TEST_DIRNAME"/fixtures/provider_resource_interaction_not_found/*.yaml; do
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

@test "todo_api v1 can be deployed to production" {
  run bidirekt can-i-deploy "$API" --version v1 --environment "$PRODUCTION"
  assert_success
}

@test "record todo_api v1 deployed to production" {
  run bidirekt record-deployment "$API" --version v1 --environment "$PRODUCTION"
  assert_success
}

@test "validate app_v1.yaml: blocked, todo_api lacks the statuses and the request body todo_app uses" {
  run bidirekt validate ./app_v1.yaml --participant "$APP" --environment "$PRODUCTION"
  assert_failure
  assert_output --partial "$APP local contract cannot be deployed to $PRODUCTION

$API (v1, deployed):
  POST /todos/*/archive
    response 200:
      - $APP reads response 200, but $API answers only 204 or 404
        → $APP: read response 204 or 404
        → $API: answer 200
  POST /todos/*/attachments
    response 201:
      - $APP reads response 201, but $API declares no response
        → $APP: stop reading it
        → $API: answer 201
  POST /todos/*/restore
    request:
      - $APP sends a request body, but $API declares none
        → $APP: stop sending it
        → $API: accept a request body"
}

@test "publish todo_app v1" {
  run bidirekt publish ./app_v1.yaml --participant "$APP" --version v1
  assert_success
}

@test "todo_app v1 blocked: same tree as validate" {
  run bidirekt can-i-deploy "$APP" --version v1 --environment "$PRODUCTION"
  assert_failure
  assert_output --partial "$APP v1 cannot be deployed to $PRODUCTION

$API (v1, deployed):
  POST /todos/*/archive
    response 200:
      - $APP reads response 200, but $API answers only 204 or 404
        → $APP: read response 204 or 404
        → $API: answer 200
  POST /todos/*/attachments
    response 201:
      - $APP reads response 201, but $API declares no response
        → $APP: stop reading it
        → $API: answer 201
  POST /todos/*/restore
    request:
      - $APP sends a request body, but $API declares none
        → $APP: stop sending it
        → $API: accept a request body"
}

@test "publish todo_app v2 (reads the statuses todo_api answers, sends no body to restore)" {
  run bidirekt publish ./app_v2.yaml --participant "$APP" --version v2
  assert_success
}

@test "todo_app v2 can be deployed to production" {
  run bidirekt can-i-deploy "$APP" --version v2 --environment "$PRODUCTION"
  assert_success
  assert_output --partial "$APP v2 can be deployed to $PRODUCTION
compared with: $API"
}
