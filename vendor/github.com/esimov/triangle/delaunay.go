package triangle

// Point defines a struct having as components the point X and Y coordinate position.
type Point struct {
	x, y int
}

// Node defines a struct having as components the node X and Y coordinate position.
type Node struct {
	X, Y int
}

// Struct which defines a circle geometry element.
type circle struct {
	x, y, radius int
}

// newNode creates a new node.
func newNode(x, y int) Node {
	return Node{x, y}
}

// isEq check if two nodes are approximately equals.
func (n Node) isEq(p Node) bool {
	dx := n.X - p.X
	dy := n.Y - p.Y

	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	if float64(dx) < 0.0001 && float64(dy) < 0.0001 {
		return true
	}
	return false
}

// Edge struct having as component the node list.
type edge struct {
	nodes []Node
}

// newEdge creates a new edge.
func newEdge(p0, p1 Node) []Node {
	nodes := []Node{p0, p1}
	return nodes
}

// isEq check if two edge are approximately equals.
func (e edge) isEq(edge edge) bool {
	na := e.nodes
	nb := edge.nodes
	na0, na1 := na[0], na[1]
	nb0, nb1 := nb[0], nb[1]

	if (na0.isEq(nb0) && na1.isEq(nb1)) ||
		(na0.isEq(nb1) && na1.isEq(nb0)) {
		return true
	}
	return false
}

// Triangle struct defines the basic components of a triangle.
// It's constructed by nodes, it's edges and the circumcircle which describes the triangle circumference.
type Triangle struct {
	Nodes  []Node
	edges  []edge
	circle circle
}

var t = Triangle{}

// newTriangle creates a new triangle which circumcircle encloses the point to be added.
func (t Triangle) newTriangle(p0, p1, p2 Node) Triangle {
	t.Nodes = []Node{p0, p1, p2}
	t.edges = []edge{{newEdge(p0, p1)}, {newEdge(p1, p2)}, {newEdge(p2, p0)}}

	// Create a circumscribed circle of this triangle.
	// The circumcircle of a triangle is the circle which has the three vertices of the triangle lying on its circumference.
	circle := t.circle
	ax, ay := p1.X-p0.X, p1.Y-p0.Y
	bx, by := p2.X-p0.X, p2.Y-p0.Y

	m := p1.X*p1.X - p0.X*p0.X + p1.Y*p1.Y - p0.Y*p0.Y
	u := p2.X*p2.X - p0.X*p0.X + p2.Y*p2.Y - p0.Y*p0.Y
	s := 1.0 / (2.0 * (float64(ax*by) - float64(ay*bx)))

	circle.x = int(float64((p2.Y-p0.Y)*m+(p0.Y-p1.Y)*u) * s)
	circle.y = int(float64((p0.X-p2.X)*m+(p1.X-p0.X)*u) * s)

	// Calculate the distance between the node points and the triangle circumcircle.
	dx := p0.X - circle.x
	dy := p0.Y - circle.y

	// Calculate the circle radius.
	circle.radius = dx*dx + dy*dy
	t.circle = circle

	return t
}

// Delaunay defines the main components for the triangulation.
type Delaunay struct {
	width     int
	height    int
	triangles []Triangle
}

// Init initialize the delaunay structure.
func (d *Delaunay) Init(width, height int) *Delaunay {
	d.width = width
	d.height = height

	d.triangles = nil
	d.clear()

	return d
}

// clear method clears the delaunay triangles slice.
func (d *Delaunay) clear() {
	p0 := newNode(0, 0)
	p1 := newNode(d.width, 0)
	p2 := newNode(d.width, d.height)
	p3 := newNode(0, d.height)

	// Create the supertriangle, an artificial triangle which encompasses all the points.
	// At the end of the triangulation process any triangles which share edges with the supertriangle are deleted from the triangle list.
	d.triangles = []Triangle{t.newTriangle(p0, p1, p2), t.newTriangle(p0, p2, p3)}
}

// Insert will insert new triangles into the triangles slice.
func (d *Delaunay) Insert(points []Point) *Delaunay {
	var (
		i, j, k      int
		x, y, dx, dy int
		distSq       int
		polygon      []edge
		edges        []edge
		temps        []Triangle
	)

	for k = 0; k < len(points); k++ {
		x = points[k].x
		y = points[k].y

		triangles := d.triangles
		edges = nil
		temps = nil

		for i = 0; i < len(d.triangles); i++ {
			t := triangles[i]

			//Check whether the points are inside the triangle circumcircle.
			circle := t.circle
			dx = circle.x - x
			dy = circle.y - y
			distSq = dx*dx + dy*dy

			if distSq < circle.radius {
				// Save triangle edges in case they are included.
				edges = append(edges, t.edges[0], t.edges[1], t.edges[2])
			} else {
				// If not included carry over.
				temps = append(temps, t)
			}
		}

		polygon = nil
		// Check duplication of edges, delete if duplicates.
	edgesLoop:
		for i = 0; i < len(edges); i++ {
			edge := edges[i]
			for j = 0; j < len(polygon); j++ {
				// Remove identical edges.
				if edge.isEq(polygon[j]) {
					// Remove polygon from the polygon slice.
					polygon = append(polygon[:j], polygon[j+1:]...)
					continue edgesLoop
				}
			}
			// Insert new edge into the polygon slice.
			polygon = append(polygon, edge)

		}
		for i = 0; i < len(polygon); i++ {
			edge := polygon[i]
			temps = append(temps, t.newTriangle(edge.nodes[0], edge.nodes[1], newNode(x, y)))
		}
		d.triangles = temps
	}
	return d
}

// GetTriangles return the generated triangles.
func (d *Delaunay) GetTriangles() []Triangle {
	return d.triangles
}
