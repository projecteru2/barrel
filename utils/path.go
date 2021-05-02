package utils

import (
	"strings"
)

func RootNode(path string) (*Node, bool) {
	array := split(path)
	if len(array) == 0 {
		return nil, false
	}
	n := &Node{
		Name: array[0],
	}
	n.AddArray(array[1:])
	return n, true
}

type Node struct {
	Name  string
	Path  bool
	Child []*Node
}

func (node *Node) Add(path string) {
	node.AddArray(split(path))
}

func (node *Node) AddArray(names []string) {
	if len(names) == 0 {
		node.Path = true
		return
	}
	name := names[0]
	nextNames := names[1:]
	for _, n := range node.Child {
		if n.Name == name {
			n.AddArray(nextNames)
			return
		}
	}
	n := &Node{
		Name: name,
	}
	n.AddArray(nextNames)
	node.Child = append(node.Child, n)
}

func (node *Node) FoldedPaths(parent string) []string {
	curr := parent + "/" + node.Name
	if node.Path {
		return []string{curr}
	}
	var r []string
	for _, n := range node.Child {
		r = append(r, n.FoldedPaths(curr)...)
	}
	return r
}

func split(path string) []string {
	return FilterStringArray(
		strings.Split(path, "/"),
		filter,
	)
}

func filter(value string) bool {
	return value != ""
}
