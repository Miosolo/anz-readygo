package io

/*Checkpoint :
the type to trace information of an Asset/ Space,
raw type for i/o.
*/
type Checkpoint struct {
	Name     string  // global unique name
	Base     string  // the base Space it lies in
	Rx       float64 // relative x
	Ry       float64 // relative y
	IsPortal bool    // indicates if it is a sub-space to another Space, like a door
	Weight   float64 // global weight in sampling, default 1
}

//Route is the type for routing used by net package and route package
type Route struct {
	Sequence []Checkpoint
	Distance float64
}
