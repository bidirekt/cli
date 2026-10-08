setup_file() {
  if [[ -z "${BIDIREKT_BROKER_URL:-}" ]]; then
    echo "BIDIREKT_BROKER_URL is required" >&2
    return 1
  fi

  export RUN="$(date +%s)$$"
  export API="todo_api_$RUN" APP="todo_app_$RUN" STAGING="staging_$RUN" PRODUCTION="production_$RUN"

  for fixture in "$BATS_TEST_DIRNAME"/fixtures/validate/*.yaml; do
    envsubst '$API $APP' < "$fixture" > "$BATS_FILE_TMPDIR/${fixture##*/}"
  done
}

setup() {
  bats_load_library bats-support
  bats_load_library bats-assert
  cd "$BATS_FILE_TMPDIR" || return 1
}

# Each pair is created in reverse byte order, so the listings prove sorting, not insertion order.
@test "create environment staging" {
  run bidirekt create-environment "$STAGING"
  assert_success
}

@test "create environment production" {
  run bidirekt create-environment "$PRODUCTION"
  assert_success
}

@test "create participant todo_app" {
  run bidirekt create-participant "$APP"
  assert_success
}

@test "create participant todo_api" {
  run bidirekt create-participant "$API"
  assert_success
}

@test "lists environments in byte order" {
  run bidirekt list-environments
  assert_success
  assert_line "$PRODUCTION"
  assert_line "$STAGING"
  assert_equal "$(grep -xF -e "$PRODUCTION" -e "$STAGING" <<< "$output")" "$PRODUCTION
$STAGING"
}

@test "lists participants in byte order" {
  run bidirekt list-participants
  assert_success
  assert_line "$API"
  assert_line "$APP"
  assert_equal "$(grep -xF -e "$API" -e "$APP" <<< "$output")" "$API
$APP"
}

@test "publish todo_api 1.0.0" {
  run bidirekt publish ./api_v1.yaml --participant "$API" --version 1.0.0
  assert_success
}

@test "publish todo_app 1.0.0" {
  run bidirekt publish ./app_v1.yaml --participant "$APP" --version 1.0.0
  assert_success
}

@test "todo_api 1.0.0 can be deployed to production" {
  run bidirekt can-i-deploy "$API" --version 1.0.0 --environment "$PRODUCTION"
  assert_success
}

@test "record todo_api 1.0.0 deployed to production" {
  run bidirekt record-deployment "$API" --version 1.0.0 --environment "$PRODUCTION"
  assert_success
}

@test "todo_app 1.0.0 can be deployed to production" {
  run bidirekt can-i-deploy "$APP" --version 1.0.0 --environment "$PRODUCTION"
  assert_success
}

@test "record todo_app 1.0.0 deployed to production" {
  run bidirekt record-deployment "$APP" --version 1.0.0 --environment "$PRODUCTION"
  assert_success
}

@test "validate api_v2.yaml: blocked, todo_app reads status" {
  run bidirekt validate ./api_v2.yaml --participant "$API" --environment "$PRODUCTION"
  assert_failure
  assert_output --partial "$API local contract cannot be deployed to $PRODUCTION

$APP (1.0.0, deployed):
  GET /pets/*
    response 200:
      - $API doesn't provide \"\$.status\", but $APP reads it → keep providing it"
}

@test "publish todo_api 2.0.0 (without status)" {
  run bidirekt publish ./api_v2.yaml --participant "$API" --version 2.0.0
  assert_success
}

@test "todo_api 2.0.0 blocked: same tree as validate" {
  run bidirekt can-i-deploy "$API" --version 2.0.0 --environment "$PRODUCTION"
  assert_failure
  assert_output --partial "$API 2.0.0 cannot be deployed to $PRODUCTION

$APP (1.0.0, deployed):
  GET /pets/*
    response 200:
      - $API doesn't provide \"\$.status\", but $APP reads it → keep providing it"
}

@test "publish todo_api 3.0.0 (optional status)" {
  run bidirekt publish ./api_v3.yaml --participant "$API" --version 3.0.0
  assert_success
}

@test "todo_api 3.0.0 blocked: status only sometimes" {
  run bidirekt can-i-deploy "$API" --version 3.0.0 --environment "$PRODUCTION"
  assert_failure
  assert_output --partial "$API provides \"\$.status\" only sometimes, but $APP requires it → keep it required"
}

@test "validate api_v4.yaml: can be deployed to production" {
  run bidirekt validate ./api_v4.yaml --participant "$API" --environment "$PRODUCTION"
  assert_success
  assert_output --partial "$API local contract can be deployed to $PRODUCTION"
}

@test "publish todo_api 4.0.0 (required status and tags)" {
  run bidirekt publish ./api_v4.yaml --participant "$API" --version 4.0.0
  assert_success
}

@test "todo_api 4.0.0 can be deployed to production" {
  run bidirekt can-i-deploy "$API" --version 4.0.0 --environment "$PRODUCTION"
  assert_success
}

@test "record todo_api 4.0.0 deployed to production" {
  run bidirekt record-deployment "$API" --version 4.0.0 --environment "$PRODUCTION"
  assert_success
}

@test "validate api_v5.yaml: blocked, todo_app still calls POST /pets" {
  run bidirekt validate ./api_v5.yaml --participant "$API" --environment "$PRODUCTION"
  assert_failure
  assert_output --partial "$API local contract cannot be deployed to $PRODUCTION

$APP (1.0.0, deployed):
  POST /pets
    request:
      - $API removed POST /pets, but $APP still calls it → keep it until $APP stops calling it
    response 201:
      - $API removed POST /pets, but $APP still calls it → keep it until $APP stops calling it"
}

@test "publish todo_api 5.0.0 (without POST /pets)" {
  run bidirekt publish ./api_v5.yaml --participant "$API" --version 5.0.0
  assert_success
}

@test "todo_api 5.0.0 blocked: same tree as validate" {
  run bidirekt can-i-deploy "$API" --version 5.0.0 --environment "$PRODUCTION"
  assert_failure
  assert_output --partial "$API 5.0.0 cannot be deployed to $PRODUCTION

$APP (1.0.0, deployed):
  POST /pets
    request:
      - $API removed POST /pets, but $APP still calls it → keep it until $APP stops calling it
    response 201:
      - $API removed POST /pets, but $APP still calls it → keep it until $APP stops calling it"
}

@test "validate api_typo.yaml: blocked, unresolved schema" {
  run bidirekt validate ./api_typo.yaml --participant "$API" --environment "$PRODUCTION"
  assert_failure
  assert_output --partial "contract validation failed
  - ./api_typo.yaml: unresolved schema \"Pets\" referenced by provides GET /pets/* 200"
}

@test "validate with nonexistent participant: blocked" {
  run bidirekt validate ./api_v4.yaml --participant "ghost_$RUN" --environment "$PRODUCTION"
  assert_failure
  assert_output --partial "cannot validate contract on broker: participant not found"
}

@test "validate with nonexistent environment: blocked" {
  run bidirekt validate ./api_v4.yaml --participant "$API" --environment "nowhere_$RUN"
  assert_failure
  assert_output --partial "cannot validate contract on broker: environment not found"
}
