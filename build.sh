#!/bin/bash

GOPATH=$(pwd)/Godeps/_workspace godep restore
docker build -t pipeline .
