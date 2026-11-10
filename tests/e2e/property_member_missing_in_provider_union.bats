setup_file() {
  if [[ -z "${BIDIREKT_BROKER_URL:-}" ]]; then
    echo "BIDIREKT_BROKER_URL is required" >&2
    return 1
  fi

  export RUN="$(date +%s)$$"
  export API="shop_api_$RUN" APP="shop_web_$RUN" PRODUCTION="production_$RUN"

  for fixture in "$BATS_TEST_DIRNAME"/fixtures/property_member_missing_in_provider_union/*.yaml; do
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

@test "create participant shop_api" {
  run bidirekt create-participant "$API"
  assert_success
}

@test "create participant shop_web" {
  run bidirekt create-participant "$APP"
  assert_success
}

@test "publish shop_api v1" {
  run bidirekt publish ./api_v1.yaml --participant "$API" --version v1
  assert_success
}

@test "publish shop_web v1" {
  run bidirekt publish ./app_v1.yaml --participant "$APP" --version v1
  assert_success
}

@test "shop_api v1 can be deployed to production" {
  run bidirekt can-i-deploy "$API" --version v1 --environment "$PRODUCTION"
  assert_success
}

@test "record shop_api v1 deployed to production" {
  run bidirekt record-deployment "$API" --version v1 --environment "$PRODUCTION"
  assert_success
}

@test "shop_web v1 can be deployed to production" {
  run bidirekt can-i-deploy "$APP" --version v1 --environment "$PRODUCTION"
  assert_success
}

@test "record shop_web v1 deployed to production" {
  run bidirekt record-deployment "$APP" --version v1 --environment "$PRODUCTION"
  assert_success
}

@test "publish shop_web v2 (reads and sends the boleto member)" {
  run bidirekt publish ./app_v2.yaml --participant "$APP" --version v2
  assert_success
}

@test "shop_web v2 blocked: sends a member shop_api doesn't accept" {
  run bidirekt can-i-deploy "$APP" --version v2 --environment "$PRODUCTION"
  assert_failure
  assert_output --partial "CreateOrderRequest.payment — $APP sends \"boleto\", but $API doesn't accept it
    → $APP: stop sending \"boleto\"
    → $API: accept \"boleto\" there
      · POST /orders request at \"\$.payment\" ($API: CreateOrderRequest.payment)"
}

@test "shop_web v2 blocked: reads a member shop_api doesn't provide" {
  run bidirekt can-i-deploy "$APP" --version v2 --environment "$PRODUCTION"
  assert_failure
  assert_output --partial "Order.payment — $APP reads \"boleto\", but $API doesn't provide it (2 places)
    → $APP: stop reading \"boleto\"
    → $API: provide \"boleto\" there
      · POST /orders response 201 at \"\$.payment\" ($API: Order.payment)
      · GET /orders/* response 200 at \"\$.payment\" ($API: Order.payment)"
}

@test "publish shop_web v3 (members back to v1)" {
  run bidirekt publish ./app_v3.yaml --participant "$APP" --version v3
  assert_success
}

@test "shop_web v3 can be deployed to production" {
  run bidirekt can-i-deploy "$APP" --version v3 --environment "$PRODUCTION"
  assert_success
}

@test "record shop_web v3 deployed to production" {
  run bidirekt record-deployment "$APP" --version v3 --environment "$PRODUCTION"
  assert_success
}

@test "publish shop_api v2 (drops the card member)" {
  run bidirekt publish ./api_v2.yaml --participant "$API" --version v2
  assert_success
}

@test "shop_api v2 blocked: doesn't accept a member shop_web sends" {
  run bidirekt can-i-deploy "$API" --version v2 --environment "$PRODUCTION"
  assert_failure
  assert_output --partial "CreateOrderRequest.payment — $API doesn't accept \"card\", but $APP sends it
    → $API: accept \"card\"
    → $APP: stop sending \"card\" there
      · POST /orders request at \"\$.payment\" ($APP: CreateOrderRequest.payment)"
}

@test "shop_api v2 blocked: doesn't provide a member shop_web reads" {
  run bidirekt can-i-deploy "$API" --version v2 --environment "$PRODUCTION"
  assert_failure
  assert_output --partial "Order.payment — $API doesn't provide \"card\", but $APP reads it (2 places)
    → $API: provide \"card\"
    → $APP: stop reading \"card\" there
      · POST /orders response 201 at \"\$.payment\" ($APP: Order.payment)
      · GET /orders/* response 200 at \"\$.payment\" ($APP: Order.payment)"
}

@test "publish shop_api v3 (members back to v1)" {
  run bidirekt publish ./api_v3.yaml --participant "$API" --version v3
  assert_success
}

@test "shop_api v3 can be deployed to production" {
  run bidirekt can-i-deploy "$API" --version v3 --environment "$PRODUCTION"
  assert_success
}

@test "record shop_api v3 deployed to production" {
  run bidirekt record-deployment "$API" --version v3 --environment "$PRODUCTION"
  assert_success
}
