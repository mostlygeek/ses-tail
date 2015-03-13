# SES Tail

The `ses-tail` program provides `tail` like output for SES mail notifications. 

Amazon's Simple Email Service (SES) is a simple way to send email but handling delivery, bounce or complaint notifications requires setting up SNS and a notification endpoint. 

`ses-tail` expects a SES => SNS => SQS architecture. It will receive, print and delete SES notifications from an SQS queue producing output suitable for piping to `grep`, `jq`, etc.

## Requirements

* go v1.4+
* git
* godep

## Installation

````
> go get github.com/tools/godep
> go get github.com/mostlygeek/ses-tail
> cd $GOPATH/src/github.com/mostlygeek/ses-tail
> godep go install
````

Now `$GOPATH/bin/ses-tail` should be available to use

## Usage

````
> ses-tail -queue https://sqs.<region>.amazonaws.com/<accountid>/<sqs-name>
````

This is the simpliest usage. Only the SQS queue URL is required. Additional flags are avaiable for the AWS access id, access secret, etc. 

````
> ses-tail -h
Usage of ses-tail:
  -access_id="": (optional) AWS Access ID, auto-detected if blank
  -format="maillog": (optional) output format, maillog|json
  -nopurge=false: leave messages in queue after receiving them
  -q="": (required) SQS queue URL -- shorthand
  -queue="": (required) SQS queue URL
  -region="us-west-2": (optional) AWS region
  -secret_id="": (optional) AWS Access ID, auto-detected if blank
````

## License

See LICENSE.txt


