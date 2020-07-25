package net

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/gomodule/redigo/redis"
	dataio "github.com/miosolo/readygo/io"
	"github.com/miosolo/readygo/route"
	"go.mongodb.org/mongo-driver/bson"

	"log"
)

type spaceNaviNode struct {
	root        Space
	subspaces   []*spaceNaviNode
	Assets      []Asset
	route       dataio.Route
	circuitFlag bool
}

var (
	wgTSP         sync.WaitGroup
	masterRootPtr *spaceNaviNode
	naviNodeIndex map[string]*spaceNaviNode // checkpoint type of Space -> spaceNaviNode (since the name of Space is unique)
	allAssets     []Asset
	initStand     Asset
)

// post-order traversal to sample and dispatch routing task
func (r RestContext) recursiveSampleTSP(rootPtr *spaceNaviNode) bool { // T/F : the sub-tree contains Assets after sampling -> need to routine or not
	setToString := func(m map[dataio.Checkpoint]bool) string {
		s := "{"
		for k, _ := range m {
			s += fmt.Sprintf("%v, ", k)
		}
		return s[:len(s)-2] + "}"
	}

	validSpaceList := []Space{}
	// filter the subtrees
	for _, subNode := range rootPtr.subspaces {
		if r.recursiveSampleTSP(subNode) { // have checkpoints
			validSpaceList = append(validSpaceList, subNode.root)
		}
	}

	if len(rootPtr.Assets) == 0 && len(validSpaceList) == 0 { // empty Asset list, empty sub trees
		return false
	}

	// could do computing in parallel
	wgTSP.Add(1)

	go func() {
		cpList := pack(rootPtr.Assets, validSpaceList) // must not be empty

		redisConn := r.redisConnPool.Get()
		defer redisConn.Close()
		keySet := make(map[dataio.Checkpoint]bool) // to construct a set Data Strcture as key
		for _, item := range cpList {
			keySet[item] = true
		}

		k := setToString(keySet)
		data, err := redis.Bytes(redisConn.Do("GET", "route-"+k))
		if err == nil { // cache hit
			err = json.Unmarshal(data, &(rootPtr.route)) // check data integrity
			if err == nil {
				wgTSP.Done()
				return
			}
		}

		if rootPtr == masterRootPtr {
			route.TSP(cpList, dataio.Checkpoint{
				Name:     initStand.Name,
				Base:     masterRootPtr.root.Name,
				Rx:       initStand.Rx,
				Ry:       initStand.Ry,
				IsPortal: false}, rootPtr.circuitFlag, &(rootPtr.route))

		} else {
			route.TSP(cpList, dataio.Checkpoint{
				Name:     rootPtr.root.Name,
				Base:     rootPtr.root.Base,
				Rx:       rootPtr.root.Rx,
				Ry:       rootPtr.root.Ry,
				IsPortal: true}, rootPtr.circuitFlag, &(rootPtr.route))
		}

		go func() { // cache
			b, err := json.Marshal(rootPtr.route) // calculated route
			if err != nil {
				log.Println(err)
			}
			redisConn.Do("SET", k, b)

		}()

		wgTSP.Done()
	}()

	return true
}

func (r RestContext) calcRoute(initPoint Asset, sampleRate float64) (finalRoutePtr *dataio.Route, errCode int, err error) {
	initStand = initPoint
	naviNodeIndex = make(map[string]*spaceNaviNode)
	allAssets = []Asset{}

	resultPtr, errCode, err := r.dbGetSpace(initStand.Base, true)
	if err != nil {
		log.Println(err)
		return nil, errCode, err
	}

	masterRootPtr = &spaceNaviNode{
		root:        *resultPtr,
		circuitFlag: false,
	}
	bfsQueue := make([]*spaceNaviNode, 0)
	bfsQueue = append(bfsQueue, masterRootPtr) // insert root node
	naviNodeIndex[masterRootPtr.root.Name] = masterRootPtr

	// BFS search tree
	for len(bfsQueue) > 0 {
		// read from head
		rootNode := bfsQueue[0]
		bfsQueue = bfsQueue[1:]

		//ctx, cf := context.WithTimeout(context.Background(), 2*time.Second)
		//defer cf()
		ctx := context.Background()

		// find subspaces
		cur, err := r.mongoDB.Collection("space").Find(ctx, bson.M{"base": rootNode.root.Name})
		if err != nil {
			log.Println(err)
			return nil, http.StatusInternalServerError, err
		}
		for cur.Next(ctx) {
			var sp Space
			err = cur.Decode(&sp)
			if err != nil {
				log.Println(err)
				return nil, http.StatusInternalServerError, err
			}
			newNaviNode := spaceNaviNode{root: sp, circuitFlag: true} // circuit for subspaces, need to return
			rootNode.subspaces = append(rootNode.subspaces, &newNaviNode)
			naviNodeIndex[sp.Name] = &newNaviNode
			bfsQueue = append(bfsQueue, &newNaviNode)
		}

		// find Assets of this space
		cur, err = r.mongoDB.Collection("asset").Find(ctx, bson.M{"base": rootNode.root.Name})
		if err != nil {
			log.Println(err)
			return nil, http.StatusInternalServerError, err
		}
		for cur.Next(ctx) {
			var as Asset
			err = cur.Decode(&as)
			if err != nil {
				log.Println(err)
				return nil, http.StatusInternalServerError, err
			}
			allAssets = append(allAssets, as)
		}
	}

	// sampling
	filteredIndexList := sample(allAssets, sampleRate)
	if len(filteredIndexList) == 0 {
		return nil, http.StatusNotAcceptable, errors.New("empty set after sampling")
	}

	for _, index := range filteredIndexList {
		// distributing seleted Assets
		baseNode, _ := naviNodeIndex[allAssets[index].Base]
		baseNode.Assets = append(baseNode.Assets, allAssets[index])
	}

	r.recursiveSampleTSP(masterRootPtr) // TSP bottom to up
	wgTSP.Wait()                        // until all computations compelete

	// traversal the tree & link route
	finalDistance := masterRootPtr.route.Distance
	finalSeq := masterRootPtr.route.Sequence // violent to the def of spaceNaviNode.route, but doesn't matter
	rootNodeStk := []string{}                // a stack to trace the root nodes

	for i := 0; i < len(finalSeq); i++ {
		if finalSeq[i].IsPortal { // 0: the init point
			if len(rootNodeStk) == 0 || rootNodeStk[len(rootNodeStk)-1] != finalSeq[i].Name { // a new subnode, insert its subsequence
				rootNodeStk = append(rootNodeStk, finalSeq[i].Name) //push
				subSpaceNavi, _ := naviNodeIndex[finalSeq[i].Name]
				for j := 0; j < len(subSpaceNavi.route.Sequence); j++ {
					subSpaceNavi.route.Sequence[j].Rx += subSpaceNavi.root.Rx
					subSpaceNavi.route.Sequence[j].Ry += subSpaceNavi.root.Ry // violent to the def of relative position, but doesn't matter
				}
				//subSpaceNavi.route[0] is the space portal itself, should be removed in case of merging
				finalSeq = append(finalSeq[:i+1], append(subSpaceNavi.route.Sequence[1:], finalSeq[i+1:]...)...)
				finalDistance += subSpaceNavi.route.Distance
			} else { // meet again
				rootNodeStk = rootNodeStk[:len(rootNodeStk)-1] //pop
				continue
			}
		}
	}

	return &dataio.Route{Sequence: finalSeq, Distance: finalDistance}, http.StatusOK, nil
}
