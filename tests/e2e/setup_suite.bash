setup_suite() {
  if [[ -z "${BIDIREKT_BROKER_URL:-}" ]]; then
    echo "BIDIREKT_BROKER_URL is required" >&2
    return 1
  fi

  local expected health found
  expected="$(sed -n 's|^ *image: ghcr.io/bidirekt/broker:||p' "$BATS_TEST_DIRNAME/compose.yaml")"

  if ! health="$(curl -fs --max-time 5 "$BIDIREKT_BROKER_URL/health")"; then
    echo "e2e expects broker $expected, but $BIDIREKT_BROKER_URL/health did not answer" >&2
    return 1
  fi

  found="$(sed -n 's/.*"brokerVersion":"\([^"]*\)".*/\1/p' <<< "$health")"
  if [[ "$found" != "$expected" ]]; then
    echo "e2e expects broker $expected, found $found at $BIDIREKT_BROKER_URL" >&2
    return 1
  fi
}
