package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/gen/sqs"
	. "github.com/tj/go-debug"
)

var (
	// general debug output
	_d = Debug("")

	// flags
	fAccessId     string
	fAccessSecret string
	fRegion       string
	fQueueUrl     string
	fNoPurge      bool
)

func init() {

	flag.StringVar(&fQueueUrl, "queue", "", "(required) SQS queue URL")
	flag.StringVar(&fQueueUrl, "q", "", "(required) SQS queue URL -- shorthand")
	flag.StringVar(&fAccessId, "access_id", "", "(optional) AWS Access ID, auto-detected if blank")
	flag.StringVar(&fAccessSecret, "secret_id", "", "(optional) AWS Access ID, auto-detected if blank")
	flag.StringVar(&fRegion, "region", "us-west-2", "(optional) AWS region")
	flag.BoolVar(&fNoPurge, "nopurge", false, "leave messages in queue after receiving them")

	flag.Parse()

	if fQueueUrl == "" {
		fmt.Fprintf(os.Stderr, "SQS queue url required\n")
		os.Exit(1)
	}

}

func main() {
	creds := aws.DetectCreds(fAccessId, fAccessSecret, "")
	conn := sqs.New(creds, fRegion, nil)

	_dbgDelete := Debug("SQS:BatchDelete")
	_dbgReceive := Debug("SQS:Receive")

	req := &sqs.ReceiveMessageRequest{
		MaxNumberOfMessages: aws.Integer(10),
		QueueURL:            aws.String(fQueueUrl),
		VisibilityTimeout:   aws.Integer(5),
		WaitTimeSeconds:     aws.Integer(20),
	}

	for {
		resp, err := conn.ReceiveMessage(req)

		if err != nil {
			_dbgReceive(err.Error())
			continue
		}

		num := len(resp.Messages)

		if num == 0 {
			_dbgReceive("Long poll timeout, No messages received")
			continue
		}

		entries := make([]sqs.DeleteMessageBatchRequestEntry, num, num)
		for i, m := range resp.Messages {

			// json extract it all
			var notify Notification

			err := json.Unmarshal([]byte(*m.Body), &notify)
			if err != nil {
				_d("JSON unmarshal failed %s", err.Error())
				continue
			}

			Maillog(&notify)
			//fmt.Println(*m.Body)

			// save them for deleting
			if fNoPurge == false {
				entries[i] = sqs.DeleteMessageBatchRequestEntry{ID: m.MessageID, ReceiptHandle: m.ReceiptHandle}
			}
		}

		if fNoPurge == false {
			delResp, delErr := conn.DeleteMessageBatch(&sqs.DeleteMessageBatchRequest{Entries: entries, QueueURL: aws.String(fQueueUrl)})
			if delErr != nil {
				_dbgDelete("ERR %s", err)
			} else {
				_dbgDelete("Batch delete Success:%d, Failed:%d", len(delResp.Successful), len(delResp.Failed))
			}
		} else {
			_dbgDelete("Skip, purge disabled")
		}
	}

}

type Notification struct {
	Type             string
	MessageId        string
	TopicArn         string
	Timestamp        time.Time
	Message          string
	Signature        string
	SignatureVersion string
	SigningCertURL   string
	UnsubscribeURL   string
}

func (n *Notification) SESNotification() (*SESNotification, error) {
	var s SESNotification
	err := json.Unmarshal([]byte(n.Message), &s)
	return &s, err
}

type SESNotification struct {
	Type      string    `json:"notificationType"`
	Mail      Mail      `json:"mail"`
	Bounce    Bounce    `json:"bounce"`
	Complaint Complaint `json:"complaint"`
	Delivery  Delivery  `json:"delivery"`
}

type Mail struct {
	ID          string    `json:"messageId"`
	Timestamp   time.Time `json:"timestamp"`
	Source      string    `json:"source"`
	Destination []string  `json:"destination"`
}

type Bounce struct {
	ID                string            `json:"feedbackId"`
	Timestamp         time.Time         `json:"timestamp"`
	Type              string            `json:"bounceType"`
	SubType           string            `json:"bounceSubType"`
	BouncedRecipients []BounceRecipient `json:"bouncedRecipients"`
}

type BounceRecipient struct {
	Email          string `json:"emailAddress"`
	Action         string `json:"action"`
	Status         string `json:"status"`
	DiagnosticCode string `json:"diagnosticCode"`
}

type Complaint struct {
	ID          string               `json:"feedbackId"`
	Timestamp   time.Time            `json:"timestamp"`
	Recipients  []ComplaintRecipient `json:"complainedRecipients"`
	UserAgent   string               `json:"userAgent"`
	Type        string               `json:"complaintFeedbackType"`
	arrivalDate string               `json:"arrivalDate"`
}

type ComplaintRecipient struct {
	Email string `json:"emailAddress"`
}

type Delivery struct {
	Timestamp    time.Time `json:"timestamp"`
	Delay        int       `json:"processingTimeMillis"`
	Recipients   []string  `json:"recipients"`
	SmtpResponse string    `json:"smtpResponse"`
	ReportingMTA string    `json:"reportingMTA"`
}

// Maillog formats output in a maillog'ish log format
func Maillog(n *Notification) {
	s, e := n.SESNotification()

	if e != nil {
		_d("Error: %s", e.Error())
		return
	}

	switch s.Type {
	case "Delivery":
		fmt.Printf("%s DELIVERY: %s: to=%v, delay=%d, dsn=%s\n",
			s.Delivery.Timestamp.Format(time.Stamp),
			n.MessageId,
			s.Delivery.Recipients,
			s.Delivery.Delay,
			s.Delivery.SmtpResponse,
		)
	case "Bounce":
		for _, r := range s.Bounce.BouncedRecipients {

			fmt.Printf("%s BOUNCE: %s: %s, to=%v, dsn=%s\n",
				s.Bounce.Timestamp.Format(time.Stamp),
				n.MessageId,
				s.Bounce.Type,
				r.Email,
				strings.Replace(r.DiagnosticCode, "\n", " ", -1),
			)
		}

	case "Complaint":
		fmt.Printf("%v\n", s.Complaint)
	default:
		_d("Unknown type %s", s.Type)
	}
}
