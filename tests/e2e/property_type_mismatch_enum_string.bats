setup_file() {
  if [[ -z "${BIDIREKT_BROKER_URL:-}" ]]; then
    echo "BIDIREKT_BROKER_URL is required" >&2
    return 1
  fi

  export RUN="$(date +%s)$$"
  export API="petstore_api_$RUN" APP="petstore_web_$RUN" PRODUCTION="production_$RUN"

  for fixture in "$BATS_TEST_DIRNAME"/fixtures/property_type_mismatch_enum_string/*.yaml; do
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

@test "publish petstore_web v2 (reads and sends status as enum)" {
  run bidirekt publish ./app_v2.yaml --participant "$APP" --version v2
  assert_success
}

@test "petstore_web v2 blocked: reads enum where petstore_api provides string" {
  run bidirekt can-i-deploy "$APP" --version v2 --environment "$PRODUCTION"
  assert_failure
  assert_output --partial "Pet.status — $APP reads it as enum, but $API provides string (2 places)
    → $APP: read it as string
    → $API: provide it as enum there
      · GET /pets response 200 at \"\$[].status\" ($API: Pet.status)
      · POST /pets response 201 at \"\$.status\" ($API: Pet.status)"
}

@test "petstore_web v2: sending enum where petstore_api expects string is no break" {
  run bidirekt can-i-deploy "$APP" --version v2 --environment "$PRODUCTION"
  assert_failure
  refute_output --partial "CreatePetRequest.status"
}

@test "publish petstore_web v3 (status back to string)" {
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

@test "publish petstore_api v2 (provides and expects status as enum)" {
  run bidirekt publish ./api_v2.yaml --participant "$API" --version v2
  assert_success
}

@test "petstore_api v2 blocked: expects enum where petstore_web sends string" {
  run bidirekt can-i-deploy "$API" --version v2 --environment "$PRODUCTION"
  assert_failure
  assert_output --partial "CreatePetRequest.status — $API expects it as enum, but $APP sends string
    → $API: accept string
    → $APP: send enum there
      · POST /pets request at \"\$.status\" ($APP: CreatePetRequest.status)"
}

@test "petstore_api v2: providing enum where petstore_web reads string is no break" {
  run bidirekt can-i-deploy "$API" --version v2 --environment "$PRODUCTION"
  assert_failure
  refute_output --partial "Pet.status"
}

@test "publish petstore_api v3 (status back to string)" {
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
