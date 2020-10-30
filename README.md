# PRCLEANER

Service that you can push json to using github webhooks to delete helm releases

## Usage

```
  -a, --app string            app to process for
      --branchLabel string    label name for branches (default "app.fedex.io/git-branch")
  -d, --dry-run               don't actually do anything (default true)
  -o, --org string            org to process for
      --releaseLabel string   label name for releases (default "helm.sh/release")
      --repoLabel string      label name for repo (default "app.fedex.io/git-repository")
  -v, --verbose               turn on verbose
```

## Flow

When a close action comes in, all pods in namespaces `<app>` and `<app>-(.*)` will be considered. If the *branchlabel* is `PR-<prnumber>`, the org of the PR matches the org passed on the command line and the *repoLabel* matches the repositoryname,
helm gets executed, deleting the release as defined in *releaseLabel*