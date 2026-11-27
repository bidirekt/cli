setup_file() {
  if [[ -z "${BIDIREKT_BROKER_URL:-}" ]]; then
    echo "BIDIREKT_BROKER_URL is required" >&2
    return 1
  fi

  export RUN="$(date +%s)$$"
  export API="petstore_api_$RUN" REVIEWS="petstore_reviews_$RUN" WEB="petstore_web_$RUN"
  export GRAPH="petstore_graph_$RUN" PRODUCTION="production_$RUN"

  for fixture in "$BATS_TEST_DIRNAME"/fixtures/graph_consumer_type_mismatch/*; do
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

@test "create participant petstore_reviews" {
  run bidirekt create-participant "$REVIEWS"
  assert_success
}

@test "create participant petstore_web" {
  run bidirekt create-participant "$WEB"
  assert_success
}

@test "publish petstore_api v1" {
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

@test "publish petstore_reviews v1" {
  run bidirekt publish ./graph.yaml ./reviews_v1.graphqls --participant "$REVIEWS" --version v1
  assert_success
}

@test "petstore_reviews v1 can be deployed to production" {
  run bidirekt can-i-deploy "$REVIEWS" --version v1 --environment "$PRODUCTION"
  assert_success
}

@test "record petstore_reviews v1 deployed to production" {
  run bidirekt record-deployment "$REVIEWS" --version v1 --environment "$PRODUCTION"
  assert_success
}

@test "publish petstore_web v1 (reads name from petstore_api and reviews from petstore_reviews)" {
  run bidirekt publish ./web_v1.yaml --participant "$WEB" --version v1
  assert_success
}

@test "petstore_web v1 can be deployed: compared with the graph" {
  run bidirekt can-i-deploy "$WEB" --version v1 --environment "$PRODUCTION"
  assert_success
  assert_output --partial "compared with: $GRAPH"
}

@test "record petstore_web v1 deployed to production" {
  run bidirekt record-deployment "$WEB" --version v1 --environment "$PRODUCTION"
  assert_success
}

@test "publish petstore_web v2 (reads age as a string)" {
  run bidirekt publish ./web_v2.yaml --participant "$WEB" --version v2
  assert_success
}

@test "petstore_web v2 blocked: the graph provides age as an integer" {
  run bidirekt can-i-deploy "$WEB" --version v2 --environment "$PRODUCTION"
  assert_failure
  assert_output --partial "$WEB v2 cannot be deployed to $PRODUCTION

$GRAPH ($API v1 + $REVIEWS v1, deployed):
  Pet.age — $WEB reads it as String, but $GRAPH provides Int
    → $WEB: read it as Int
    → $GRAPH: provide it as String there
      · query pet returns at \"\$.age\" ($GRAPH: Pet.age)"
}

@test "validate web_v2.yaml: the same break against the graph" {
  run bidirekt validate ./web_v2.yaml --participant "$WEB" --environment "$PRODUCTION"
  assert_failure
  assert_output --partial "$WEB local contract cannot be deployed to $PRODUCTION

$GRAPH ($API v1 + $REVIEWS v1, deployed):
  Pet.age — $WEB reads it as String, but $GRAPH provides Int
    → $WEB: read it as Int
    → $GRAPH: provide it as String there
      · query pet returns at \"\$.age\" ($GRAPH: Pet.age)"
}
