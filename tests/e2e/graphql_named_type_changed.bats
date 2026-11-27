setup_file() {
  if [[ -z "${BIDIREKT_BROKER_URL:-}" ]]; then
    echo "BIDIREKT_BROKER_URL is required" >&2
    return 1
  fi

  export RUN="$(date +%s)$$"
  export API="petstore_api_$RUN" WEB="petstore_web_$RUN" PRODUCTION="production_$RUN"

  for fixture in "$BATS_TEST_DIRNAME"/fixtures/graphql_named_type_changed/*; do
    envsubst '$API' < "$fixture" > "$BATS_FILE_TMPDIR/${fixture##*/}"
  done
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

@test "create participant petstore_web" {
  run bidirekt create-participant "$WEB"
  assert_success
}

@test "publish petstore_api v1" {
  run bidirekt publish ./api_v1.graphqls --participant "$API" --version v1
  assert_success
}

@test "petstore_api v1 can be deployed to production" {
  run bidirekt can-i-deploy "$API" --version v1 --environment "$PRODUCTION"
  assert_success
}

@test "record petstore_api v1 deployed to production" {
  run bidirekt record-deployment "$API" --version v1 --environment "$PRODUCTION"
  assert_success
}

@test "publish petstore_web v1" {
  run bidirekt publish ./web_v1.yaml --participant "$WEB" --version v1
  assert_success
}

@test "petstore_web v1 can be deployed to production" {
  run bidirekt can-i-deploy "$WEB" --version v1 --environment "$PRODUCTION"
  assert_success
}

@test "record petstore_web v1 deployed to production" {
  run bidirekt record-deployment "$WEB" --version v1 --environment "$PRODUCTION"
  assert_success
}

@test "publish petstore_api v2 (the id argument becomes a String)" {
  run bidirekt publish ./api_v2.graphqls --participant "$API" --version v2
  assert_success
}

@test "petstore_api v2 blocked: petstore_web declares id as an ID" {
  run bidirekt can-i-deploy "$API" --version v2 --environment "$PRODUCTION"
  assert_failure
  assert_output --partial "$API v2 cannot be deployed to $PRODUCTION

$WEB (v1, deployed):
  Query.pet(id:) — $API expects it as String, but $WEB sends ID
    → $API: accept ID
    → $WEB: send String there
      · query pet arguments (argument \"id\") ($WEB: PetArgs.id)"
}

@test "validate api_v2.graphqls: the same break" {
  run bidirekt validate ./api_v2.graphqls --participant "$API" --environment "$PRODUCTION"
  assert_failure
  assert_output --partial "$API local contract cannot be deployed to $PRODUCTION

$WEB (v1, deployed):
  Query.pet(id:) — $API expects it as String, but $WEB sends ID
    → $API: accept ID
    → $WEB: send String there
      · query pet arguments (argument \"id\") ($WEB: PetArgs.id)"
}

@test "record petstore_api v2 deployed to production anyway" {
  run bidirekt record-deployment "$API" --version v2 --environment "$PRODUCTION"
  assert_success
}

@test "petstore_web v1 can be deployed against the deployed petstore_api v2: the consumer declares no type names" {
  run bidirekt can-i-deploy "$WEB" --version v1 --environment "$PRODUCTION"
  assert_success
  assert_output --partial "$WEB v1 can be deployed to $PRODUCTION
compared with: $API"
}
