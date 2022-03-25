package main

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Arriven/db1000n/src/mobilelogger"
	"github.com/alecthomas/kingpin"
	"github.com/valyala/fasthttp"
)

var serverPort = kingpin.Flag("port", "port to use for benchmarks").
	Default("8080").
	Short('p').
	String()
var responseSize = kingpin.Flag("size", "size of response in bytes").
	Default("1024").
	Short('s').
	Uint()

func main() {
	var requests uint64
	start := time.Now()
	kingpin.Parse()
	response := strings.Repeat("a", int(*responseSize))
	addr := "localhost:" + *serverPort
	mobilelogger.Infof("Starting HTTP server on:", addr)
	go func() {
		for {
			time.Sleep(time.Second)
			fmt.Println(time.Since(start), "requests handled", atomic.LoadUint64(&requests))
		}
	}()
	err := fasthttp.ListenAndServe(addr, func(c *fasthttp.RequestCtx) {
		defer atomic.AddUint64(&requests, 1)
		_, werr := c.WriteString(response)
		if werr != nil {
			mobilelogger.Infof(werr)
		}
	})
	if err != nil {
		mobilelogger.Infof(err)
	}
}
