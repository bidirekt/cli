setup_file() {
  if [[ -z "${BIDIREKT_BROKER_URL:-}" ]]; then
    echo "BIDIREKT_BROKER_URL is required" >&2
    return 1
  fi

  export RUN="$(date +%s)$$"
  export API="todo_api_$RUN" TASKS_API="tasks_api_$RUN"
  export BIDIREKT_CONFIG_FILE="$BATS_FILE_TMPDIR/config.json"
}

setup() {
  bats_load_library bats-support
  bats_load_library bats-assert
}

@test "configure profile e2e" {
  run bidirekt configure --profile "e2e_$RUN" --broker-url "$BIDIREKT_BROKER_URL"
  assert_success
}

@test "uses the broker of profile e2e without BIDIREKT_BROKER_URL" {
  run env -u BIDIREKT_BROKER_URL bidirekt list-participants --profile "e2e_$RUN"
  assert_success
  assert_line "Broker: $BIDIREKT_BROKER_URL (profile: e2e_$RUN)"
}

@test "fails without BIDIREKT_BROKER_URL or profile" {
  run env -u BIDIREKT_BROKER_URL -u BIDIREKT_PROFILE bidirekt list-participants
  assert_failure 1
  assert_output --partial "no broker configured — pass --broker-url, set BIDIREKT_BROKER_URL, or run \"bidirekt configure\""
}

@test "create participant todo_api" {
  run bidirekt create-participant "$API"
  assert_success
}

@test "rename todo_api to tasks_api" {
  run bidirekt rename-participant "$API" "$TASKS_API"
  assert_success
  assert_output --partial "$API participant renamed to $TASKS_API"
}

@test "lists tasks_api instead of todo_api" {
  run bidirekt list-participants
  assert_success
  assert_line "$TASKS_API"
  refute_line "$API"
}

@test "shows the dev version" {
  run bidirekt version
  assert_success
  assert_output "bidirekt version dev"
}
