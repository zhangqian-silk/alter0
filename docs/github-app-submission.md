# GitHub App Verified Submission

`alter0` 在受保护分支上要求提交满足 `Commits must have verified signatures`。

为保证机器人提交可验证，远端提交流程统一采用 GitHub App installation token + GitHub API，禁止本地 `git commit + git push` 直接推送。

## 1. 前置条件

1. 已有 GitHub App installation token（建议通过环境变量 `GITHUB_TOKEN` 注入）。
2. Token 具备目标仓库 `contents:write`、`pull_requests:write` 权限。
3. `gh auth status` 显示当前使用的是 App token 对应身份。

## 2. 标准流程

### Step A - 获取基线提交 SHA

```bash
OWNER="zhangqian-silk"
REPO="alter0"
BASE_BRANCH="master"
WORK_BRANCH="docs/update-upcoming-requirements"

BASE_SHA=$(gh api repos/${OWNER}/${REPO}/git/ref/heads/${BASE_BRANCH} --jq '.object.sha')
echo "base sha: ${BASE_SHA}"
```

### Step B - 创建工作分支（若不存在）

```bash
gh api repos/${OWNER}/${REPO}/git/refs \
  -X POST \
  -f ref="refs/heads/${WORK_BRANCH}" \
  -f sha="${BASE_SHA}"
```

### Step C - 通过 Contents API 提交文件（示例：单文件）

> 说明：Contents API 默认每次提交一个文件。多文件可连续调用，或使用 GraphQL `createCommitOnBranch` 做单提交聚合。

```bash
FILE_PATH="docs/requirements.md"
COMMIT_MSG="docs: refresh requirements"

# 若文件存在，取当前 blob sha（新文件可不传）
FILE_SHA=$(gh api repos/${OWNER}/${REPO}/contents/${FILE_PATH}?ref=${WORK_BRANCH} --jq '.sha' 2>/dev/null || true)

CONTENT_B64=$(base64 -w0 "${FILE_PATH}")

if [ -n "${FILE_SHA}" ]; then
  gh api repos/${OWNER}/${REPO}/contents/${FILE_PATH} \
    -X PUT \
    -f message="${COMMIT_MSG}" \
    -f content="${CONTENT_B64}" \
    -f sha="${FILE_SHA}" \
    -f branch="${WORK_BRANCH}"
else
  gh api repos/${OWNER}/${REPO}/contents/${FILE_PATH} \
    -X PUT \
    -f message="${COMMIT_MSG}" \
    -f content="${CONTENT_B64}" \
    -f branch="${WORK_BRANCH}"
fi
```

### Step D - 创建或更新 PR

```bash
gh pr create \
  --repo ${OWNER}/${REPO} \
  --base ${BASE_BRANCH} \
  --head ${WORK_BRANCH} \
  --title "docs: update requirements" \
  --body "docs-only changes"
```

## 3. 验证提交签名状态

```bash
HEAD_SHA=$(gh api repos/${OWNER}/${REPO}/git/ref/heads/${WORK_BRANCH} --jq '.object.sha')
gh api repos/${OWNER}/${REPO}/commits/${HEAD_SHA} --jq '.commit.verification'
```

结果要求：

- `.verified == true`
- `.reason` 为可验证状态（非 `unsigned`）

## 4. 强约束

1. API 提交请求中不传自定义 `author` / `committer` / `signature`。
2. 若出现未验证提交，视为流程失败，必须重走 API 提交流程。
3. 不允许退回本地 `git commit + git push` 绕过策略。
