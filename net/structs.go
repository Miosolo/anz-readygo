package net

import (
	"os"
	"strings"
)

// Space defines the space as a Go struct
type Space struct { // specified checkpoint, upper-layer
	Name string  `json:"name" description:"global unique name of the space"`
	Base string  `json:"base" description:" the parent space it lies in" default:"base"`
	Rx   float64 `json:"rx" description:"relative x axis value of the parent space"`
	Ry   float64 `json:"ry" description:"relative y axis value of the parent space"`
}

// Asset defines the asset belonging to a space as a Go struct
type Asset struct { // specified checkpoint, upper-layer
	Name   string  `json:"name" description:"unique name in its base space"`
	Base   string  `json:"base" description:"the base space it lies in" default:"base"`
	Rx     float64 `json:"rx" description:"relative x axis value of the parent space"`
	Ry     float64 `json:"ry" description:"relative y axis value of the parent space"`
	Weight float64 `json:"weight" description:"global weight in sampling" default:"1.0"`
}

const (
	//WEEK_SECONDS meas the seconds of a week
	WEEK_SECONDS = 604800 // 7 * 24 * 3600
	//MONTH_SECONDS meas the seconds of a month
	MONTH_SECONDS = 259200 // 30 * 24 * 3600
)

//BakCtx is the default config in production env
var BakCtx = RestContext{ // hard-coded backup context
	KeyPath:     "/root/readygo_keys/api.readygo.miosolo.top.key",
	CrtPath:     "/root/readygo_keys/api.readygo.miosolo.top.crt",
	MongoURI:    "mongodb://readygo-test:readygo2019@miosolo.top:8017",
	MongoDBName: "readygo",
	RedisURL:    "miosolo.top:8079",
	RedisPass:   "readygo2019",
}

//RCTest is the defult test config of test env
var RCTest = RestContext{ // hard-coded backup context
	KeyPath:     strings.Join([]string{os.Getenv("GOPATH"), "src", "github.com", "miosolo", "readygo", "test", "test.key"}, string(os.PathSeparator)),
	CrtPath:     strings.Join([]string{os.Getenv("GOPATH"), "src", "github.com", "miosolo", "readygo", "test", "test.crt"}, string(os.PathSeparator)),
	MongoURI:    "mongodb://readygo-test:readygo2019@miosolo.top:8017",
	MongoDBName: "readygo",
	RedisURL:    "miosolo.top:8079",
	RedisPass:   "readygo2019",
}
