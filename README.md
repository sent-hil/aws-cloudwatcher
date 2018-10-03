# aws-cloudwatcher

[![Documentation](https://godoc.org/github.com/sent-hil/aws-cloudwatcher?status.svg)](https://godoc.org/github.com/sent-hil/aws-cloudwatcher)

## Install

    $ go get -u github.com/sent-hil/aws-cloudwatcher

## Getting Started

* Setup AWS credentials, ie `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY` `$AWS_REGION` env variables should all be non empty strings.
* Give cloudwatch IAM access to the credential above.

Stream a particular log group and their streams:

    aws-cloudwatcher -log='/aws/elasticbeanstalk/app-staging/var/log/awslogs.log'

Stream log groups that match keyword and their streams:

    aws-cloudwatcher -log='app-staging'

Stream log groups that match keyword and their streams starting after given time:

    aws-cloudwatcher -log='app-staging' -start='1 min ago'

Stream log groups that match keyword for particular stream:

    aws-cloudwatcher -log='app-staging' -start='1 day ago' -stream='i-xxxx'

See https://github.com/wanasit/chrono for more docs on what arguments `-start` accepts.

## TODO:

* Check for new log groups and their streams after command is started.
