**Deprecated; this tool is no longer maintained.**

# heroku-pg-s3-backup-tool

This is a tool for backing up a PostgreSQL database running as a Heroku Add-on
to an S3 bucket.

## Usage

### Mandatory environment variables

* `HEROKU_APP_NAME` (ex. `appname`)
* `PGBACKUP_HEROKU_API_TOKEN` (ex. `00000000-0000-0000-0000-000000000000`)
* `PGBACKUP_AWS_ACCESS_KEY_ID` (ex. `AKIAXXXXXXXXXXXXXXXX`)
* `PGBACKUP_AWS_SECRET_ACCESS_KEY` (ex. `xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx`)
* `PGBACKUP_AWS_REGION` (ex. `eu-central-1`)
* `PGBACKUP_S3_BUCKET_NAME` (ex. `backup-bucket`)
