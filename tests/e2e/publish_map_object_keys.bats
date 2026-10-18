setup_file() {
  if [[ -z "${BIDIREKT_BROKER_URL:-}" ]]; then
    echo "BIDIREKT_BROKER_URL is required" >&2
    return 1
  fi

  export RUN="$(date +%s)$$"
  export API="todo_api_$RUN"

  for fixture in "$BATS_TEST_DIRNAME"/fixtures/publish_map_object_keys/*.yaml; do
    envsubst '$API' < "$fixture" > "$BATS_FILE_TMPDIR/${fixture##*/}"
  done
}

setup() {
  bats_load_library bats-support
  bats_load_library bats-assert
  cd "$BATS_FILE_TMPDIR" || return 1
}

@test "create participant todo_api" {
  run bidirekt create-participant "$API"
  assert_success
}

@test "publish todo_api v1" {
  run bidirekt publish ./api_v1.yaml --participant "$API" --version v1
  assert_success
}

@test "publish todo_api v2 (labels without values): blocked, map without values" {
  run bidirekt publish ./api_v2.yaml --participant "$API" --version v2
  assert_failure
  assert_output --partial "contract validation failed
  - ./api_v2.yaml: map schema without values at schemas Pet properties labels"
}

@test "publish todo_api v3 (properties on the labels map): blocked, properties not allowed on a map" {
  run bidirekt publish ./api_v3.yaml --participant "$API" --version v3
  assert_failure
  assert_output --partial "contract validation failed
  - ./api_v3.yaml: \"properties\" is not allowed on a map schema at schemas Pet properties labels, use it with \"type: object\""
}

@test "publish todo_api v4 (values on the labels object): blocked, values not allowed on an object, object without properties" {
  run bidirekt publish ./api_v4.yaml --participant "$API" --version v4
  assert_failure
  assert_output --partial "contract validation failed
  - ./api_v4.yaml: \"values\" is not allowed on an object schema at schemas Pet properties labels, use it with \"type: map\"
  - ./api_v4.yaml: object schema without properties at schemas Pet properties labels, declare the keys you read, or use \"type: map\" with \"values\" if the keys vary"
}
