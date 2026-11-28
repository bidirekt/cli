setup_file() {
  if [[ -z "${BIDIREKT_BROKER_URL:-}" ]]; then
    echo "BIDIREKT_BROKER_URL is required" >&2
    return 1
  fi

  export RUN="$(date +%s)$$"
  export API="petstore_api_$RUN" REVIEWS="petstore_reviews_$RUN" WEB="petstore_web_$RUN"
  export GRAPH="petstore_graph_$RUN" PRODUCTION="production_$RUN"

  for fixture in "$BATS_TEST_DIRNAME"/fixtures/graph_subgraph_consumes_own_graph/*; do
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

@test "publish petstore_api v1 (reads Pet.reviews through the graph)" {
  run bidirekt publish ./graph.yaml ./api_v1.graphqls ./api_consumes.yaml --participant "$API" --version v1
  assert_success
}

@test "petstore_api v1 can be deployed: its own graph resolves reviews" {
  run bidirekt can-i-deploy "$API" --version v1 --environment "$PRODUCTION"
  assert_success
  assert_output --partial "compared with: $GRAPH"
}

@test "record petstore_api v1 deployed to production" {
  run bidirekt record-deployment "$API" --version v1 --environment "$PRODUCTION"
  assert_success
}

@test "publish petstore_api v2 (also resolves Pet.reviews, without @shareable)" {
  run bidirekt publish ./graph.yaml ./api_v2.graphqls ./api_consumes.yaml --participant "$API" --version v2
  assert_success
}

@test "petstore_api v2 blocked: its own graph doesn't compose with it" {
  run bidirekt can-i-deploy "$API" --version v2 --environment "$PRODUCTION"
  assert_failure
  assert_output --partial "$API v2 cannot be deployed to $PRODUCTION

$GRAPH ($API v2 candidate + $REVIEWS v1, deployed):
  query pet
    - $API calls query pet, but $GRAPH can't be built in $PRODUCTION
      $API v2 and $REVIEWS v1 both resolve Pet.reviews without @shareable
      → $GRAPH subgraphs: make it compose
      deployed graph: /graphs/$GRAPH"
}

@test "petstore_api v2: the deployed petstore_api v1 is no counterpart of it" {
  run bidirekt can-i-deploy "$API" --version v2 --environment "$PRODUCTION"
  assert_failure
  refute_output --partial "$API (v1"
}

@test "validate the petstore_api v2 files: the graph composes with the candidate" {
  run bidirekt validate ./graph.yaml ./api_v2.graphqls ./api_consumes.yaml --participant "$API" --environment "$PRODUCTION"
  assert_failure
  assert_output --partial "$API local contract cannot be deployed to $PRODUCTION

$GRAPH ($API candidate + $REVIEWS v1, deployed):
  query pet
    - $API calls query pet, but $GRAPH can't be built in $PRODUCTION
      $API candidate and $REVIEWS v1 both resolve Pet.reviews without @shareable
      → $GRAPH subgraphs: make it compose
      deployed graph: /graphs/$GRAPH"
}
