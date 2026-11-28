setup_file() {
  if [[ -z "${BIDIREKT_BROKER_URL:-}" ]]; then
    echo "BIDIREKT_BROKER_URL is required" >&2
    return 1
  fi

  export RUN="$(date +%s)$$"
  export API="todo_api_$RUN" APP="todo_app_$RUN" PRODUCTION="production_$RUN"

  local fixtures="$BATS_TEST_DIRNAME/fixtures/folder_arguments"
  while IFS= read -r fixture; do
    mkdir -p "$BATS_FILE_TMPDIR/${fixture%/*}"
    envsubst '$API $APP' < "$fixtures/$fixture" > "$BATS_FILE_TMPDIR/$fixture"
  done < <(cd "$fixtures" && find . -type f)
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

@test "create participant todo_api" {
  run bidirekt create-participant "$API"
  assert_success
}

@test "create participant todo_app" {
  run bidirekt create-participant "$APP"
  assert_success
}

@test "validate the todo_api contracts folder with no argument: its subfolder is read, its README is not" {
  cd api || return 1
  run bidirekt validate --participant "$API" --environment "$PRODUCTION"
  assert_success
  assert_output --partial "$API local contract can be deployed to $PRODUCTION
nothing deployed in $PRODUCTION to compare with"
}

@test "publish todo_api v1 from api/contracts" {
  run bidirekt publish ./api/contracts --participant "$API" --version v1
  assert_success
}

@test "todo_api v1 can be deployed to production" {
  run bidirekt can-i-deploy "$API" --version v1 --environment "$PRODUCTION"
  assert_success
}

@test "record todo_api v1 deployed to production" {
  run bidirekt record-deployment "$API" --version v1 --environment "$PRODUCTION"
  assert_success
}

@test "publish todo_app v1 from app/contracts" {
  run bidirekt publish ./app/contracts --participant "$APP" --version v1
  assert_success
}

@test "todo_app v1 can be deployed to production: the todo_api contracts folder published both endpoints" {
  run bidirekt can-i-deploy "$APP" --version v1 --environment "$PRODUCTION"
  assert_success
}

@test "publish todo_app v2 with no argument: the contracts folder here is read" {
  cd app || return 1
  run bidirekt publish --participant "$APP" --version v2
  assert_success
}

@test "todo_app v2 can be deployed to production" {
  run bidirekt can-i-deploy "$APP" --version v2 --environment "$PRODUCTION"
  assert_success
}

@test "publish todo_api v2 from its contracts folder and a file inside it: the file is read once" {
  run bidirekt publish ./api/contracts ./api/contracts/todos.yaml --participant "$API" --version v2
  assert_success
}

@test "validate a contracts folder with a typo: the report names the file by the folder and its path inside it" {
  run bidirekt validate ./typo/contracts --participant "$API" --environment "$PRODUCTION"
  assert_failure
  assert_output --partial "contract validation failed
  - typo/contracts/todos.yaml: unresolved schema \"Todso\" referenced by provides GET /todos 200"
}

@test "validate with no argument and no contracts folder here: fails" {
  run bidirekt validate --participant "$API" --environment "$PRODUCTION"
  assert_failure 1
  assert_output --partial "no contract files given and no contracts/ folder here"
}

@test "publish with no argument and no contracts folder here: fails" {
  run bidirekt publish --participant "$API" --version v3
  assert_failure 1
  assert_output --partial "no contract files given and no contracts/ folder here"
}

@test "validate a folder with another name: fails with the hint to pass its files or move them" {
  run bidirekt validate ./docs --participant "$API" --environment "$PRODUCTION"
  assert_failure 1
  assert_output --partial "docs/ is not the contracts folder: pass its files by name, or move them to contracts/"
}

@test "publish a folder with another name: fails with the hint to pass its files or move them" {
  run bidirekt publish ./docs --participant "$API" --version v3
  assert_failure 1
  assert_output --partial "docs/ is not the contracts folder: pass its files by name, or move them to contracts/"
}

@test "publish a subfolder of the contracts folder alone: fails with the hint to pass the contracts folder" {
  run bidirekt publish ./api/contracts/schemas --participant "$API" --version v3
  assert_failure 1
  assert_output --partial "api/contracts/schemas/ is inside the contracts folder: pass api/contracts/ instead"
}

@test "validate a contracts folder without contract files: fails naming the folder" {
  run bidirekt validate ./notes/contracts --participant "$API" --environment "$PRODUCTION"
  assert_failure 1
  assert_output --partial "no contract files (.yaml, .yml, .graphql, .graphqls, .gql) in folder: \"./notes/contracts\""
}

@test "publish a contracts folder without contract files: fails naming the folder" {
  run bidirekt publish ./notes/contracts --participant "$API" --version v3
  assert_failure 1
  assert_output --partial "no contract files (.yaml, .yml, .graphql, .graphqls, .gql) in folder: \"./notes/contracts\""
}
