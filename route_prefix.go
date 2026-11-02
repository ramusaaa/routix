package routix

type routePrefix struct {
	prefix   string
	route    *node
	children []*routePrefix
	wildcard bool
	low      byte
	flat     *prefixFlat
}

type prefixFlat struct {
	depth  uint8
	lows   [3]byte
	widths [3]uint16
	lookup *[3][256]uint16
}

func (m *methodRoutes) addPrefix(prefix string, route *node, wildcard bool) {
	if m.prefixes == nil {
		m.prefixes = &routePrefix{prefix: prefix, route: route, wildcard: wildcard}
		return
	}
	m.prefixes.insert(prefix, route, wildcard)
}

func (n *routePrefix) child(label byte) *routePrefix {
	index := int(label) - int(n.low)
	if uint(index) < uint(len(n.children)) {
		return n.children[index]
	}
	return nil
}

func (n *routePrefix) setChild(label byte, child *routePrefix) {
	if len(n.children) == 0 {
		n.low = label
		n.children = []*routePrefix{child}
		return
	}
	if label < n.low {
		offset := int(n.low) - int(label)
		children := make([]*routePrefix, len(n.children)+offset)
		copy(children[offset:], n.children)
		n.children = children
		n.low = label
	}
	index := int(label) - int(n.low)
	if index >= len(n.children) {
		n.children = append(n.children, make([]*routePrefix, index+1-len(n.children))...)
	}
	n.children[index] = child
}

func (n *routePrefix) insert(prefix string, route *node, wildcard bool) {
	common := 0
	for common < len(prefix) && common < len(n.prefix) && prefix[common] == n.prefix[common] {
		common++
	}
	if common < len(n.prefix) {
		child := &routePrefix{prefix: n.prefix[common:], route: n.route, wildcard: n.wildcard, low: n.low, children: n.children}
		n.prefix = n.prefix[:common]
		n.route = nil
		n.wildcard = false
		n.children = nil
		n.setChild(child.prefix[0], child)
	}
	rest := prefix[common:]
	if rest == "" {
		n.route = route
		n.wildcard = wildcard
		return
	}
	if child := n.child(rest[0]); child != nil {
		child.insert(rest, route, wildcard)
		return
	}
	n.setChild(rest[0], &routePrefix{prefix: rest, route: route, wildcard: wildcard})
}

func (n *routePrefix) flatten() {
	parents := []*routePrefix{n}
	var lows [3]byte
	var widths [3]uint16
	size, chosen, chosenSize := 1, 0, 0
	for level := 0; level < 3; level++ {
		low, high := 256, -1
		valid := true
		for _, parent := range parents {
			if len(parent.children) == 0 || parent.route != nil || level > 0 && len(parent.prefix) != 1 {
				valid = false
				break
			}
			if int(parent.low) < low {
				low = int(parent.low)
			}
			end := int(parent.low) + len(parent.children) - 1
			if end > high {
				high = end
			}
		}
		if !valid {
			break
		}
		width := high - low + 1
		if size > 4096/width {
			break
		}
		size *= width
		next := make([]*routePrefix, 0)
		for _, parent := range parents {
			for _, child := range parent.children {
				if child != nil {
					next = append(next, child)
				}
			}
		}
		if size > 2*len(next) {
			break
		}
		lows[level] = byte(low)
		widths[level] = uint16(width)
		if level > 0 {
			chosen = level + 1
			chosenSize = size
		}
		parents = next
	}
	if chosen > 1 {
		table := make([]*routePrefix, chosenSize)
		var fill func(*routePrefix, int, int)
		fill = func(parent *routePrefix, level, base int) {
			for offset, child := range parent.children {
				if child == nil {
					continue
				}
				index := base*int(widths[level]) + int(parent.low) + offset - int(lows[level])
				if level+1 == chosen {
					table[index] = child
				} else {
					fill(child, level+1, index)
				}
			}
		}
		fill(n, 0, 0)
		n.children = table
		n.low = lows[0]
		flat := &prefixFlat{depth: uint8(chosen), lows: lows, widths: widths}
		if chosenSize >= 256 {
			lookup := new([3][256]uint16)
			for i := range lookup {
				for j := range lookup[i] {
					lookup[i][j] = 65535
				}
			}
			for i := 0; i < chosen; i++ {
				weight := 1
				for j := i + 1; j < chosen; j++ {
					weight *= int(widths[j])
				}
				for rank := 0; rank < int(widths[i]); rank++ {
					lookup[i][int(lows[i])+rank] = uint16(rank * weight)
				}
			}
			flat.lookup = lookup
		}
		n.flat = flat
	}
	for _, child := range n.children {
		if child != nil {
			child.flatten()
		}
	}
}
