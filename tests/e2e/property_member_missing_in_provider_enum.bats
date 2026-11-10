setup_file() {
  if [[ -z "${BIDIREKT_BROKER_URL:-}" ]]; then
    echo "BIDIREKT_BROKER_URL is required" >&2
    return 1
  fi

  export RUN="$(date +%s)$$"
  export API="petstore_api_$RUN" APP="petstore_web_$RUN" PRODUCTION="production_$RUN"

  for fixture in "$BATS_TEST_DIRNAME"/fixtures/property_member_missing_in_provider_enum/*.yaml; do
    envsubst '$API $APP' < "$fixture" > "$BATS_FILE_TMPDIR/${fixture##*/}"
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
  run bidirekt create-participant "$APP"
  assert_success
}

@test "publish petstore_api v1" {
  run bidirekt publish ./api_v1.yaml --participant "$API" --version v1
  assert_success
}

@test "publish petstore_web v1" {
  run bidirekt publish ./app_v1.yaml --participant "$APP" --version v1
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

@test "petstore_web v1 can be deployed to production" {
  run bidirekt can-i-deploy "$APP" --version v1 --environment "$PRODUCTION"
  assert_success
}

@test "record petstore_web v1 deployed to production" {
  run bidirekt record-deployment "$APP" --version v1 --environment "$PRODUCTION"
  assert_success
}

@test "publish petstore_web v2 (reads adopted, sends sold)" {
  run bidirekt publish ./app_v2.yaml --participant "$APP" --version v2
  assert_success
}

@test "petstore_web v2 blocked: sends a value petstore_api doesn't accept" {
  run bidirekt can-i-deploy "$APP" --version v2 --environment "$PRODUCTION"
  assert_failure
  assert_output --partial "CreatePetRequest.status — $APP sends \"sold\", but $API doesn't accept it
    → $APP: stop sending \"sold\"
    → $API: accept \"sold\" there
      · POST /pets request at \"\$.status\" ($API: CreatePetRequest.status)"
}

@test "petstore_web v2 blocked: reads a value petstore_api doesn't provide" {
  run bidirekt can-i-deploy "$APP" --version v2 --environment "$PRODUCTION"
  assert_failure
  assert_output --partial "Pet.status — $APP reads \"adopted\", but $API doesn't provide it (2 places)
    → $APP: stop reading \"adopted\"
    → $API: provide \"adopted\" there
      · GET /pets response 200 at \"\$[].status\" ($API: Pet.status)
      · POST /pets response 201 at \"\$.status\" ($API: Pet.status)"
}

@test "publish petstore_web v3 (values back to v1)" {
  run bidirekt publish ./app_v3.yaml --participant "$APP" --version v3
  assert_success
}

@test "petstore_web v3 can be deployed to production" {
  run bidirekt can-i-deploy "$APP" --version v3 --environment "$PRODUCTION"
  assert_success
}

@test "record petstore_web v3 deployed to production" {
  run bidirekt record-deployment "$APP" --version v3 --environment "$PRODUCTION"
  assert_success
}

@test "publish petstore_api v2 (stops providing pending, stops accepting available)" {
  run bidirekt publish ./api_v2.yaml --participant "$API" --version v2
  assert_success
}

@test "petstore_api v2 blocked: doesn't accept a value petstore_web sends" {
  run bidirekt can-i-deploy "$API" --version v2 --environment "$PRODUCTION"
  assert_failure
  assert_output --partial "CreatePetRequest.status — $API doesn't accept \"available\", but $APP sends it
    → $API: accept \"available\"
    → $APP: stop sending \"available\" there
      · POST /pets request at \"\$.status\" ($APP: CreatePetRequest.status)"
}

@test "petstore_api v2 blocked: doesn't provide a value petstore_web reads" {
  run bidirekt can-i-deploy "$API" --version v2 --environment "$PRODUCTION"
  assert_failure
  assert_output --partial "Pet.status — $API doesn't provide \"pending\", but $APP reads it (2 places)
    → $API: provide \"pending\"
    → $APP: stop reading \"pending\" there
      · GET /pets response 200 at \"\$[].status\" ($APP: Pet.status)
      · POST /pets response 201 at \"\$.status\" ($APP: Pet.status)"
}

@test "publish petstore_api v3 (values back to v1)" {
  run bidirekt publish ./api_v3.yaml --participant "$API" --version v3
  assert_success
}

@test "petstore_api v3 can be deployed to production" {
  run bidirekt can-i-deploy "$API" --version v3 --environment "$PRODUCTION"
  assert_success
}

@test "record petstore_api v3 deployed to production" {
  run bidirekt record-deployment "$API" --version v3 --environment "$PRODUCTION"
  assert_success
}
