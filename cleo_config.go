package main

import "os"
import "time"
import "net/http"
import "strings"
import "runtime"

// Timeout of web client request
// Used when fetching heap information
// from go web server.
const dft = 45 * time.Second

// name of file appended to
// project.
const Utilname = "cleo_util.go"

const HeapUrl = "/debug/pprof/heap"

const ProfileUrl = "/debug/pprof/profile"

const DebugParam = "?debug=1"

const CPUTOPSamples int = 100

// Bash script used to start
// and test local go web server.
const BuildScript string = `#!/bin/bash  
cmd="%s"
startServer="%s-cleo"
eval "${startServer}" &>%s.log &disown
sleep %v
eval "${cmd}" >%s.test &disown
exit 0`

// Batch script used to start
// and test local go web server.
const BatchBuildScript string = "START \"\" %s-cleo 1>%s.log\nTIMEOUT %v\nSTART \"\" %s\\%s 1>%s.test"

// Batch script used to launch
// test to external go web server.
const BatchLaunchScript = "START \"\" %s\\%s 1>%s.test"

// Bash script used to launch
// test to external go web server.
const LaunchScript = `#!/bin/bash  
cmd="%s"
eval "${cmd}" >%s.test &disown
exit 0`

const HostAddress = "http://127.0.0.1"

// key used to encrypt cleo config.
// Must be AES-16 or AES-32
var Key []byte = []byte("a very very very very secret key")

// abbreviation of DeFault Directory.
// Update with the path to your go src.
// If $GOPATH is set leave as is.
var dfd string = os.ExpandEnv("$GOPATH")

var cleoWorkspace string = "cleo_workspace"

// Wait time before cleo
// begins sending test requests
var serverWaitTime int = 20

var configPath string

//default timeout of test request

//file appended to project.
// To enable pprof debug urls
var CleoUtil = []byte(`package main


import (
	_ "net/http/pprof"
)`)

var tr *http.Transport = &http.Transport{
	MaxIdleConns:          2,
	IdleConnTimeout:       dft,
	ResponseHeaderTimeout: dft,
	ExpectContinueTimeout: dft,
}

var isWindows bool = strings.Contains(runtime.GOOS, "indows")
