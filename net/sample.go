package net

import (
	"math"
	"math/rand"
	"sort"
)

type rank struct {
	index   int
	feature float64
}
type rankSlice []rank

// rewrite the sort method
func (rs rankSlice) Len() int {
	return len(rs)
}
func (rs rankSlice) Swap(i, j int) {
	rs[i], rs[j] = rs[j], rs[i]
}
func (rs rankSlice) Less(i, j int) bool {
	return rs[i].feature < rs[j].feature
}

/*
sample :
Function(
	wholeList: a slice of the whole set of Assets,
	rate: sample rate) -> (sampledList: a sclice of sampled indexes)

Powered by Algorithm A
*/
func sample(wholeList []Asset, rate float64) (sampledIndexList []int) {
	N := len(wholeList)
	sampleN := int(rate * (float64(N) + 0.5)) // round

	if N == 0 {
		return []int{}
	}

	// Algorithm A by Pavlos S. Efraimidis et al.
	rankList := make([]rank, N, N)
	for i := 0; i < N; i++ {
		rankList[i].index = i
		rankList[i].feature = math.Pow(rand.Float64(), 1/wholeList[i].Weight)
	}
	sort.Sort(rankSlice(rankList))
	rankList = rankList[:sampleN]

	sampledIndexList = make([]int, 0, sampleN)
	for _, r := range rankList {
		sampledIndexList = append(sampledIndexList, r.index)
	}

	return sampledIndexList
}
