# PRCLEANER

Service that you can push json to using github webhooks to delete helm releases

## Usage

Program uses the following env variables:
* `CLEANER_RELEASELABEL` label to match releases on, default `helm.sh/release`
* `CLEANER_BRANCHLABEL` label to match branches on
* `CLEANER_COMMITSHALABEL` label to match sha on
* `CLEANER_REPOLABEL` label to match reposutory on
* `CLEANER_SECRET` secret for the webhook to be used
* `CLEANER_DRYRUN` self-explanatory
* `CLEANER_DEBUG` self-explanatory

## Flow

Webhook that acts when PR is opened, PR is closed or when branch is deleted.

### PR closed

All pods with correspoding SHA, PR label and repo label will have their corresponding helm release deleted

### PR Created/branch deleted

All pods with correspoding SHA, branch label and repo label will have their corresponding helm release deleted
