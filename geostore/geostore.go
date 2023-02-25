package geostore

import "sync"

//go:generate mockgen -destination=../mocks/mock_geostore.go -package=mocks   github.com/rchapin/go-geocache-api/geostore GeoStore

// FIXME: This is not the best way to generate an id for each QuadTree, but it works for now.
var quadTreeId uint64

// GpsToAbsolute converts the x (longitude) and y (latitude) values of gps coordinates to absolute
// values to be used on a grid that starts with 0,0 in the bottom-right corner, and ends with
// xMax,yMax in the top-right corner.
func GpsToAbsolute(x, y float64) (xAbs, yAbs float64) {
	return x + 180, y + 90
}

type GeoStore interface {
	Find(lat, long float64) uint64
	FindNearest(lat, long, maxDistance float64, limit int) []uint64
	Insert(node *Node)
	Shutdown() error
	getRootQuadTree() *QuadTree
}

type Node struct {
	X, Y float64
	Id   uint64
}

func NewNode(long, lat float64, id uint64) *Node {
	xAbs, yAbs := GpsToAbsolute(long, lat)
	return &Node{
		X:  xAbs,
		Y:  yAbs,
		Id: id,
	}
}

type Quadrant struct {
	XMin, YMin, XMax, YMax float64
}

func NewQuadrant(xMin, yMin, xMax, yMax float64, convertFromGps bool) *Quadrant {
	xMinAbs, yMinAbs := xMin, yMin
	xMaxAbs, yMaxAbs := xMax, yMax
	if convertFromGps {
		xMinAbs, yMinAbs = GpsToAbsolute(xMin, yMin)
		xMaxAbs, yMaxAbs = GpsToAbsolute(xMax, yMax)
	}
	return &Quadrant{
		XMin: xMinAbs,
		YMin: yMinAbs,
		XMax: xMaxAbs,
		YMax: yMaxAbs,
	}
}

func (q *Quadrant) inQuadrant(node *Node) bool {
	if node.X < q.XMin || node.X > q.XMax || node.Y < q.YMin || node.Y > q.YMax {
		return false
	}
	return true
}

type QuadTree struct {
	id           uint64
	MaxCapacity  int
	Level        int
	Nodes        []*Node
	NW           *QuadTree
	NE           *QuadTree
	SW           *QuadTree
	SE           *QuadTree
	QuadTrees    []*QuadTree
	Quadrant     *Quadrant
	isSubdivided bool
}

func NewQuadTree(level int, quadrant *Quadrant, maxCapacity int) *QuadTree {
	quadTreeId++
	var nodes []*Node
	return &QuadTree{
		id:          quadTreeId,
		MaxCapacity: maxCapacity,
		Level:       level,
		Nodes:       nodes,
		Quadrant:    quadrant,
	}
}

func (q *QuadTree) insert(node *Node) bool {
	// Is the node that we are trying to insert in the range of our Quadrant?
	if !q.Quadrant.inQuadrant(node) {
		return false
	}

	// Have we already split this QuadTree into subdivisions?
	if q.isSubdivided {
		// If so, we will iterate over all of subdivisions in this QuadTree to attempt to find a
		// location to insert this Node.
		for _, qt := range q.QuadTrees {
			if qt.insert(node) {
				break
			}
		}
	} else {
		// If it is a node that would otherwise belong in the boundaries defined in this Quadrant, have
		// we already reached our max capacity for this QuadTree?
		if len(q.Nodes) < q.MaxCapacity {
			q.Nodes = append(q.Nodes, node)
			return true
		}
		// Otherwise, we have exceeded our capacity, we need to split this QuadTree in 4 and repartition
		// our Nodes.
		return q.split(node)
	}
	return true
}

func (q *QuadTree) split(node *Node) bool {
	xOffset := q.Quadrant.XMin + ((q.Quadrant.XMax - q.Quadrant.XMin) / 2)
	yOffset := q.Quadrant.YMin + ((q.Quadrant.YMax - q.Quadrant.YMin) / 2)
	newLevel := q.Level + 1

	nwQuadrant := NewQuadrant(
		q.Quadrant.XMin,
		yOffset,
		xOffset,
		q.Quadrant.YMax,
		false,
	)
	q.NW = NewQuadTree(newLevel, nwQuadrant, q.MaxCapacity)

	neQuadrant := NewQuadrant(
		xOffset,
		yOffset,
		q.Quadrant.XMax,
		q.Quadrant.YMax,
		false,
	)
	q.NE = NewQuadTree(newLevel, neQuadrant, q.MaxCapacity)

	swQuadrant := NewQuadrant(
		q.Quadrant.XMin,
		q.Quadrant.YMin,
		xOffset,
		yOffset,
		false,
	)
	q.SW = NewQuadTree(newLevel, swQuadrant, q.MaxCapacity)

	seQuadrant := NewQuadrant(
		xOffset,
		q.Quadrant.YMin,
		q.Quadrant.XMax,
		yOffset,
		false,
	)
	q.SE = NewQuadTree(newLevel, seQuadrant, q.MaxCapacity)
	q.QuadTrees = []*QuadTree{q.NW, q.NE, q.SW, q.SE}

	// Now that we have generated subdivisions for this QuadTree flip the flag
	q.isSubdivided = true

	// Now we need to iterate through all of the Nodes in the current QuadTree's list of Nodes
	// and put them in the correct quadrant, along with the node that triggered the split.
	qts := []*QuadTree{q.NW, q.NE, q.SE, q.SW}
	nodes := append(q.Nodes, node)
NodesLoop:
	for _, n := range nodes {
		for _, qt := range qts {
			if qt.insert(n) {
				continue NodesLoop
			}
		}
	}
	q.Nodes = nil
	return true
}

type InMemGeoStore struct {
	Root *QuadTree
	mux  *sync.RWMutex
}

func NewGeoStoreInMem(root *QuadTree) *InMemGeoStore {
	return &InMemGeoStore{
		Root: root,
		mux:  &sync.RWMutex{},
	}
}

func (g *InMemGeoStore) Find(lat, long float64) uint64 {
	g.mux.Lock()
	defer g.mux.Unlock()

	// TODO:
	return 0
}

// Find will return a slice of nodes that are nearest to the provided lat/long coordinates.
// Currently, we are just going to return the nodes in the same quadrant in which this node would
// otherwise be inserted and will leave implementing max distance and limit for later.
func (g *InMemGeoStore) FindNearest(
	lat, long, maxDistance float64,
	limit int,
) []uint64 {
	g.mux.Lock()
	defer g.mux.Unlock()

	// Find the QuadTree in which this lat/long would be placed if it were inserted
	node := NewNode(long, lat, 1)
	q := findQuadTree(node, g.Root)

	// Now that we have the right QuadTree, just return a slice of the ids for each of the nodes
	retval := make([]uint64, len(q.Nodes))
	i := 0
	for _, node := range q.Nodes {
		retval[i] = node.Id
		i++
	}
	return retval
}

func findQuadTree(node *Node, root *QuadTree) *QuadTree {
	// Execute a BFS looking for the quadrant in which we would add this node if this was an insert
	// operation.  We will start by adding the root QuadTree to the queue.
	queue := make(chan *QuadTree, 128)
	queue <- root
	for q := range queue {
		if q.isSubdivided {
			// Find the child QuadTree in which this node belongs and add it to the queue
			for _, qt := range q.QuadTrees {
				if qt.Quadrant.inQuadrant(node) {
					queue <- qt
					break
				}
			}
			continue
		} else {
			// Would this node fit within the boundaries of this
			if q.Quadrant.inQuadrant(node) {
				return q
			}
		}
	}
	return nil
}

func (g *InMemGeoStore) Insert(node *Node) {
	g.mux.Lock()
	defer g.mux.Unlock()
	g.Root.insert(node)
}

func (g *InMemGeoStore) Shutdown() error {
	return nil
}

func (g *InMemGeoStore) getRootQuadTree() *QuadTree {
	return g.Root
}
