package net

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/miosolo/readygo/route"

	"github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
	"github.com/go-openapi/spec"
	"github.com/gomodule/redigo/redis"
	dataio "github.com/miosolo/readygo/io"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"log"
	"os"
	"strings"
)

// RestContext : Restful API context on system
type RestContext struct {
	KeyPath       string
	CrtPath       string // HTTPS crt and key, signed to api.readygo.miosolo.top
	MongoURI      string // URI to connect mongoDB
	mongoDB       *mongo.Database
	MongoDBName   string
	RedisURL      string // URL of Redis Server
	RedisPass     string
	redisConnPool *redis.Pool
}

// InitEnv : check and try to correct the RestContext and connet to DB servers
func (r *RestContext) InitEnv() (err error) {

	defer func() { // global recover
		if err != nil { // using clousure to detect err's return value
			log.Println("all tries failed")
			debug.PrintStack()
		}
	}()

	if _, err = os.Stat(r.CrtPath); err != nil {
		log.Println(err)
		log.Println("crt file not exist, trying BakCtx crt file")
		r.CrtPath = BakCtx.CrtPath
		if _, err := os.Stat(r.CrtPath); err != nil {
			log.Println(err)
			log.Println("crt file not exist, trying RCTest crt file")
			r.CrtPath = RCTest.CrtPath
			if _, err := os.Stat(r.CrtPath); err != nil {
				log.Println(err)
				log.Println("crt file not exist again")
				return err
			}
		}
	}

	if _, err = os.Stat(r.KeyPath); err != nil {
		log.Println(err)
		log.Println("key file not exist, trying BakCtx key file")
		r.KeyPath = BakCtx.KeyPath
		if _, err := os.Stat(r.KeyPath); err != nil {
			log.Println(err)
			log.Println("key file not exist, trying RCTest key file")
			r.KeyPath = RCTest.KeyPath
			if _, err := os.Stat(r.KeyPath); err != nil {
				log.Println(err)
				log.Println("key file not exist again")
				return err
			}
		}
	}

	if r.MongoDBName == "" {
		r.MongoDBName = BakCtx.MongoDBName
	}
	if err = r.connectMongoDB(); err != nil {
		log.Println("connet to mongodb failed, trying backup URI")
		r.MongoURI = BakCtx.MongoURI
		if err = r.connectMongoDB(); err != nil {
			log.Println("connet to mongodb failed again")
			return err
		}
	}

	if r.RedisURL == "" || r.RedisPass == "" {
		r.RedisURL, r.RedisPass = BakCtx.RedisURL, BakCtx.RedisPass
	}
	r.makeRedisPool()
	tmpCon := r.redisConnPool.Get()
	defer tmpCon.Close()
	if _, err = tmpCon.Do("ping"); err != nil {
		log.Println("connect to Redis failed, trying backup URL and pass")
		r.RedisURL, r.RedisPass = BakCtx.RedisURL, BakCtx.RedisPass
		r.makeRedisPool()
		tmpCon = r.redisConnPool.Get()
		if _, err = tmpCon.Do("ping"); err != nil {
			log.Println("connect to Redis failed again")
			return err
		}
	}

	return nil
}

//CheckResource is the REST layer to access Asset and Space
type CheckResource struct {
	Assets map[string]Asset
	Spaces map[string]Space
}

// WebService creates a new service that can handle REST requests.
func (c *CheckResource) WebService(r *RestContext) *restful.WebService {

	ws := new(restful.WebService)
	ws.Path("/v1").Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON).
		ApiVersion("v1")

	// install a webservice filter (processed before any route) & basic auth filter
	ws.Filter(measureTime).Filter(ncsaCommonLogFormatLogger()).Filter(basicAuthenticate)

	// GET
	ws.Route(ws.GET("/").To(r.showHomePage).
		//docs
		Doc("Get the homepage.").
		Writes(restful.MIME_JSON).
		Returns(200, "OK", restful.MIME_JSON).
		DefaultReturns("OK", restful.MIME_JSON))

	ws.Route(ws.GET("/spaces/{space-name}/assets/{asset-name}").To(r.findAsset).
		//docs
		Doc("Get the specified asset in the specified space.").
		Param(ws.PathParameter("space-name", "the base space's name").DataType("string").DefaultValue("base")).
		Param(ws.PathParameter("asset-name", "the asset's name").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, []string{"Assets"}).
		Writes(Asset{}).
		Returns(200, "OK", Asset{}).
		Returns(404, "Not Found", nil).
		DefaultReturns("OK", Asset{}))

	ws.Route(ws.GET("/spaces/{space-name}").To(r.findSpace).
		//docs
		Doc("Get the specified space.").
		Param(ws.PathParameter("space-name", "the space's name").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, []string{"Spaces"}).
		Writes(Space{}).
		Returns(200, "OK", Space{}).
		Returns(404, "Not Found", nil).
		DefaultReturns("OK", Space{}))

	ws.Route(ws.GET("/route/space/{space-name}").To(r.findRoute).
		//docs
		Doc("Get the optimal check route of the specified space under the given sampling rate.").
		Param(ws.PathParameter("space-name", "the root space's name").DataType("string").DefaultValue("base")).
		Param(ws.QueryParameter("sample-rate", "the global sampling rate of all the assets"+
			"belonging to the root space and all its subspaces")).
		Param(ws.QueryParameter("init-x", "the initial point's relative x position").DataType("integer")).
		Param(ws.QueryParameter("init-y", "the initial point's relative y position").DataType("integer")).
		Writes(restful.MIME_OCTET).
		Returns(200, "OK", restful.MIME_OCTET).
		Returns(http.StatusNotAcceptable, "Params Not Acceptable", nil).
		Returns(500, "Internal Error", nil).
		Returns(404, "Not Found", nil).
		DefaultReturns("OK", restful.MIME_OCTET))

	// POST
	ws.Route(ws.POST("/checkpoints").Consumes("multipart/form-data").To(r.uploadCsv).
		//docs
		Doc("Post the raw checkpoint(including space and asset) file to add spaces and/or assets.").
		Metadata(restfulspec.KeyOpenAPITags, []string{"Assets", "Spaces"}).
		Returns(http.StatusCreated, "Objects uploaded", restful.MIME_JSON).
		Returns(http.StatusRequestEntityTooLarge, "File too large", nil).
		Returns(http.StatusNotAcceptable, "Not Acceptable", nil).
		Returns(http.StatusPartialContent, "Some lines are omitted", restful.MIME_JSON).
		Returns(http.StatusConflict, "Some objects already exists", nil).
		Returns(500, "Internal Error", nil).
		DefaultReturns("Objects uploaded", nil))

	// PUT
	ws.Route(ws.PUT("/spaces/{space-name}/assets/{asset-name}").To(r.createAsset).
		//docs
		Doc("Put the asset to the space that exists.").
		Param(ws.PathParameter("space-name", "the base space's name").DataType("string").DefaultValue("base")).
		Param(ws.PathParameter("asset-name", "the asset's name").DataType("string")).
		Reads(Asset{}).
		Writes(Asset{}).
		Metadata(restfulspec.KeyOpenAPITags, []string{"Assets"}).
		Returns(http.StatusCreated, "Asset uploaded", Asset{}).
		Returns(http.StatusNotAcceptable, "Invalid asset object", nil).
		Returns(http.StatusConflict, "Some objects already exists", nil).
		Returns(500, "Internal Error", nil).
		DefaultReturns("Asset uploaded", Asset{}))

	ws.Route(ws.PUT("/spaces/{space-name}").To(r.createSpace).
		//docs
		Doc("Put the space specified.").
		Param(ws.PathParameter("space-name", "the space's name").DataType("string").DefaultValue("base")).
		Reads(Space{}).
		Writes(Space{}).
		Metadata(restfulspec.KeyOpenAPITags, []string{"Spaces"}).
		Returns(http.StatusCreated, "Space uploaded", Space{}).
		Returns(http.StatusNotAcceptable, "Invalid space object", nil).
		Returns(http.StatusConflict, "Some objects already exists", nil).
		Returns(500, "Internal Error", nil).
		DefaultReturns("Space uploaded", Space{}))

	// PATCH
	ws.Route(ws.PATCH("/spaces/{space-name}/assets/{asset-name}").To(r.updateAsset).
		//docs
		Doc("Update the asset's infomation addressed.").
		Param(ws.PathParameter("space-name", "the base space's name").DataType("string").DefaultValue("base")).
		Param(ws.PathParameter("asset-name", "the asset's name").DataType("string")).
		Param(ws.QueryParameter("rx", "the new relative x position value").DataType("number").DefaultValue("")).
		Param(ws.QueryParameter("ry", "the new relative y position value").DataType("number").DefaultValue("")).
		Param(ws.QueryParameter("weight", "the new sampling weight").DataType("number").DefaultValue("")).
		Writes(Asset{}).
		Metadata(restfulspec.KeyOpenAPITags, []string{"Assets"}).
		Returns(200, "Asset updated", Asset{}).
		Returns(http.StatusNotAcceptable, "Invalid parameters", nil).
		Returns(500, "Internal Error", nil).
		Returns(404, "Original asset not found", nil).
		DefaultReturns("Asset updated", Asset{}))

	// DELETE
	ws.Route(ws.DELETE("/spaces/{space-name}/assets/{asset-name}").To(r.deleteAsset).
		//docs
		Doc("Delete the specified asset.").
		Param(ws.PathParameter("space-name", "the base space's name").DataType("string").DefaultValue("base")).
		Param(ws.PathParameter("asset-name", "the asset's name").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, []string{"Assets"}).
		Returns(200, "Asset deleted", nil).
		Returns(500, "Internal Error", nil).
		Returns(404, "Asset not found", nil).
		DefaultReturns("Asset deleted", nil))

	ws.Route(ws.DELETE("/spaces/{space-name}").To(r.deleteSpace).
		//docs
		Doc("Delete all the space, its subspace and all the underlying assets.").
		Param(ws.PathParameter("space-name", "the root space's name").DataType("string").DefaultValue("base")).
		Metadata(restfulspec.KeyOpenAPITags, []string{"Spaces"}).
		Returns(200, "Objects deleted", nil).
		Returns(500, "Internal Error", nil).
		Returns(400, "Bad Request", nil).
		Returns(404, "Asset not found", nil).
		DefaultReturns("Objects deleted", nil))

	return ws
}

//EnrichSwaggerObject support function
func EnrichSwaggerObject(swo *spec.Swagger) {
	swo.Info = &spec.Info{
		InfoProps: spec.InfoProps{
			Title:       "Asset Check Service",
			Description: "Resource for checking office assets",
			Contact: &spec.ContactInfo{
				Name:  "Mio Xie",
				Email: "xht2012.jn@gmail.com",
			},
			License: &spec.License{
				Name: "GPL v3",
				URL:  "http://gpl.org",
			},
			Version: "1.0",
		},
	}
	swo.Tags = []spec.Tag{
		spec.Tag{TagProps: spec.TagProps{
			Name:        "Assets",
			Description: "The real asset objects in various space."}},
		spec.Tag{TagProps: spec.TagProps{
			Name: "Spaces",
			Description: "The layered space objects containing assets, " +
				"having one 'door' as the base point in it."}}}
	swo.SecurityDefinitions = map[string]*spec.SecurityScheme{
		"basic": spec.BasicAuth(),
	}
}

//basic HTTP auth
func basicAuthenticate(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	// usr/pwd = readygo/readygo2019
	u, p, ok := req.Request.BasicAuth()
	if !ok || u != "readygo-test" || p != "readygo2019" {
		resp.AddHeader("WWW-Authenticate", "Basic realm=Protected Area")
		resp.WriteErrorString(401, "401: Not Authorized")
		return
	}
	chain.ProcessFilter(req, resp)
}

//ncsaCommonLogFormatLogger : WebService NCSA Logging Filter
func ncsaCommonLogFormatLogger() restful.FilterFunction {
	return func(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
		var username = "-"
		if req.Request.URL.User != nil {
			if name := req.Request.URL.User.Username(); name != "" {
				username = name
			}
		}
		chain.ProcessFilter(req, resp)
		log.Printf("%s - %s [%s] \"%s %s %s\" %d %d",
			strings.Split(req.Request.RemoteAddr, ":")[0],
			username,
			time.Now().Format("02-Jan-2006-15:04:05"),
			req.Request.Method,
			req.Request.URL.RequestURI(),
			req.Request.Proto,
			resp.StatusCode(),
			resp.ContentLength(),
		)
	}
}

// WebService (post-process) Filter (as a struct that defines a FilterFunction)
func measureTime(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	now := time.Now()
	chain.ProcessFilter(req, resp)
	log.Printf("[webservice-filter (timer)] %v\n", time.Now().Sub(now))
}

// POST PREFIX/checkpoints
// form: {name: "csv" ...}
func (r RestContext) uploadCsv(req *restful.Request, resp *restful.Response) {
	// set memory
	req.Request.ParseMultipartForm(10 << 20) // 10M
	// get first form file, key: "csv"
	file, header, err := req.Request.FormFile("csv")
	defer file.Close()
	if err != nil {
		log.Printf("error during reading form @uploadCsv: %v\n", err)
		resp.WriteError(http.StatusRequestEntityTooLarge, err)
		return
	}

	csvFileName := "uploadcsv-" + time.Now().Format("02-Jan-2006-15-04-05-") + header.Filename
	folderPath := strings.Join([]string{os.Getenv("GOPATH"), "src", "github.com",
		"miosolo", "readygo", "archive"}, string(os.PathSeparator))
	os.Mkdir(folderPath, os.ModePerm) // ensure the folder exists
	folderPath = strings.Join([]string{folderPath, "upload"}, string(os.PathSeparator))
	os.Mkdir(folderPath, os.ModePerm) // ensure the folder exists

	// archive the uploaded csv file
	cur, err := os.Create(strings.Join([]string{folderPath, csvFileName}, string(os.PathSeparator)))
	defer cur.Close()
	if err != nil {
		log.Println(err)
		resp.WriteError(http.StatusInternalServerError, err)
		return
	}

	// copy uploaded file to local disk file
	io.Copy(cur, file)

	// save the checkpoints to Redis
	cpListPtr, errCode, err := dataio.ReadCsvByPtr(cur)
	if err != nil {
		log.Printf("error during parsing csv file @uploadCsv: %v\n", err)
		resp.WriteError(errCode, err)
		return
	}

	assetList, spaceList := unpack(*cpListPtr)
	// insert to DB
	errCode, err = r.dbInsertSpace(spaceList)
	if err != nil {
		log.Printf("error during storing space to DB @uploadCsv: %v\n", err)
		resp.WriteError(errCode, err)
		return
	}
	errCode, err = r.dbInsertAsset(assetList)
	if err != nil {
		log.Printf("error during storing asset to DB @uploadCsv: %v\n", err)
		resp.WriteError(errCode, err)
		return
	}

	resp.WriteHeaderAndEntity(http.StatusCreated, struct {
		assetInserted int
		spaceInserted int
	}{len(assetList), len(spaceList)})
}

// PUT PREFIX/spaces/{space-name}/assets/{asset-name}
// form: Asset: {name: "A", base: "", rx: 2, ry: 1, weight: 1}
func (r RestContext) createAsset(req *restful.Request, resp *restful.Response) {
	spaceName := req.PathParameter("space-name")
	assetName := req.PathParameter("asset-name")

	if resultPtr, _, _ := r.dbGetAsset(assetName, spaceName, false); resultPtr != nil {
		// already in the DB
		resp.WriteError(http.StatusConflict, errors.New("the asset provided already exists"))
		return
	}

	var newAsset Asset

	if err := req.ReadEntity(&newAsset); err != nil {
		resp.WriteError(http.StatusNotAcceptable, err)
		return
	}

	// check integrity on name and space
	if spaceName != newAsset.Base || assetName != newAsset.Name {
		resp.WriteError(http.StatusNotAcceptable, errors.New(
			"the asset's name or base space provided is in content conflict with the URL"))
		return
	}

	if errCode, err := r.dbInsertAsset([]Asset{newAsset}); err != nil {
		resp.WriteError(errCode, err)
		return
	}

	resp.WriteHeaderAndEntity(http.StatusCreated, newAsset)
}

// PUT PREFIX/spaces/{space-name}
// Space: {name: "A", base: "", rx: 2, ry: 1}
func (r RestContext) createSpace(req *restful.Request, resp *restful.Response) {
	spaceName := req.PathParameter("space-name")

	if resultPtr, _, _ := r.dbGetSpace(spaceName, false); resultPtr != nil {
		// already in the DB
		resp.WriteError(http.StatusConflict, errors.New("the space object provided already exists"))
		return
	}

	var newSpace Space
	if err := req.ReadEntity(&newSpace); err != nil {
		resp.WriteError(http.StatusNotAcceptable, err)
		return
	}

	// check integrity on name and space
	if spaceName != newSpace.Name {
		resp.WriteError(http.StatusNotAcceptable, errors.New(
			"the space object's name provided is in content conflict with the URL"))
		return
	}

	if errCode, err := r.dbInsertSpace([]Space{newSpace}); err != nil {
		resp.WriteError(errCode, err)
		return
	}

	resp.WriteHeaderAndEntity(http.StatusCreated, newSpace)
}

// GET PREFIX/spaces/{space-name}/assets/{asset-name}
func (r RestContext) findAsset(req *restful.Request, resp *restful.Response) {
	spaceName := req.PathParameter("space-name")
	assetName := req.PathParameter("asset-name")

	resultPtr, errCode, err := r.dbGetAsset(assetName, spaceName, true)
	if err != nil {
		resp.WriteError(errCode, err)
		return
	}
	resp.WriteHeaderAndEntity(http.StatusOK, *resultPtr)
}

// GET PREFIX/spaces/{space-name}
func (r RestContext) findSpace(req *restful.Request, resp *restful.Response) {
	spaceName := req.PathParameter("space-name")

	resultPtr, errCode, err := r.dbGetSpace(spaceName, true)
	if err != nil {
		resp.WriteError(errCode, err)
		return
	}
	resp.WriteHeaderAndEntity(http.StatusOK, *resultPtr)
}

// GET PREFIX/route/spaces/{space-name}?sample-rate=0.xx&init-x=xx&init-y=xx
func (r RestContext) findRoute(req *restful.Request, resp *restful.Response) {
	spaceName := req.PathParameter("space-name")
	qr := req.Request.URL.Query()

	rate, err := strconv.ParseFloat(qr.Get("sample-rate"), 64)
	if err != nil || rate <= 0 || rate > 1 {
		// invalid sample rate
		resp.WriteError(http.StatusNotAcceptable, errors.New("sampling rate out of range"))
		return
	}

	initx, err := strconv.ParseFloat(qr.Get("init-x"), 64)
	if err != nil {
		resp.WriteError(http.StatusNotAcceptable, errors.New("invalid init point's x-value"))
		return
	}

	inity, err := strconv.ParseFloat(qr.Get("init-y"), 64)
	if err != nil {
		resp.WriteError(http.StatusNotAcceptable, errors.New("invalid init point's y-value"))
		return
	}

	finalRoutePtr, errCode, err := r.calcRoute(Asset{Name: "Initial Point", Base: spaceName, Rx: initx, Ry: inity}, rate)
	if err != nil {
		resp.WriteError(errCode, err)
		return
	}

	pic, errCode, err := route.DrawRoute(finalRoutePtr.Sequence)
	if err != nil {
		resp.WriteError(errCode, err)
		return
	}

	http.ServeFile(resp.ResponseWriter, req.Request, pic)
}

// PATCH PREFIX/spaces/{space-name}/assets/{asset-name}?rx=x,ry=x,weight=x
func (r RestContext) updateAsset(req *restful.Request, resp *restful.Response) {
	spaceName := req.PathParameter("space-name")
	assetName := req.PathParameter("asset-name")
	paramQuery := req.Request.URL.Query()

	// check if any param exists
	params := []string{"rx", "ry", "weight"}
	exists := []bool{false, false, false}
	setParams := bson.D{}

	for i, s := range params {
		if paramQuery.Get(s) != "" {
			exists[i] = true
		}
	}

	// all empty err
	if !(exists[0] && exists[1] && exists[2]) {
		resp.WriteError(http.StatusNotAcceptable, errors.New("no valid query parameter"))
		return
	}

	// check if all exist param values are vaid float number
	for i, b := range exists {
		if b {
			if v, err := strconv.ParseFloat(paramQuery.Get(params[i]), 64); err == nil {
				setParams = append(setParams, bson.E{params[i], v})
			} else {
				resp.WriteError(http.StatusNotAcceptable, err)
				return
			}
		}
	}

	toSet := bson.D{{"$set", setParams}}
	newAssetPtr, errCode, err := r.dbUpdateAsset(assetName, spaceName, toSet)
	if errCode == http.StatusOK {
		resp.WriteEntity(*newAssetPtr)
	} else {
		resp.WriteError(errCode, err)
	}
}

// DELETE PREFIX/spaces/{space-name}/assets/{asset-name}
func (r RestContext) deleteAsset(req *restful.Request, resp *restful.Response) {
	spaceName := req.PathParameter("space-name")
	assetName := req.PathParameter("asset-name")

	if errCode, err := r.dbDeleteAsset(assetName, spaceName); errCode == http.StatusOK {
		resp.WriteHeader(http.StatusOK)
	} else {
		resp.WriteHeaderAndEntity(errCode, err)
	}
}

// DELETE PREFIX/spaces/{space-name}
func (r RestContext) deleteSpace(req *restful.Request, resp *restful.Response) {
	spaceName := req.PathParameter("space-name")

	if errCode, err := r.dbDeleteSpace(spaceName); err != nil {
		resp.WriteError(errCode, err)
	} else {
		resp.WriteHeader(http.StatusOK)
	}
}

func (r RestContext) showHomePage(req *restful.Request, resp *restful.Response) {
	io.WriteString(resp.ResponseWriter,
		fmt.Sprintf(`{
		"version": "1.0",
		"resources": [
			{
				"label": "Spaces",
				"description": "Sub-spaces of the whole area",
				"uri": "/spaces",
				"operations": ["GET", "PUT", "DELETE"]
			}, {
				"label": "Assets",
				"description": "Assets of the office",
				"uri": "/assets",
				"operations": ["GET", "PUT", "DELETE", "PATCH"]
			}, {
				"label": "Route",
				"description": "Randomly sampling and generated optimal route for checking in the given space",
				"uri": "/route/space",
				"operations": ["GET"]
			}
		],
		"detailed API doc": %s/apidocs.json
	}`, "https://"+req.Request.Host+req.Request.RequestURI)) // whole URL
}
