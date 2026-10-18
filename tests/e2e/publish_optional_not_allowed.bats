setup_file() {
  if [[ -z "${BIDIREKT_BROKER_URL:-}" ]]; then
    echo "BIDIREKT_BROKER_URL is required" >&2
    return 1
  fi

  export RUN="$(date +%s)$$"
  export API="todo_api_$RUN"

  for fixture in "$BATS_TEST_DIRNAME"/fixtures/publish_optional_not_allowed/*.yaml; do
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

@test "publish todo_api v2 (optional on the tags items): blocked, optional not allowed on array items" {
  run bidirekt publish ./api_v2.yaml --participant "$API" --version v2
  assert_failure
  assert_output --partial "contract validation failed
  - ./api_v2.yaml: \"optional\" is not allowed on array items at schemas Pet properties tags items, mark the property that holds the array optional, or use \"nullable: true\" for an item that may be null"
}

@test "publish todo_api v3 (optional on the labels values): blocked, optional not allowed on map values" {
  run bidirekt publish ./api_v3.yaml --participant "$API" --version v3
  assert_failure
  assert_output --partial "contract validation failed
  - ./api_v3.yaml: \"optional\" is not allowed on map values at schemas Pet properties labels values, mark the property that holds the map optional, or use \"nullable: true\" for a value that may be null"
}

@test "publish todo_api v4 (optional on the Pet schema): blocked, optional not allowed on a schema" {
  run bidirekt publish ./api_v4.yaml --participant "$API" --version v4
  assert_failure
  assert_output --partial "contract validation failed
  - ./api_v4.yaml: \"optional\" is not allowed on a schema at schemas Pet, mark optional the property that uses it, beside its \"ref\""
}
