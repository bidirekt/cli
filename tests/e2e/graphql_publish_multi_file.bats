setup_file() {
  if [[ -z "${BIDIREKT_BROKER_URL:-}" ]]; then
    echo "BIDIREKT_BROKER_URL is required" >&2
    return 1
  fi

  export RUN="$(date +%s)$$"
  export API="petstore_api_$RUN" WEB="petstore_web_$RUN" PRODUCTION="production_$RUN"

  cp -R "$BATS_TEST_DIRNAME"/fixtures/graphql_publish_multi_file/api "$BATS_FILE_TMPDIR/api"
  envsubst '$API' < "$BATS_TEST_DIRNAME"/fixtures/graphql_publish_multi_file/web_v1.yaml > "$BATS_FILE_TMPDIR/web_v1.yaml"
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

@test "create participant petstore_api" {
  run bidirekt create-participant "$API"
  assert_success
}

@test "create participant petstore_web" {
  run bidirekt create-participant "$WEB"
  assert_success
}

@test "petstore_api v1 rejected: the query file alone doesn't define Pet" {
  run bidirekt publish ./api/query.graphqls --participant "$API" --version v1
  assert_failure
  assert_output --partial "./api/query.graphqls:2:17: Undefined type Pet."
}

@test "publish petstore_api v1 (the query file and the types file)" {
  run bidirekt publish ./api/query.graphqls ./api/types.graphqls --participant "$API" --version v1
  assert_success
}

@test "petstore_api v1 can be deployed to production" {
  run bidirekt can-i-deploy "$API" --version v1 --environment "$PRODUCTION"
  assert_success
}

@test "record petstore_api v1 deployed to production" {
  run bidirekt record-deployment "$API" --version v1 --environment "$PRODUCTION"
  assert_success
}

@test "publish petstore_api v2 (the folder of both files)" {
  run bidirekt publish ./api --participant "$API" --version v2
  assert_success
}

@test "petstore_api v2 can be deployed to production" {
  run bidirekt can-i-deploy "$API" --version v2 --environment "$PRODUCTION"
  assert_success
}

@test "record petstore_api v2 deployed to production" {
  run bidirekt record-deployment "$API" --version v2 --environment "$PRODUCTION"
  assert_success
}

@test "publish petstore_web v1" {
  run bidirekt publish ./web_v1.yaml --participant "$WEB" --version v1
  assert_success
}

@test "petstore_web v1 can be deployed: Pet.name comes from the types file" {
  run bidirekt can-i-deploy "$WEB" --version v1 --environment "$PRODUCTION"
  assert_success
  assert_output --partial "$WEB v1 can be deployed to $PRODUCTION
compared with: $API"
}

@test "record petstore_web v1 deployed to production" {
  run bidirekt record-deployment "$WEB" --version v1 --environment "$PRODUCTION"
  assert_success
}
