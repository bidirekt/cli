setup_file() {
  if [[ -z "${BIDIREKT_BROKER_URL:-}" ]]; then
    echo "BIDIREKT_BROKER_URL is required" >&2
    return 1
  fi

  export RUN="$(date +%s)$$"
  export API="todo_api_$RUN"
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

@test "fails without BIDIREKT_BROKER_URL or profile, naming profile e2e" {
  run env -u BIDIREKT_BROKER_URL -u BIDIREKT_PROFILE bidirekt list-participants
  assert_failure 1
  assert_output --partial "no broker configured — pass --broker-url, set BIDIREKT_BROKER_URL, or run \"bidirekt configure\"
the config file has the profile e2e_$RUN — pass --profile e2e_$RUN or set BIDIREKT_PROFILE=e2e_$RUN"
}

@test "create participant todo_api" {
  run bidirekt create-participant "$API"
  assert_success
}

@test "shows the dev version" {
  run bidirekt version
  assert_success
  assert_output "bidirekt version dev"
}
