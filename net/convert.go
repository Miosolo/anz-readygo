// Convert upper-layer Space & Asset to lower-layer Checkpoint

package net

import (
	dataio "github.com/miosolo/readygo/io"
)

// unpack Checkpoints[] to Space[] and Asset[]
func unpack(cpList []dataio.Checkpoint) (assetList []Asset, spaceList []Space) {
	assetList = make([]Asset, 0, len(cpList)) // same length of checkpoints, for most of them are Asset
	spaceList = make([]Space, 0)

	for _, item := range cpList {
		if item.IsPortal {
			spaceList = append(spaceList, Space{
				Name: item.Name,
				Base: item.Base,
				Rx:   item.Rx,
				Ry:   item.Ry})
		} else {
			assetList = append(assetList, Asset{
				Name:   item.Name,
				Base:   item.Base,
				Rx:     item.Rx,
				Ry:     item.Ry,
				Weight: item.Weight})
		}
	}

	return assetList, spaceList
}

// package Space[] and Asset[] to checkpoint[]
func pack(asList []Asset, spList []Space) (cpList []dataio.Checkpoint) {
	cpList = make([]dataio.Checkpoint, 0, len(asList)+len(spList))
	for _, item := range asList {
		cpList = append(cpList, dataio.Checkpoint{
			Name:     item.Name,
			Base:     item.Base,
			Rx:       item.Rx,
			Ry:       item.Ry,
			IsPortal: false,
			Weight:   item.Weight,
		})
	}
	for _, item := range spList {
		cpList = append(cpList, dataio.Checkpoint{
			Name:     item.Name,
			Base:     item.Base,
			Rx:       item.Rx,
			Ry:       item.Ry,
			IsPortal: true,
		})
	}

	return cpList
}
