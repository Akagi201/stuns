package main

import (
	"runtime"
	"strings"
	"time"

	"github.com/Akagi201/utilgo/conflag"
	flags "github.com/jessevdk/go-flags"
	log "github.com/sirupsen/logrus"
)

var opts struct {
	Conf      string `long:"conf" description:"stuns config file"`
	Transport string `long:"transport" default:"udp" description:"transport protocol"`
	Addr      string `long:"addr" default:"0.0.0.0:3478" description:"address to listen"`
	Profile   bool   `long:"profile" description:"whether profile or not"`
	LogLevel  string `long:"log_level" default:"info" description:"log level"`
}

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func init() {
	parser := flags.NewParser(&opts, flags.Default|flags.IgnoreUnknown)

	parser.Parse()

	if opts.Conf != "" {
		conflag.LongHyphen = true
		conflag.BoolValue = false
		args, err := conflag.ArgsFrom(opts.Conf)
		if err != nil {
			panic(err)
		}

		parser.ParseArgs(args)
	}

	log.Infof("stuns opts: %+v", opts)
}

func init() {
	if level, err := log.ParseLevel(strings.ToLower(opts.LogLevel)); err != nil {
		log.SetLevel(level)
	}

	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: time.RFC3339,
	})
}
