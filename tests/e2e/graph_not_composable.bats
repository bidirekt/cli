setup_file() {
  if [[ -z "${BIDIREKT_BROKER_URL:-}" ]]; then
    echo "BIDIREKT_BROKER_URL is required" >&2
    return 1
  fi

  export RUN="$(date +%s)$$"
  export API="petstore_api_$RUN" REVIEWS="petstore_reviews_$RUN" WEB="petstore_web_$RUN"
  export GRAPH="petstore_graph_$RUN" PRODUCTION="production_$RUN"

  for fixture in "$BATS_TEST_DIRNAME"/fixtures/graph_not_composable/*; do
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

@test "publish petstore_reviews v2 (also resolves Pet.name, without @shareable)" {
  run bidirekt publish ./graph.yaml ./reviews_v2.graphqls --participant "$REVIEWS" --version v2
  assert_success
}

@test "petstore_reviews v2 blocked: it makes the graph fail to compose" {
  run bidirekt can-i-deploy "$REVIEWS" --version v2 --environment "$PRODUCTION"
  assert_failure
  assert_output --partial "$REVIEWS v2 cannot be deployed to $PRODUCTION

$WEB (v1, deployed):
  query pet
    - $REVIEWS (via $GRAPH) makes $GRAPH fail to compose in $PRODUCTION: $API v1 and $REVIEWS v2 both resolve Pet.name without @shareable, and $WEB calls query pet
      → $REVIEWS: make $GRAPH compose (see /graphs/$GRAPH)"
}

@test "record petstore_reviews v2 deployed to production" {
  run bidirekt record-deployment "$REVIEWS" --version v2 --environment "$PRODUCTION"
  assert_success
}

@test "petstore_web v1 blocked: the deployed graph doesn't compose" {
  run bidirekt can-i-deploy "$WEB" --version v1 --environment "$PRODUCTION"
  assert_failure
  assert_output --partial "$WEB v1 cannot be deployed to $PRODUCTION

$GRAPH ($API v1 + $REVIEWS v2, deployed):
  query pet
    - $WEB calls query pet, but $GRAPH can't be built in $PRODUCTION: $API v1 and $REVIEWS v2 both resolve Pet.name without @shareable
      → $GRAPH subgraphs: make it compose (see /graphs/$GRAPH)"
}
