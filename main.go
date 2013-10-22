package main

import (
	"flag"
	"fmt"
	"github.com/Mistobaan/sqs"
	"io"
	"launchpad.net/goamz/aws"
	"log"
	"os"
	"os/exec"
)

func openQueue(auth aws.Auth, region aws.Region, name string) (queue *sqs.Queue, err error) {
	awsSQS := sqs.New(auth, aws.APNortheast)
	queue, err = awsSQS.GetQueue(name)
	return
}

func dispatchMessages(queue *sqs.Queue, messages []sqs.Message, bodyHandler func(string) error) []error {
	// Prepare error list.
	errorList := make([]error, len(messages))
	errorCount := 0

	// Prepare for delete messages.
	deleteList := make([]sqs.Message, len(messages))
	deleteCount := 0
	defer func() {
		if deleteCount > 0 {
			go func() {
				resp, err := queue.DeleteMessageBatch(deleteList[0:deleteCount])
				if err != nil {
					log.Println("failed to delele messages", err, resp)
					recover()
				}
			}()
		}
	}()

	// Dispatch all messages.
	for _, m := range messages {
		err2 := bodyHandler(m.Body)
		if err2 != nil {
			errorList[errorCount] = err2
			errorCount += 1
			continue
		}
		deleteList[deleteCount] = m
		deleteCount += 1
	}

	return errorList[0:errorCount]
}

func listenQueue(queue *sqs.Queue, bodyHandler func(string) error) (err error) {
	for {
		resp, err := queue.ReceiveMessage(10)
		if err != nil {
			return err
		}

		errs := dispatchMessages(queue, resp.Messages, bodyHandler)
		if errs != nil && len(errs) > 0 {
			return errs[0]
		}
	}
}

func usage() {
	fmt.Printf(`Usage: %s {queue name} {command}

Environment variables:
  AWS_ACCESS_KEY_ID
  AWS_SECRET_ACCESS_KEY
`, os.Args[0])
	os.Exit(1)
}

func runCmd(cmd *exec.Cmd, msgbody string) (err error) {
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return
	}
	err = cmd.Start()
	if err != nil {
		return
	}
	go io.Copy(os.Stdout, stdout)
	go io.Copy(os.Stderr, stderr)
	go func() {
		stdin.Write([]byte(msgbody))
		stdin.Close()
	}()
	return cmd.Wait()
}

func main() {
	flag.Parse()
	args := flag.Args()

	if len(args) < 2 {
		usage()
	}
	name := args[0]

	auth, err := aws.EnvAuth()
	if err != nil {
		log.Fatalln("sqs-notify:", err)
	}

	queue, err := openQueue(auth, aws.APNortheast, name)
	if err != nil {
		log.Fatalln("sqs-notify:", err)
	}

	listenQueue(queue, func(body string) (error) {
		cmd := exec.Command(args[1], args[2:]...)
		return runCmd(cmd, body)
	})
}