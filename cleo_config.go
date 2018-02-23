package main

import "os"
import "time"
import "net/http"

var Key []byte = []byte("a very very very very secret key")

// abbreviation of DeFault Directory.
var dfd string = os.ExpandEnv("$GOPATH")

var cleoWorkspace string = "cleo_workspace"

var serverWaitTime int = 20

var configPath string

//default timeout of test request
var dft = 45 * time.Second

const Utilname = "cleo_util.go"

const HeapUrl = "/debug/pprof/heap"

const ProfileUrl = "/debug/pprof/profile"

const DebugParam = "?debug=1"

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
