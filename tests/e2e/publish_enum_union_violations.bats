setup_file() {
  if [[ -z "${BIDIREKT_BROKER_URL:-}" ]]; then
    echo "BIDIREKT_BROKER_URL is required" >&2
    return 1
  fi

  export RUN="$(date +%s)$$"
  export API="shop_api_$RUN" APP="shop_web_$RUN"

  for fixture in "$BATS_TEST_DIRNAME"/fixtures/publish_enum_union_violations/*.yaml; do
    envsubst '$API' < "$fixture" > "$BATS_FILE_TMPDIR/${fixture##*/}"
  done
}

setup() {
  bats_load_library bats-support
  bats_load_library bats-assert
  cd "$BATS_FILE_TMPDIR" || return 1
}

@test "create participant shop_api" {
  run bidirekt create-participant "$API"
  assert_success
}

@test "create participant shop_web" {
  run bidirekt create-participant "$APP"
  assert_success
}

@test "publish shop_api v1 (enum status, union payment)" {
  run bidirekt publish ./api_v1.yaml --participant "$API" --version v1
  assert_success
}

@test "publish shop_api v2 (enum without members): blocked, members required" {
  run bidirekt publish ./api_v2.yaml --participant "$API" --version v2
  assert_failure
  assert_output --partial "contract validation failed
  - ./api_v2.yaml: \"type: enum\" needs \"members\" at schemas Order properties status: list its values"
}

@test "publish shop_api v3 (union without members): blocked, members required" {
  run bidirekt publish ./api_v3.yaml --participant "$API" --version v3
  assert_failure
  assert_output --partial "contract validation failed
  - ./api_v3.yaml: \"type: union\" needs \"members\" at schemas Order properties payment: map each key to a ref"
}

@test "publish shop_api v4 (enum with an empty members list): blocked, members empty" {
  run bidirekt publish ./api_v4.yaml --participant "$API" --version v4
  assert_failure
  assert_output --partial "contract validation failed
  - ./api_v4.yaml: \"members\" is empty at schemas Order properties status: list at least one value"
}

@test "publish shop_api v5 (union with an empty members map): blocked, members empty" {
  run bidirekt publish ./api_v5.yaml --participant "$API" --version v5
  assert_failure
  assert_output --partial "contract validation failed
  - ./api_v5.yaml: \"members\" is empty at schemas Order properties payment: list at least one member"
}

@test "publish shop_api v6 (enum value paid twice): blocked, members duplicate" {
  run bidirekt publish ./api_v6.yaml --participant "$API" --version v6
  assert_failure
  assert_output --partial "contract validation failed
  - ./api_v6.yaml: value \"paid\" is repeated in \"members\" at schemas Order properties status"
}

@test "publish shop_api v7 (empty enum value): blocked, members not text" {
  run bidirekt publish ./api_v7.yaml --participant "$API" --version v7
  assert_failure
  assert_output --partial "contract validation failed
  - ./api_v7.yaml: each value in \"members\" must be non-empty text at schemas Order properties status"
}

@test "publish shop_api v8 (members on a string): blocked, key not allowed" {
  run bidirekt publish ./api_v8.yaml --participant "$API" --version v8
  assert_failure
  assert_output --partial "contract validation failed
  - ./api_v8.yaml: \"members\" is not allowed on a string schema at schemas Order properties status, use it with \"type: enum\" or \"type: union\""
}

@test "publish shop_api v9 (union member without ref): blocked, member ref required" {
  run bidirekt publish ./api_v9.yaml --participant "$API" --version v9
  assert_failure
  assert_output --partial "contract validation failed
  - ./api_v9.yaml: member \"card\" needs a \"ref\" to a schema at schemas Order properties payment"
}

@test "publish shop_api v10 (union member key card(x)): blocked, invalid member key" {
  run bidirekt publish ./api_v10.yaml --participant "$API" --version v10
  assert_failure
  assert_output --partial "contract validation failed
  - ./api_v10.yaml: member key \"card(x)\" must not contain \".\", \"[\", \"]\", \"{\", \"}\", \";\", \"(\" or \")\" at schemas Order properties payment"
}

@test "publish shop_api v11 (union member with type beside ref): blocked, unknown key" {
  run bidirekt publish ./api_v11.yaml --participant "$API" --version v11
  assert_failure
  assert_output --partial "contract validation failed
  - ./api_v11.yaml: unknown key \"type\" at schemas Order properties payment members card"
}

@test "publish shop_api v12 (union member ref to a missing schema): blocked, unresolved ref" {
  run bidirekt publish ./api_v12.yaml --participant "$API" --version v12
  assert_failure
  assert_output --partial "contract validation failed
  - ./api_v12.yaml: unresolved ref \"BoletoPayment\" in Order.payment(card)"
}

@test "publish shop_web v1 (status values differ between fragments): blocked, conflicting values" {
  run bidirekt publish ./app_v1_status_paid.yaml ./app_v1_status_shipped.yaml --participant "$APP" --version v1
  assert_failure
  assert_output --partial "contract validation failed
  - ./app_v1_status_shipped.yaml: conflicting values for property \"\$.status\" of consumes $API GET /orders/* 200: [placed, shipped] here, [paid, placed] in ./app_v1_status_paid.yaml"
}
