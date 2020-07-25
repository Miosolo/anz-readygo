package net

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/gomodule/redigo/redis"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"encoding/json"
)

/*
MongoDB collection structure:
readygo(DB): {
	Space(Collection): {
		_id: "", // name
		base: "",
		rx: 0,
		ry: 0
	},
	Asset(Collection): {
		_id: {name: "", base: ""},
		rx: 0,
		ry: 0,
		weight: 1
	}
}*/

// connect to MongoDB server
func (r *RestContext) connectMongoDB() (err error) {

	// ctx, cfunc := context.WithTimeout(context.Background(), 5*time.Second)
	// defer cfunc()

	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(r.MongoURI))
	if err != nil {
		log.Println("trying connecting to MongoDB server failed: " + err.Error())
		return err
	}

	if err = client.Ping(ctx, readpref.Primary()); err != nil {
		log.Println("trying connecting to MongoDB server failed: " + err.Error())
		return err
	}

	r.mongoDB = client.Database(r.MongoDBName)
	if r.mongoDB == nil {
		return errors.New("Failed connecting to MongoDB database specified")
	}

	return nil
}

/*
Redis Key names rules:
- space-{space-name}: Space
- {Asset-name}@{space-name}: Asset
- route-{checkpointSet}: []checkpoint
*/

// Establish a connection pool
func (r *RestContext) makeRedisPool() {
	r.redisConnPool = &redis.Pool{
		MaxIdle:     1,
		MaxActive:   0,
		IdleTimeout: 180 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", r.RedisURL,
				redis.DialKeepAlive(1*time.Second),
				redis.DialPassword(r.RedisPass),
				redis.DialConnectTimeout(5*time.Second),
				redis.DialReadTimeout(1*time.Second),
				redis.DialWriteTimeout(1*time.Second))
			if err != nil {
				log.Println(err)
				return nil, err
			}
			return c, nil
		},
	}
}

//dbInsertSpace accept []Space, insert them through goroutine to Redis,
//at the same time insert to MongoDB, then returns the non-volatile DB insert error
func (r RestContext) dbInsertSpace(list []Space) (errCode int, err error) {

	// insert into mongo
	col := r.mongoDB.Collection("space")
	ctx, cancelFunc := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelFunc()

	var items []interface{}
	for _, i := range list {
		items = append(items, i)
	}

	if _, err := col.InsertMany(ctx, items); err != nil {
		log.Println(err)
		if wrEx, ok := err.(mongo.BulkWriteException); ok == true {
			for _, wrErr := range wrEx.WriteErrors {
				if wrErr.Code == 11000 { //duplicate key
					return http.StatusConflict, wrErr
				}
			}
		} else {
			return http.StatusInternalServerError, err
		}
	}

	// if no conflict & error, cache into Redis
	for _, i := range list {
		go func(item Space) {
			redisConn := r.redisConnPool.Get()
			defer redisConn.Close()
			// don't care much about insert err
			k := "space-" + item.Name
			b, _ := json.Marshal(item)
			redisConn.Do("SET", k, b)               // use space-name as the key
			redisConn.Do("EXPIRE", k, WEEK_SECONDS) // don't care expire err a bit
		}(i)
	}

	return http.StatusCreated, nil
}

//dbInsertAsset accept []Asset, insert them through goroutine to Redis,
//at the same time insert to MongoDB, then returns the non-volatile DB insert error
func (r RestContext) dbInsertAsset(list []Asset) (errCode int, err error) {

	ctx, cancelFunc := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelFunc()

	// check base space existance
	baseSpaceSet := make(map[string]bool)
	for _, a := range list {
		baseSpaceSet[a.Base] = true
	}
	for k, _ := range baseSpaceSet {
		if result := r.mongoDB.Collection("space").FindOne(ctx, bson.M{"base": k}); result.Err() != nil {
			return http.StatusForbidden, errors.New("some base spaces not exists now")
		}
	}

	// insert into mongo
	mongoHandler := r.mongoDB.Collection("asset")
	var items []interface{}
	for _, a := range list {
		items = append(items, a)
	}
	if _, err := mongoHandler.InsertMany(ctx, items); err != nil {
		log.Println(err)
		if wrEx, ok := err.(mongo.BulkWriteException); ok == true {
			for _, wrErr := range wrEx.WriteErrors {
				if wrErr.Code == 11000 { //duplicate key
					return http.StatusConflict, wrErr
				}
			}
		} else {
			return http.StatusInternalServerError, err
		}
	}

	// if no conflict & error, cache into Redis
	for _, i := range list {
		go func(item Asset) {
			redisConn := r.redisConnPool.Get()
			defer redisConn.Close()
			// don't care whether cache success or not

			k := item.Name + "@" + item.Base
			b, _ := json.Marshal(item)
			redisConn.Do("SET", k, b)               // use name@space as a compound key
			redisConn.Do("EXPIRE", k, WEEK_SECONDS) // don't care expire err a bit
		}(i)
	}

	return http.StatusCreated, nil
}

// dbGetAsset find the Asset provided with its compound key in Redis cache or MongoDB,
// if cacheFlag is set to T, then the value retrived from Mongo (after cache not hit)
// will be cached to Redis (recommend for find opreation); otherwise it is recommned to
// set it to false if just want to seek for its existance(like in PUT)
func (r RestContext) dbGetAsset(name string, base string, cacheFlag bool) (result *Asset, errCode int, err error) {
	// read from Redis cache
	redisConn := r.redisConnPool.Get()
	defer redisConn.Close()
	result = new(Asset)

	k := name + "@" + base
	data, err := redis.Bytes(redisConn.Do("GET", k))
	if err == nil {
		err = json.Unmarshal(data, result) // check data integrity
	}
	//check err agian
	if err != nil { // cache not hit or dirty data
		ctx, cf := context.WithTimeout(context.Background(), 2*time.Second)
		defer cf()
		col := r.mongoDB.Collection("asset")
		err = col.FindOne(ctx, bson.M{"name": name, "base": base}).Decode(result)
		if err != nil {
			log.Println(err)
			return nil, http.StatusNotFound, err
		}

		go func() { // write to cache
			if cacheFlag {
				b, _ := json.Marshal(*result)
				redisConn.Do("SET", k, b)
				redisConn.Do("EXPIRE", k, MONTH_SECONDS)
			}
		}()
	}
	return result, http.StatusOK, nil
}

// dbGetSpace find the space provided with its name in Redis cache or MongoDB,
// if cacheFlag is set to T, then the value retrived from Mongo (after cache not hit)
// will be cached to Redis (recommend for find opreation); otherwise it is recommned to
// set it to false if just want to seek for its existance(like in PUT)
func (r RestContext) dbGetSpace(name string, cacheFlag bool) (result *Space, errCode int, err error) {
	// read from Redis cache
	redisConn := r.redisConnPool.Get()
	result = new(Space)
	defer redisConn.Close()

	k := "space-" + name
	data, err := redis.Bytes(redisConn.Do("GET", k))
	if err == nil {
		err = json.Unmarshal(data, result) // check data integrity
	}
	//check err agian
	if err != nil { // cache not hit or dirty data
		ctx, cf := context.WithTimeout(context.Background(), 2*time.Second)
		defer cf()
		col := r.mongoDB.Collection("space")
		err = col.FindOne(ctx, bson.M{"name": name}).Decode(result)
		if err != nil {
			log.Println(err)
			return nil, http.StatusNotFound, err
		}
		go func() { // write to cache
			if cacheFlag {
				b, _ := json.Marshal(*result)
				redisConn.Do("SET", k, b)
				redisConn.Do("EXPIRE", k, MONTH_SECONDS)
			}
		}()
	}
	return result, http.StatusOK, nil
}

// dbUpdateAsset partically update the Asset, makes the new cache, and return the new Asset
func (r RestContext) dbUpdateAsset(name string, base string, toSet bson.D) (newAssetPtr *Asset, errCode int, err error) {

	ctx, cf := context.WithTimeout(context.Background(), 2*time.Second)
	defer cf()
	col := r.mongoDB.Collection("asset")
	updateResult, err := col.UpdateOne(ctx,
		bson.M{"name": name, "base": base},
		toSet)
	if err != nil {
		log.Println(err)
		return nil, http.StatusInternalServerError, err
	}
	if updateResult.MatchedCount == 0 {
		return nil, http.StatusNotFound, errors.New("the original asset does not exist")
	}

	updated := Asset{}
	result := col.FindOne(ctx, bson.M{"name": name, "base": base})
	err = result.Decode(&updated)
	if err != nil {
		log.Println(err)
		return nil, http.StatusInternalServerError, err
	}

	go func() {
		// update Redis cache
		redisConn := r.redisConnPool.Get()
		defer redisConn.Close()

		k := name + "@" + base
		b, _ := json.Marshal(updated)

		if _, err = redisConn.Do("SET", k, b); err != nil {
			log.Println("unable to update cache of key " + k + " in Redis, data may be dirty")
			log.Println(err)
		}
		redisConn.Do("EXPIRE", k, MONTH_SECONDS)
	}()

	return &updated, http.StatusOK, nil
}

// dbDeleteAsset delete the specified Asset from cache and DB and return err report.
// Note that  the cache may not be successfully deleted.
func (r RestContext) dbDeleteAsset(name string, base string) (errCode int, err error) {
	go func() {
		redisConn := r.redisConnPool.Get()
		defer redisConn.Close()

		k := name + "@" + base
		if _, err = redisConn.Do("DEL", k); err != nil {
			log.Println("unable to remove cache of key " + k + " in Redis, data may be dirty")
			log.Println(err)
		}
	}()

	ctx, cf := context.WithTimeout(context.Background(), 2*time.Second)
	defer cf()
	col := r.mongoDB.Collection("asset")
	deleteResult, err := col.DeleteOne(ctx, bson.M{"name": name, "base": base})
	if err != nil {
		log.Println(err)
		return http.StatusInternalServerError, err
	}
	if deleteResult.DeletedCount == 0 {
		return http.StatusNotFound, errors.New("the asset specified does not exist")
	}
	return http.StatusOK, nil
}

// dbDeleteSpace delete the space specified and all its Assets and sub-spaces.
// Note that removing cache may not successful
func (r RestContext) dbDeleteSpace(rootSpace string) (errCode int, err error) {
	var eg errgroup.Group

	// store subspaces -> del this space by name -> find space's Assets
	// -> del everyone in Redis -> del all Assets by base in mongo
	bfsQueue := []string{rootSpace}

	for len(bfsQueue) > 0 {
		// read from head
		base := bfsQueue[0]
		bfsQueue = bfsQueue[1:]

		{ // find subspaces
			ctx, cf := context.WithTimeout(context.Background(), 2*time.Second)
			defer cf()
			cur, err := r.mongoDB.Collection("space").Find(ctx, bson.M{"base": base})
			if err != nil {
				log.Println(err)
				return http.StatusInternalServerError, err
			}
			for cur.Next(ctx) {
				var sp Space
				err = cur.Decode(&sp)
				if err != nil {
					log.Println(err)
					return http.StatusInternalServerError, err
				}
				bfsQueue = append(bfsQueue, sp.Name)
			}
		}

		{ // delete the space in mongo in the first place to avoid err (space not exist)
			ctx, cf := context.WithTimeout(context.Background(), 2*time.Second)
			defer cf()
			col := r.mongoDB.Collection("space")
			if delResult, err := col.DeleteMany(ctx, bson.M{"name": base}); err != nil {
				return http.StatusInternalServerError, err
			} else if delResult.DeletedCount == 0 {
				return http.StatusNotFound, errors.New("the root space specified not exist")
			}
		}

		eg.Go(func() error { // redis delete this space
			redisConn := r.redisConnPool.Get()
			defer redisConn.Close()
			if _, err := redisConn.Do("DEL", "space-"+base); err != nil {
				return err
			}
			return nil
		})

		// find Assets of this space
		{
			ctx, cf := context.WithTimeout(context.Background(), 2*time.Second)
			defer cf()
			cur, err := r.mongoDB.Collection("asset").Find(ctx, bson.M{"base": base})
			if err != nil {
				log.Println(err)
				return http.StatusInternalServerError, err
			}
			for cur.Next(ctx) {
				var as Asset
				err = cur.Decode(&as)
				if err != nil {
					log.Println(err)
					return http.StatusInternalServerError, err
				}

				// for each Asset
				eg.Go(func() error { // Redis delete this space's Assets
					redisConn := r.redisConnPool.Get()
					defer redisConn.Close()

					k := as.Name + "@" + as.Base
					if _, err := redisConn.Do("DEL", k); err != nil {
						return err
					}
					return nil
				})
			}

		}

		eg.Go(func() error { // mongoDB delete this space's Assets by base space names
			ctx, cf := context.WithTimeout(context.Background(), 2*time.Second)
			defer cf()
			col := r.mongoDB.Collection("asset")
			if _, err := col.DeleteMany(ctx, bson.M{"base": base}); err != nil {
				return err
			}
			return nil
		})
	}

	if returnErr := eg.Wait(); returnErr != nil { // wait for every goroutine to finish
		log.Println(err)
		return http.StatusBadRequest, err
	}

	// without errors
	return http.StatusOK, nil
}

//LoadDemoData loads the demo data to mongoDB if demo flag is enabled (after r.InitEnv())
func (r RestContext) LoadDemoData() error {
	if _, err := r.dbInsertSpace([]Space{
		Space{Name: "webtest", Base: "", Rx: 0, Ry: 0},
		Space{Name: "webtest-Meeting Room", Base: "webtest", Rx: 2, Ry: 4}}); err != nil {
		return err
	}

	if _, err := r.dbInsertAsset([]Asset{
		Asset{Name: "A", Base: "webtest", Rx: 1, Ry: 3, Weight: 1},
		Asset{Name: "D", Base: "webtest-Meeting Room", Rx: 0, Ry: 1, Weight: 1},
		Asset{Name: "B", Base: "webtest", Rx: 3, Ry: 3, Weight: 1},
		Asset{Name: "C", Base: "webtest", Rx: 4, Ry: 0, Weight: 1}}); err != nil {
		return err
	}

	return nil
}

//UnloadDemoData deletes all the demo data of on DB
func (r RestContext) UnloadDemoData() {
	if _, err := r.dbDeleteSpace("webtest"); err != nil {
		log.Println(err)
	}
}
