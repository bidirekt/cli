setup_file() {
  if [[ -z "${BIDIREKT_BROKER_URL:-}" ]]; then
    echo "BIDIREKT_BROKER_URL is required" >&2
    return 1
  fi

  export RUN="$(date +%s)$$"
  export API="petstore_api_$RUN" PRODUCTION="production_$RUN"

  cp "$BATS_TEST_DIRNAME"/fixtures/graphql_publish_client_operations/* "$BATS_FILE_TMPDIR/"
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

@test "petstore_api v1 rejected: the file holds a client query, not the schema" {
  run bidirekt publish ./operations.graphql --participant "$API" --version v1
  assert_failure
  assert_output --partial "cannot post contract to broker: ./operations.graphql: holds client operations; publish expects the GraphQL schema"
}

@test "petstore_api v1 rejected: the file holds only client fragments" {
  run bidirekt publish ./fragments.graphql --participant "$API" --version v1
  assert_failure
  assert_output --partial "cannot post contract to broker: ./fragments.graphql: holds client operations; publish expects the GraphQL schema"
}

@test "petstore_api v1 rejected: the client query is cut off, at the place it ends" {
  run bidirekt publish ./broken.graphql --participant "$API" --version v1
  assert_failure
  assert_output --partial "cannot post contract to broker: ./broken.graphql:5:1: Expected Name, found <EOF> in client operations; publish expects the GraphQL schema"
}

@test "v1 was not stored" {
  run bidirekt can-i-deploy "$API" --version v1 --environment "$PRODUCTION"
  assert_failure
  assert_output --partial "contract not found"
}
