setup_file() {
  if [[ -z "${BIDIREKT_BROKER_URL:-}" ]]; then
    echo "BIDIREKT_BROKER_URL is required" >&2
    return 1
  fi

  export RUN="$(date +%s)$$"
  export API="todo_api_$RUN"

  for fixture in "$BATS_TEST_DIRNAME"/fixtures/publish_invalid_name/*.yaml; do
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

@test "publish todo_api v2 (property user.name): blocked, invalid property name" {
  run bidirekt publish ./api_v2.yaml --participant "$API" --version v2
  assert_failure
  assert_output --partial "contract validation failed
  - ./api_v2.yaml: invalid property name \"user.name\" at schemas Pet properties, must not contain \".\", \"[\", \"]\", \"{\", \"}\" or \";\""
}

@test "publish todo_api v3 (property tags[]): blocked, invalid property name" {
  run bidirekt publish ./api_v3.yaml --participant "$API" --version v3
  assert_failure
  assert_output --partial "contract validation failed
  - ./api_v3.yaml: invalid property name \"tags[]\" at schemas Pet properties, must not contain \".\", \"[\", \"]\", \"{\", \"}\" or \";\""
}

@test "publish todo_api v4 (property a;b): blocked, invalid property name" {
  run bidirekt publish ./api_v4.yaml --participant "$API" --version v4
  assert_failure
  assert_output --partial "contract validation failed
  - ./api_v4.yaml: invalid property name \"a;b\" at schemas Pet properties, must not contain \".\", \"[\", \"]\", \"{\", \"}\" or \";\""
}

@test "publish todo_api v5 (empty property name): blocked, invalid property name" {
  run bidirekt publish ./api_v5.yaml --participant "$API" --version v5
  assert_failure
  assert_output --partial "contract validation failed
  - ./api_v5.yaml: invalid property name \"\" at schemas Pet properties, must not be empty"
}

@test "publish todo_api v6 (property name with a tab): blocked, invalid property name" {
  run bidirekt publish ./api_v6.yaml --participant "$API" --version v6
  assert_failure
  assert_output --partial "contract validation failed
  - ./api_v6.yaml: invalid property name \"a\\tb\" at schemas Pet properties, must not contain control characters"
}

@test "publish todo_api v7 (schema Pet.v2): blocked, invalid schema name" {
  run bidirekt publish ./api_v7.yaml --participant "$API" --version v7
  assert_failure
  assert_output --partial "contract validation failed
  - ./api_v7.yaml: invalid schema name \"Pet.v2\" at schemas, must not contain \".\", \"[\", \"]\", \"{\", \"}\" or \";\""
}
