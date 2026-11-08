readonly ADMIN_EMAIL="admin@e2e.example.com"
readonly ADMIN_PASSWORD="e2e-admin-password"

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

  local cookies="$BATS_RUN_TMPDIR/cookies.txt"
  sign_in_as_admin "$cookies" || return 1

  local key
  if ! key="$(create_write_api_key "$cookies")"; then
    return 1
  fi

  export BIDIREKT_API_KEY="$key"
}

sign_in_as_admin() {
  local cookies="$1" status

  status="$(post_form "$cookies" /setup -o /dev/null -w '%{http_code}')"
  if [[ "$status" == "404" ]]; then
    status="$(post_form "$cookies" /sign-in -o /dev/null -w '%{http_code}')"
  fi

  if [[ "$status" != 3* ]]; then
    echo "e2e could not sign in as $ADMIN_EMAIL at $BIDIREKT_BROKER_URL (HTTP $status)" >&2
    return 1
  fi
}

post_form() {
  local cookies="$1" path="$2"
  shift 2

  curl -s --max-time 5 -b "$cookies" -c "$cookies" \
    -H "Origin: $BIDIREKT_BROKER_URL" \
    --data-urlencode "email=$ADMIN_EMAIL" \
    --data-urlencode "password=$ADMIN_PASSWORD" \
    "$@" "$BIDIREKT_BROKER_URL$path"
}

create_write_api_key() {
  local cookies="$1" form token created key

  if ! form="$(curl -fs --max-time 5 -b "$cookies" "$BIDIREKT_BROKER_URL/profile/api-keys/new")"; then
    echo "e2e could not open $BIDIREKT_BROKER_URL/profile/api-keys/new" >&2
    return 1
  fi
  token="$(sed -n 's/.*"X-CSRF-Token": "\([^"]*\)".*/\1/p' <<< "$form")"

  if ! created="$(curl -fs --max-time 5 -b "$cookies" \
    -H "Origin: $BIDIREKT_BROKER_URL" \
    -H "X-CSRF-Token: $token" \
    --data name=e2e --data permission=write --data expiration=never \
    "$BIDIREKT_BROKER_URL/profile/api-keys/new")"; then
    echo "e2e could not create an API key at $BIDIREKT_BROKER_URL" >&2
    return 1
  fi

  key="$(sed -n 's/.*<input class="copy" type="text" readonly value="\([^"]*\)".*/\1/p' <<< "$created")"
  if [[ -z "$key" ]]; then
    echo "e2e found no API key in the page $BIDIREKT_BROKER_URL/profile/api-keys/new answered" >&2
    return 1
  fi

  echo "$key"
}
