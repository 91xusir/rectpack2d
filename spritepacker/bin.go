package texture_packer_go

// Bin usedToHoldRectangularContainers

type Bin struct {
	// The width of the boundary

	W int
	// The height of the boundary

	H int
	// Rectangle list

	ReqPackRectList []*Rect
	// Whether to allow rectangle rotation

	IsRotateEnable bool
}
