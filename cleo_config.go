package main

import "os"

var Key []byte = []byte("a very very very very secret key")

// abbreviation of DeFault Directory.
var dfd string = os.ExpandEnv("$GOPATH")

var cleoWorkspace string = "cleo_workspace"

var configPath string
