setup_file() {
  if [[ -z "${BIDIREKT_BROKER_URL:-}" ]]; then
    echo "BIDIREKT_BROKER_URL is required" >&2
    return 1
  fi

  export RUN="$(date +%s)$$"
  export API="petstore_api_$RUN" ORDERS="petstore_orders_$RUN" WEB="petstore_web_$RUN"
  export GRAPH="petstore_graph_$RUN" PRODUCTION="production_$RUN"

  for fixture in "$BATS_TEST_DIRNAME"/fixtures/graph_subgraph_fixes_composition/*; do
    envsubst '$GRAPH' < "$fixture" > "$BATS_FILE_TMPDIR/${fixture##*/}"
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

@test "create participant petstore_orders" {
  run bidirekt create-participant "$ORDERS"
  assert_success
}

@test "create participant petstore_web" {
  run bidirekt create-participant "$WEB"
  assert_success
}

@test "publish petstore_api v1 (Money without @shareable)" {
  run bidirekt publish ./graph.yaml ./api_v1.graphqls --participant "$API" --version v1
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

@test "publish petstore_web v1 (reads age as a string)" {
  run bidirekt publish ./web_v1.yaml --participant "$WEB" --version v1
  assert_success
}

@test "petstore_web v1 blocked: the graph provides age as an integer" {
  run bidirekt can-i-deploy "$WEB" --version v1 --environment "$PRODUCTION"
  assert_failure
  assert_output --partial "$WEB v1 cannot be deployed to $PRODUCTION

$GRAPH ($API v1, deployed):
  Pet.age — $WEB reads it as string, but $GRAPH provides integer
    → $WEB: read it as integer
    → $GRAPH: provide it as string there
      · query pet returns at \"\$.age\" ($GRAPH: Pet.age)"
}

@test "record petstore_web v1 deployed to production" {
  run bidirekt record-deployment "$WEB" --version v1 --environment "$PRODUCTION"
  assert_success
}

@test "publish petstore_orders v1 (Money with @shareable)" {
  run bidirekt publish ./graph.yaml ./orders_v1.graphqls --participant "$ORDERS" --version v1
  assert_success
}

@test "petstore_orders v1 blocked: it makes the graph fail to compose" {
  run bidirekt can-i-deploy "$ORDERS" --version v1 --environment "$PRODUCTION"
  assert_failure
  assert_output --partial "$ORDERS v1 cannot be deployed to $PRODUCTION

$WEB (v1, deployed):
  query pet
    - $ORDERS (via $GRAPH) makes $GRAPH fail to compose in $PRODUCTION: $API v1 and $ORDERS v1 both resolve Money.amount without @shareable, and $WEB calls query pet
      → $ORDERS: make $GRAPH compose (see /graphs/$GRAPH)"
}

@test "record petstore_orders v1 deployed to production" {
  run bidirekt record-deployment "$ORDERS" --version v1 --environment "$PRODUCTION"
  assert_success
}

@test "publish petstore_orders v2 (drops Money, so the graph composes again)" {
  run bidirekt publish ./graph.yaml ./orders_v2.graphqls --participant "$ORDERS" --version v2
  assert_success
}

@test "petstore_orders v2 can be deployed: the age break of petstore_web is not its own" {
  run bidirekt can-i-deploy "$ORDERS" --version v2 --environment "$PRODUCTION"
  assert_success
  assert_output --partial "compared with: $WEB"
}
