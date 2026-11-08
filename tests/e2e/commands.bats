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
  bats_require_minimum_version 1.5.0
  bats_load_library bats-support
  bats_load_library bats-assert
}

@test "uses the broker and API key of profile e2e without BIDIREKT_BROKER_URL and BIDIREKT_API_KEY" {
  printf '{"profiles": {"e2e_%s": {"brokerUrl": "%s", "apiKey": "%s"}}}\n' "$RUN" "$BIDIREKT_BROKER_URL" "$BIDIREKT_API_KEY" > "$BIDIREKT_CONFIG_FILE"

  run env -u BIDIREKT_BROKER_URL -u BIDIREKT_API_KEY bidirekt list-participants --profile "e2e_$RUN"
  assert_success
  assert_line "Broker: $BIDIREKT_BROKER_URL (profile: e2e_$RUN)"
}

@test "fails without BIDIREKT_BROKER_URL, BIDIREKT_API_KEY or profile, naming profile e2e" {
  run env -u BIDIREKT_BROKER_URL -u BIDIREKT_API_KEY -u BIDIREKT_PROFILE bidirekt list-participants
  assert_failure 1
  assert_output --partial "no broker configured — run \"bidirekt login\" or set BIDIREKT_BROKER_URL and BIDIREKT_API_KEY
the config file has the profile e2e_$RUN — pass --profile e2e_$RUN or set BIDIREKT_PROFILE=e2e_$RUN"
}

@test "fails with BIDIREKT_BROKER_URL set and BIDIREKT_API_KEY unset" {
  run env -u BIDIREKT_API_KEY bidirekt list-participants
  assert_failure 1
  assert_output "set both BIDIREKT_BROKER_URL and BIDIREKT_API_KEY, or neither"
}

@test "shows only not signed in for a profile without an API key" {
  local config_file="$BATS_TEST_TMPDIR/config.json"
  printf '{"profiles": {"no_key": {"brokerUrl": "%s"}}}\n' "$BIDIREKT_BROKER_URL" > "$config_file"

  run env -u BIDIREKT_BROKER_URL -u BIDIREKT_API_KEY BIDIREKT_CONFIG_FILE="$config_file" bidirekt list-participants --profile no_key
  assert_failure 1
  assert_output "Broker: $BIDIREKT_BROKER_URL (profile: no_key)
not signed in to $BIDIREKT_BROKER_URL — run \"bidirekt login\" or set BIDIREKT_BROKER_URL and BIDIREKT_API_KEY"
}

@test "prints only the broker line on stderr for a localhost broker" {
  run --separate-stderr bidirekt list-participants
  assert_success
  assert_equal "$stderr" "Broker: $BIDIREKT_BROKER_URL (from BIDIREKT_BROKER_URL)"
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
