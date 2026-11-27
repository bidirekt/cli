setup_file() {
  if [[ -z "${BIDIREKT_BROKER_URL:-}" ]]; then
    echo "BIDIREKT_BROKER_URL is required" >&2
    return 1
  fi

  export RUN="$(date +%s)$$"
  export API="petstore_api_$RUN"

  cp "$BATS_TEST_DIRNAME"/fixtures/graphql_publish_scalar_in_rest/*.yaml "$BATS_FILE_TMPDIR"
}

setup() {
  bats_load_library bats-support
  bats_load_library bats-assert
  cd "$BATS_FILE_TMPDIR" || return 1
}

@test "create participant petstore_api" {
  run bidirekt create-participant "$API"
  assert_success
}

@test "petstore_api v1 rejected: a scalar in a schema a REST endpoint reads" {
  run bidirekt publish ./api_v1.yaml --participant "$API" --version v1
  assert_failure
  assert_output --partial "contract validation failed
  - ./api_v1.yaml: type \"scalar\" is only allowed in GraphQL contracts at schemas Pet properties bornAt"
}

@test "validate api_v1.yaml: the same refusal" {
  run bidirekt validate ./api_v1.yaml
  assert_failure
  assert_output --partial "contract validation failed
  - ./api_v1.yaml: type \"scalar\" is only allowed in GraphQL contracts at schemas Pet properties bornAt"
}

@test "publish petstore_api v2 (bornAt as a string)" {
  run bidirekt publish ./api_v2.yaml --participant "$API" --version v2
  assert_success
}
