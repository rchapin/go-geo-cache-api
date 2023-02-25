package geostore

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	canadaSaskatchewan  *Node = NewNode(-106.72319029988779, 53.61760431337473, 1)
	canadaEdmonton      *Node = NewNode(-113.37174481536333, 53.678868921462815, 6)
	canadaDraytonValley *Node = NewNode(-114.99135644079897, 53.222683424857216, 7)
	canadaRedDeer       *Node = NewNode(-113.80500041394141, 52.26871035649865, 8)
	canadaCalgary       *Node = NewNode(-114.06243444911559, 51.04653767382061, 9)
	peruNode            *Node = NewNode(-72.27006768132226, -36.351849320377774, 2)
	australiaNode       *Node = NewNode(124.42913747263096, -23.605766549164937, 3)
	mongoliaNode        *Node = NewNode(97.00726436805842, 46.88910832340091, 4)
	oregonNode          *Node = NewNode(-120.54074440145642, 43.38552157601114, 5)
	testNodes                 = []*Node{
		canadaSaskatchewan,
		canadaEdmonton,
		canadaDraytonValley,
		canadaRedDeer,
		canadaCalgary,
		peruNode,
		australiaNode,
		mongoliaNode,
		oregonNode,
	}
)

func getTestGeoStore(maxCapacity int) GeoStore {
	// The whole globe
	quadrant := NewQuadrant(-180, -90, 180, 90, true)
	q := NewQuadTree(1, quadrant, maxCapacity)
	return NewGeoStoreInMem(q)
}

func TestFindNearest(t *testing.T) {
	g := getTestGeoStore(4)

	// Add all of the nodes that we have predefined.  The predominant number of nodes are in Canada.
	// Once added we will look for a nodes that are near one that is in relative close proximity to
	// the Western-most Canadian nodes and expect to find ones that are in the same quadrant based
	// on a 4 node maximum capacity.
	for _, n := range testNodes {
		g.Insert(n)
	}

	expectedNearestNodes := map[uint64]bool{
		6: true,
		7: true,
		8: true,
		9: true,
	}
	actualNearestNodes := g.FindNearest(52.58987722297317, -114.69660872375789, 0, 0)
	// assert.Equal(t, len(expectedNearestNodes), len(actualNearestNodes))
	actualNearestNodesMap := make(map[uint64]bool)
	for _, id := range actualNearestNodes {
		actualNearestNodesMap[id] = true
	}
	assert.True(
		t,
		reflect.DeepEqual(expectedNearestNodes, actualNearestNodesMap),
		fmt.Sprintf(
			"expected and actual node ids did not match; expected=%+v, actual=%+v",
			expectedNearestNodes,
			actualNearestNodesMap,
		),
	)

	// Render the map for fun
	printMap(g, 5)
}

func TestGpsToAbsolute(t *testing.T) {
	testData := []struct {
		gpsX, gpsY, expectedAbsX, expectedAbsY float64
	}{
		{
			gpsX:         -150.0,
			gpsY:         23.9,
			expectedAbsX: 30,
			expectedAbsY: 113.9,
		},
	}
	for _, td := range testData {
		actualAbsX, actualAbsY := GpsToAbsolute(td.gpsX, td.gpsY)
		assert.Equal(t, td.expectedAbsX, actualAbsX)
		assert.Equal(t, td.expectedAbsY, actualAbsY)
	}
}

func TestInQuadrant(t *testing.T) {
	q1 := NewQuadrant(
		-77.39510974768805,
		38.99772147594664,
		-76.92407096188735,
		39.298054420228155,
		true,
	)
	assert.True(t, q1.inQuadrant(NewNode(-77.13281118183403, 39.113426782280676, 1)))
	assert.False(t, q1.inQuadrant(NewNode(-77.49535998489928, 39.20021432781471, 2)))
}

func TestInsertNode(t *testing.T) {
	g := getTestGeoStore(4)
	g.Insert(canadaSaskatchewan)
	g.Insert(peruNode)
	g.Insert(australiaNode)
	g.Insert(mongoliaNode)
	g.Insert(oregonNode)

	rootQuadTree := g.getRootQuadTree()
	print(rootQuadTree)

	// The root QuadTree should have a nil Nodes slice and should have the following breakdown of nodes
	// in the child QuadTrees
	// NW: 2; canada and oregon nodes
	// NE: 1; mongoila
	// SW: 1; peru
	// SE: 1; australia
	assert.Nil(t, rootQuadTree.Nodes)
	nwExpectedIds := map[uint64]bool{1: true, 5: true}
	assertQTNodesContainsIds(t, rootQuadTree.NW.Nodes, nwExpectedIds)

	neExpectedIds := map[uint64]bool{4: true}
	assertQTNodesContainsIds(t, rootQuadTree.NE.Nodes, neExpectedIds)

	swExpectedIds := map[uint64]bool{2: true}
	assertQTNodesContainsIds(t, rootQuadTree.SW.Nodes, swExpectedIds)

	seExpectedIds := map[uint64]bool{3: true}
	assertQTNodesContainsIds(t, rootQuadTree.SW.Nodes, seExpectedIds)
}

func assertQTNodesContainsIds(t *testing.T, nodes []*Node, expectedIds map[uint64]bool) {
	actualIds := make(map[uint64]bool)
	for _, n := range nodes {
		actualIds[n.Id] = true
	}
	assert.Equal(t, len(expectedIds), len(actualIds))
}
