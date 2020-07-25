package route

import (
	"math"

	dataio "github.com/miosolo/readygo/io"
)

/*
TSP : solves the Trvaling Salesman Problem using DP
Portal is the initial point and endpoint of the circlar path, which is the door

NOTE: using dp will cost RAM in O(n * 2^n) level,
			so theoritical max point is 25 under 4G RAM assigned to the program
*/
func TSP(cpList []dataio.Checkpoint, Portal dataio.Checkpoint, circuitFlag bool, result *dataio.Route) {
	// change portal as the base point
	if Portal.IsPortal {
		// T->Portal is space, set its Rx,Ry to 0,0 as base point
		// F->Portal is the init point, keep its Rx,Ry in root space
		Portal.Rx = 0
		Portal.Ry = 0
	}

	// init
	cpList = append([]dataio.Checkpoint{Portal}, cpList...) // put Portal to [0]
	N := uint(len(cpList))
	sqr := func(x float64) float64 {
		return x * x
	}

	// using Euler distance:
	dis := make([][]float64, N, N)
	var ix, iy float64
	for i := 0; i < int(N); i++ {
		ix, iy = cpList[i].Rx, cpList[i].Ry
		dis[i] = make([]float64, N, N)
		for j := 0; j < int(N); j++ {
			if i < j {
				dis[i][j] = math.Sqrt(sqr(ix-cpList[j].Rx) + sqr(iy-cpList[j].Ry))
			} else if i == j {
				dis[i][j] = 0
			} else { // i > j
				dis[i][j] = dis[j][i] // dis[j][i] must be assigned in a former loop
			}
		}
	}

	type trace struct {
		cost      float64
		lastIndex uint8
	}

	INF_F64 := math.Inf(1)
	NAN_U8 := uint8(0xff)
	initTraceBlock := trace{INF_F64, NAN_U8}
	// dp[from: set][to: node]
	dp := make([][]trace, 1<<N, 1<<N) // row index: bitwase representatdataion of V set
	for i := 1; i < 1<<N; i++ {
		if 1&i != 0 { // even number <=> init point included
			dp[i] = make([]trace, N, N) // col index: destinatdataion
		} else {
			continue
		}
		for j := 0; j < int(N); j++ {
			dp[i][j] = initTraceBlock
		}
	}
	dp[1][0] = trace{0, 0} // {init} -> init

	for i := 1; i < 1<<N; i++ {
		if (i & 1) == 0 { // i not in set
			continue
		}
		// for every status
		for j := 1; j < int(N); j++ {
			// select next node to be added
			if i&(1<<uint(j)) != 0 {
				continue // if j already in set
			}
			for k := 0; k < int(N); k++ {
				// try for every node in set to relax
				if i&(1<<uint(k)) != 0 {
					// k in this set
					if dp[(1<<uint(j))|i][j].cost > dp[i][k].cost+dis[j][k] {
						dp[(1<<uint(j))|i][j].cost = dp[i][k].cost + dis[j][k]
						dp[(1<<uint(j))|i][j].lastIndex = uint8(k)
					}
					// tranform formula, add dp(VU{j}, j)
				}
			}
		}
	}

	// trace back to init Portal
	var tbSet uint   // save the trace back set status, init to U
	var tbPrev uint8 // save the prev index to trace back, which is the min_k{dp[V][j] + dis[j][k]}
	var tbNext uint8 // save the next index, which is j
	var minTour float64
	tourSeqList := make([]dataio.Checkpoint, 0, N+1) // if to make a circuit, max length will be N + 1
	if circuitFlag {
		minCircuitLen := INF_F64
		tbSet = 1<<N - 1
		for i := 1; i < int(N); i++ {
			// find the last node of the shortest Hamilton circuit
			if dp[tbSet][i].cost+dis[i][0] < minCircuitLen {
				minCircuitLen = dp[tbSet][i].cost + dis[i][0]
				tbPrev = dp[tbSet][i].lastIndex
				tbNext = uint8(i)
			}
		}
		minTour = minCircuitLen
		tourSeqList = append(tourSeqList, Portal)
	} else { // do not return to init point
		minPathLen := INF_F64
		tbSet = 1<<N - 1
		for i := 1; i < int(N); i++ {
			// find the last node of the shortest Hamilton path
			if dp[tbSet][i].cost < minPathLen {
				minPathLen = dp[tbSet][i].cost
				tbPrev = dp[tbSet][i].lastIndex
				tbNext = uint8(i)
			}
		}
		minTour = minPathLen
	}

	tourSeqList = append(tourSeqList, cpList[tbNext])
	for tbPrev != 0 {
		tourSeqList = append(tourSeqList, cpList[tbPrev])
		tbSet &= ^(1 << uint(tbNext)) // remove the target bit
		tbNext = tbPrev
		tbPrev = dp[tbSet][tbPrev].lastIndex
	}
	tourSeqList = append(tourSeqList, Portal)

	// revert the tourSeqList, since it is back to front
	for i, j := 0, len(tourSeqList)-1; i < j; i, j = i+1, j-1 {
		tourSeqList[i], tourSeqList[j] = tourSeqList[j], tourSeqList[i]
	}

	result.Sequence = tourSeqList
	result.Distance = minTour
}
