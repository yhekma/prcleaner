# PRCLEANER

Service that you can push json to using github webhooks to delete helm releases

## Usage

```
      --branchLabel string    label name for branches (default "app.fedex.io/git-branch")
      --commitSha string      label name for commit sha (default "app.fedex.io/git-commit")
  -d, --dry-run               don't actually do anything (default true)
      --releaseLabel string   label name for releases (default "helm.sh/release")
      --repoLabel string      label name for repo (default "app.fedex.io/git-repository")
  -v, --verbose               turn on verbose
```

## Flow

Webhook that acts when PR is opened, PR is closed or when branch is deleted.

### PR closed

All pods with correspoding SHA, PR label and repo label will have their corresponding helm release deleted

### PR Created/branch deleted

All pods running with branch label of the PR will have their corresponding helm release deleted

