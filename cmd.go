package main

import (
	b64 "encoding/base64"
	"log"
	"os/exec"
)

type CommandResponse struct {
	Status int
	Output string
}

func New(cmdString string) (CommandResponse, error) {
	cmd := exec.Command("/bin/sh", "-c", cmdString)
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatal(err)
		return CommandResponse{Status: 1, Output: ""}, err
	}
	return CommandResponse{Status: 0, Output: b64.StdEncoding.EncodeToString(stdoutStderr)}, nil
}

func Worker(job Job, output chan Job) {
	cmdOutput, err := New(job.CMDString)
	if err != nil {
		return
	}
	job.Output = cmdOutput.Output
	job.Status = cmdOutput.Status
	output <- job
}
