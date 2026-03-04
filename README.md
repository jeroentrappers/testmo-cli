# testmo-cli

A command-line tool for managing test cases in [Testmo](https://www.testmo.com). Provides full CRUD operations on projects, folders, and test cases, plus a YAML-based sync workflow that lets you define test cases as code and keep them in sync with your Testmo instance.

## Table of Contents

- [Installation](#installation)
- [Configuration](#configuration)
- [Quick Start](#quick-start)
- [Commands](#commands)
  - [testmo init](#testmo-init)
  - [testmo projects list](#testmo-projects-list)
  - [testmo folders](#testmo-folders)
  - [testmo cases](#testmo-cases)
  - [testmo sync](#testmo-sync)
- [YAML Sync File Format](#yaml-sync-file-format)
- [Workflows](#workflows)
  - [Exploring Your Testmo Instance](#exploring-your-testmo-instance)
  - [Managing Folders](#managing-folders)
  - [Managing Test Cases](#managing-test-cases)
  - [YAML Sync Workflow](#yaml-sync-workflow)
  - [CI/CD Integration](#cicd-integration)
- [API Details](#api-details)
- [Project Structure](#project-structure)

---

## Installation

Requires [Go 1.21+](https://go.dev/dl/).

```bash
# Clone and build
git clone <repo-url>
cd testmo-cli
go build -o testmo .

# Optionally move to PATH
sudo mv testmo /usr/local/bin/
```

## Configuration

The CLI needs two values to connect to your Testmo instance:

| Setting | Description | Example |
|---------|-------------|---------|
| **URL** | Your Testmo instance URL | `https://mycompany.testmo.net` |
| **Token** | API token for authentication | `testmo_api_eyJ...` |

### Generating an API Token

1. Log in to your Testmo instance
2. Click your avatar (top-right corner) to open your user profile
3. Find the **API access** section and click the change link
4. Generate a new token and copy it immediately (it is only shown once)

### Configuration Methods

Configuration is resolved in this order (highest priority first):

1. **Environment variables** (recommended for CI/CD)
2. **Config file** (`.testmo.yaml`, recommended for local development)

#### Option 1: Environment Variables

```bash
export TESTMO_URL="https://mycompany.testmo.net"
export TESTMO_TOKEN="testmo_api_eyJ..."
```

The CLI also accepts `TESTMO_API_TOKEN` as an alias for `TESTMO_TOKEN`.

#### Option 2: Config File via `testmo init`

```bash
testmo init
```

This prompts for your instance URL and API token, then saves them to `.testmo.yaml` in the current directory (file permissions `0600`). The CLI searches for this file starting in the current directory and walking up to parent directories, so you can place it at your project root.

Example `.testmo.yaml`:

```yaml
url: https://mycompany.testmo.net
token: testmo_api_eyJ...
```

> **Security note:** Add `.testmo.yaml` to your `.gitignore` to avoid committing credentials.

#### Option 3: Mixed

Environment variables override the config file. A common pattern is to store the URL in `.testmo.yaml` and pass the token via environment:

```yaml
# .testmo.yaml
url: https://mycompany.testmo.net
```

```bash
export TESTMO_TOKEN="testmo_api_eyJ..."
testmo projects list
```

---

## Quick Start

```bash
# 1. Configure
testmo init

# 2. Discover your projects
testmo projects list

# 3. View folder structure for a project
testmo folders list -p 1

# 4. List all test cases
testmo cases list -p 1

# 5. Pull everything to a YAML file
testmo sync pull -p 1 -f testcases.yaml

# 6. Edit the YAML file, then preview changes
testmo sync diff -p 1 -f testcases.yaml

# 7. Push changes to Testmo
testmo sync push -p 1 -f testcases.yaml
```

---

## Commands

Every command that interacts with project data requires the `-p` / `--project` flag to specify the project ID. Use `testmo projects list` to find your project ID.

### testmo init

Interactively configures the CLI by prompting for your Testmo instance URL and API token. Saves the result to `.testmo.yaml` in the current directory.

```bash
testmo init
```

```
Testmo instance URL (e.g., mycompany.testmo.net): mycompany.testmo.net
API token: testmo_api_eyJ...
Configuration saved to .testmo.yaml
```

### testmo projects list

Lists all projects in your Testmo instance with key metrics.

```bash
testmo projects list
```

```
ID  NAME            COMPLETED  RUNS  AUTOMATION RUNS
1   My Project      false      0     127
2   Another Project false      5     42
```

### testmo folders

Manage the folder (section) hierarchy that organizes test cases within a project.

#### List folders

Displays all folders as an indented tree, sorted by display order. Includes folder IDs and truncated documentation.

```bash
testmo folders list -p 1
```

```
ID  NAME                          DOCS
2   ASM Module
37    DEVRQ801 GET Domains        DEVRQ801
38    DEVRQ802 GET Subdomains     DEVRQ802
1   TAM Module
6     DEVRQ284                    GET /api/tam/v1/assets/ - Read Assets...
```

#### Create a folder

```bash
# Top-level folder
testmo folders create -p 1 --name "New Module"

# Nested folder (under parent ID 2)
testmo folders create -p 1 --name "Feature X" --parent-id 2 --docs "Tests for feature X"
```

#### Update a folder

```bash
testmo folders update -p 1 --id 10 --name "Renamed Module"
testmo folders update -p 1 --id 10 --docs "Updated description"
```

#### Delete folders

```bash
testmo folders delete -p 1 --ids 10,11,12
```

### testmo cases

Manage individual test cases within a project.

#### List cases

Lists all test cases, optionally filtered by folder. Automatically paginates through all results.

```bash
# All cases in the project
testmo cases list -p 1

# Cases in a specific folder only
testmo cases list -p 1 --folder-id 6
```

```
ID   KEY    FOLDER  NAME                                          STATE  AUTOMATION
120  C-120  6       DEVRQ284-TS001 Get assets returns 200         4      false
121  C-121  6       DEVRQ284-TS002 Get assets with pagination     4      true

Total: 464 cases
```

**Column reference:**

| Column | Description |
|--------|-------------|
| ID | Internal case ID (used for update/delete operations) |
| KEY | Display key shown in the Testmo UI (e.g., C-120) |
| FOLDER | ID of the folder containing this case |
| NAME | Test case name |
| STATE | State ID (e.g., 4 = active). See your Testmo instance for state definitions |
| AUTOMATION | Whether the case is linked to test automation |

#### Create a case

```bash
testmo cases create -p 1 --name "Verify login with valid credentials" --folder-id 6
testmo cases create -p 1 --name "Edge case test" --folder-id 6 --template-id 4 --state-id 4
```

#### Update cases

Updates one or more cases at once. All specified IDs receive the same changes.

```bash
# Rename a case
testmo cases update -p 1 --ids 120 --name "New test name"

# Move multiple cases to a different folder
testmo cases update -p 1 --ids 120,121,122 --folder-id 8

# Change state for multiple cases
testmo cases update -p 1 --ids 120,121 --state-id 1
```

#### Delete cases

```bash
testmo cases delete -p 1 --ids 120,121,122
```

### testmo sync

The sync commands provide a YAML-based workflow for managing test cases as code. This is the recommended approach for teams that want to version-control their test case definitions alongside their source code.

#### Pull: Testmo to YAML

Downloads all folders and test cases from a project and writes them to a single YAML file. The file preserves the full folder hierarchy, case names, priorities, and BDD descriptions.

```bash
testmo sync pull -p 1 -f testcases.yaml
```

```
Pulling from project 1...
Saved 93 folders and 464 cases to testcases.yaml
```

#### Diff: Preview changes (dry run)

Compares the local YAML file against the current state of the project in Testmo and shows what would change, without applying anything.

```bash
testmo sync diff -p 1 -f testcases.yaml
```

```
Computing diff for project 1...

Folders to CREATE (1):
  + New Feature Module

Cases to CREATE (3):
  + TS001 Happy path (folder: 6)
  + TS002 Error handling (folder: 6)
  + TS003 Edge case (folder: 6)

Cases to DELETE (1):
  - ID: 55

No changes detected.
```

#### Push: YAML to Testmo

Applies changes from the YAML file to Testmo. First shows a diff summary, then creates/updates/deletes as needed.

```bash
# Push changes (creates and updates only, does not delete)
testmo sync push -p 1 -f testcases.yaml

# Push changes AND delete cases/folders in Testmo that are not in the YAML
testmo sync push -p 1 -f testcases.yaml --delete
```

> **Warning:** The `--delete` flag will permanently remove test cases and folders from Testmo that are not present in your YAML file. Always run `testmo sync diff` first to preview what will be deleted.

---

## YAML Sync File Format

The sync file is a single YAML document that mirrors the full folder hierarchy of a Testmo project. Here is the complete schema:

```yaml
project: 1                          # Testmo project ID

folders:
  - name: "API Module"              # Folder name (must be unique within its parent)
    docs: "End-to-end API tests"    # Optional: folder description/documentation
    cases:                          # Optional: test cases directly in this folder
      - name: "TS001 Verify GET /users returns 200"
        priority: 2                 # Optional: priority level (integer)
        description: |              # Optional: BDD scenario or free-text description
          Scenario: List users successfully
            Given the database contains 5 users
            When the client sends GET /api/v1/users
            Then the response status code is 200
            And the response contains 5 user objects

      - name: "TS002 Unauthorized request returns 401"
        priority: 2
        description: |
          Scenario: Request without auth token
            Given no Authorization header is present
            When the client sends GET /api/v1/users
            Then the response status code is 401

    folders:                        # Optional: nested sub-folders
      - name: "DEVRQ100 User CRUD"
        docs: "Tests for user create, read, update, delete"
        cases:
          - name: "DEVRQ100-TS001 Create user"
            priority: 1
          - name: "DEVRQ100-TS002 Delete user"
            priority: 2

  - name: "Auth Module"
    cases:
      - name: "Login with valid credentials"
      - name: "Login with expired token"
```

### Key rules

- **Folder identity:** Folders are matched by their `name` within their parent. Renaming a folder in the YAML is treated as deleting the old folder and creating a new one.
- **Case identity:** Cases are matched by their `name` within their folder. The name serves as the unique key for sync operations.
- **Nesting:** Folders can be nested to arbitrary depth. Each folder can contain both `cases` and child `folders`.
- **Ordering:** Cases within a folder are sorted alphabetically by name in the YAML file after a pull. Folders are sorted by their display order in Testmo.

---

## Workflows

### Exploring Your Testmo Instance

When first connecting to a Testmo instance, use these commands to understand what is available:

```bash
# Step 1: List all projects to find the one you need
testmo projects list

# Step 2: View the folder structure for your project
testmo folders list -p 1

# Step 3: List all test cases (or filter by folder)
testmo cases list -p 1
testmo cases list -p 1 --folder-id 6
```

### Managing Folders

Folders organize test cases into logical groups (modules, features, requirements, etc.). The folder hierarchy can be as deep as needed.

```bash
# Create a top-level module folder
testmo folders create -p 1 --name "Payment Module"

# Create a sub-folder under it (use the ID from the create output)
testmo folders create -p 1 --name "DEVRQ500 Checkout Flow" --parent-id 100 \
  --docs "E2E tests for the checkout payment flow"

# Rename a folder
testmo folders update -p 1 --id 100 --name "Payment Module v2"

# Delete unused folders
testmo folders delete -p 1 --ids 100,101
```

### Managing Test Cases

Individual test cases can be created, updated, and deleted directly from the command line.

```bash
# Create a test case in a specific folder
testmo cases create -p 1 --name "DEVRQ500-TS001 Checkout with valid card" --folder-id 100

# Update a test case name
testmo cases update -p 1 --ids 250 --name "DEVRQ500-TS001 Checkout with valid credit card"

# Move cases to a different folder
testmo cases update -p 1 --ids 250,251,252 --folder-id 105

# Delete test cases
testmo cases delete -p 1 --ids 250,251
```

### YAML Sync Workflow

The sync workflow is the most powerful feature. It enables a **test-cases-as-code** approach where your test case definitions live in a YAML file under version control, and you sync them to Testmo as needed.

#### Initial Setup: Pull from Testmo

Start by pulling the current state from Testmo into a YAML file:

```bash
testmo sync pull -p 1 -f testcases.yaml
```

This creates a `testcases.yaml` file containing every folder and test case in the project. Commit this file to your repository.

#### Day-to-Day: Edit and Push

When you need to add, rename, or reorganize test cases:

1. **Edit the YAML file** - add new cases, rename existing ones, create new folders, etc.

2. **Preview changes** - always review what will change before pushing:
   ```bash
   testmo sync diff -p 1 -f testcases.yaml
   ```

3. **Push changes** - apply the diff to Testmo:
   ```bash
   testmo sync push -p 1 -f testcases.yaml
   ```

4. **Commit the YAML file** to version control so the team has a shared source of truth.

#### Handling Deletions

By default, `sync push` only creates and updates. It will **not** delete cases or folders from Testmo that are missing from the YAML file. This is a safety measure.

To also delete orphaned items, use the `--delete` flag:

```bash
# Preview deletions first
testmo sync diff -p 1 -f testcases.yaml

# Then push with deletions enabled
testmo sync push -p 1 -f testcases.yaml --delete
```

#### Staying in Sync

If test cases are modified directly in the Testmo UI (by testers, for example), re-pull periodically to capture those changes:

```bash
testmo sync pull -p 1 -f testcases.yaml
```

This overwrites the local YAML with the current Testmo state. Use `git diff` to review what changed, then commit.

### CI/CD Integration

The CLI works well in CI/CD pipelines. Use environment variables for authentication:

```yaml
# Example: GitHub Actions
steps:
  - name: Sync test cases to Testmo
    env:
      TESTMO_URL: ${{ secrets.TESTMO_URL }}
      TESTMO_TOKEN: ${{ secrets.TESTMO_TOKEN }}
    run: |
      testmo sync diff -p 1 -f testcases.yaml
      testmo sync push -p 1 -f testcases.yaml
```

```yaml
# Example: GitLab CI
sync-testmo:
  script:
    - testmo sync push -p $TESTMO_PROJECT_ID -f testcases.yaml
  variables:
    TESTMO_URL: $TESTMO_URL
    TESTMO_TOKEN: $TESTMO_TOKEN
```

---

## API Details

The CLI communicates with the [Testmo REST API v1](https://docs.testmo.com/api).

### Authentication

All requests include the header `Authorization: Bearer <token>`.

### Pagination

List endpoints return up to 100 items per page. The CLI automatically fetches all pages, so you always get the complete result set regardless of how many items exist.

### Bulk Operations

Create, update, and delete operations are batched in groups of up to 100 items per API request. This is handled transparently -- you can create or delete hundreds of cases in a single CLI call.

### Rate Limiting

If the API returns HTTP 429 (rate limited), the CLI waits for the duration specified in the `Retry-After` header (default: 60 seconds) and retries automatically, up to 3 attempts.

### Error Handling

API errors are reported with the HTTP status code and the full error response body from Testmo, making it straightforward to diagnose issues.

---

## Project Structure

```
testmo-cli/
├── main.go                      # Entry point
├── go.mod                       # Go module definition
├── cmd/
│   ├── root.go                  # Root command, config loading
│   ├── init.go                  # `testmo init` - interactive setup
│   ├── projects.go              # `testmo projects list`
│   ├── folders.go               # `testmo folders list/create/update/delete`
│   ├── cases.go                 # `testmo cases list/create/update/delete`
│   └── sync.go                  # `testmo sync pull/push/diff`
├── internal/
│   ├── api/
│   │   ├── client.go            # HTTP client (auth, pagination, rate limiting)
│   │   ├── projects.go          # Project API types and methods
│   │   ├── folders.go           # Folder API types and methods (bulk CRUD)
│   │   └── cases.go             # Case API types and methods (bulk CRUD)
│   ├── config/
│   │   └── config.go            # Config loading (env vars + .testmo.yaml)
│   └── sync/
│       └── sync.go              # YAML parsing, diff computation, apply logic
└── README.md
```

### Key design decisions

- **Go + Cobra**: Standard Go CLI stack. Produces a single static binary with no runtime dependencies.
- **No external HTTP libraries**: Uses the Go standard library `net/http` for API calls, keeping the dependency footprint minimal.
- **Automatic pagination**: All list operations fetch every page transparently, so consumers never need to think about pagination.
- **Bulk batching**: Creates, updates, and deletes are automatically batched in groups of 100 (the Testmo API maximum), minimizing the number of API calls.
- **Name-based matching**: The sync engine matches folders by `(parent, name)` and cases by `(folder, name)`. This means renaming is treated as a delete + create, which is the safest approach for idempotent syncs.

---

## License

MIT
