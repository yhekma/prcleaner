# PRCLEANER

Service that you can push json to using github webhooks to delete helm releases

## Usage

Program uses the following env variables:
* `CLEANER_RELEASELABEL` label to match releases on, default `helm.sh/release`
* `CLEANER_BRANCHLABEL` label to match branches on, default `app.org.io/git-branch`
* `CLEANER_OWNERLABEL` label to match owner on, default `app.org.io/git-owner`
* `CLEANER_REPOLABEL` label to match repository on, default `app.org.io/git-repository`
* `CLEANER_SECRET` secret for the webhook to be used
* `CLEANER_DRYRUN` self-explanatory
* `CLEANER_DEBUG` self-explanatory
* `CLEANER_DELAY` rerun the same query to find deployments to be deleted after this many seconds, default `300`

## Flow

Webhook that acts when PR is opened, PR is closed or when branch is deleted.

### PR closed

All pods with correspoding SHA, PR label and repo label will have their corresponding helm release deleted

### PR Created/branch deleted

All pods with correspoding SHA, branch label and repo label will have their corresponding helm release deleted
