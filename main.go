package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
	"github.com/miosolo/readygo/net"
)

func main() {
	demo := flag.Bool("demo", false, "web demo mode will use the self-signed certficates and load test data")
	sample := flag.Bool("sample", false, "sample mode will load the test data")
	key := flag.String("key", "", "server private key file")
	crt := flag.String("crt", "", "server certificate file")
	mongoURI := flag.String("mongouri", "", "mongoDB server URI")
	mongoDB := flag.String("mongodb", "", "mongoDB Database name")
	redisURL := flag.String("redisurl", "", "Redis server URL")
	redisPass := flag.String("redispass", "", "Redis auth password")

	var r net.RestContext
	c := net.CheckResource{
		Assets: map[string]net.Asset{},
		Spaces: map[string]net.Space{}}

	defer func() {
		if p := recover(); p != nil {
			log.Printf("the server encountered a fatal error and exit, %v\n", p)
		}
	}()

	flag.Parse()
	if *demo || *sample {
		if *demo {
			r = net.RCTest
		} else {
			r = net.BakCtx
		}

		if err := r.InitEnv(); err != nil {
			log.Panicln("cannot init the Restful web server")
		}
		if err := r.LoadDemoData(); err != nil {
			log.Panicln("failed to load demo data to database")
		}

		intChan := make(chan os.Signal, 1) // handles Ctrl-C to cleanup
		signal.Notify(intChan, os.Interrupt)
		go func() {
			<-intChan
			r.UnloadDemoData()
			os.Exit(1)
		}()
	} else {
		r = net.BakCtx
		if err := r.InitEnv(); err != nil {
			log.Panicln("cannot init the Restful web server")
		}
	}

	if *key != "" {
		r.KeyPath = *key
	}
	if *crt != "" {
		r.CrtPath = *crt
	}
	if *mongoURI != "" {
		r.MongoURI = *mongoURI
	}
	if *mongoDB != "" {
		r.MongoDBName = *mongoDB
	}
	if *redisURL != "" {
		r.RedisURL = *redisURL
	}
	if *redisPass != "" {
		r.RedisPass = *redisPass
	}

	restful.DefaultContainer.Add(c.WebService(&r))
	config := restfulspec.Config{
		WebServices:                   restful.RegisteredWebServices(), // you control what services are visible
		APIPath:                       "v1/apidocs.json",
		PostBuildSwaggerObjectHandler: net.EnrichSwaggerObject,
		DisableCORS:                   false}
	restful.DefaultContainer.Add(restfulspec.NewOpenAPIService(config))

	http.Handle("/apidocs/", http.StripPrefix("/apidocs/", http.FileServer(http.Dir("swagger-ui/dist"))))
	log.Printf("start listening on :8043...")
	server := &http.Server{Addr: ":8043", Handler: restful.DefaultContainer}
	log.Panicln(server.ListenAndServeTLS(r.CrtPath, r.KeyPath))
}
