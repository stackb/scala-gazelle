package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/label"
)
