package routing

import (
	"strings"

	"example.com/m/v2/internal/utils"
)

type Trie struct {
	root *TrieNode
}

type TrieNode struct {
	children map[string]*TrieNode
	Route    *utils.Route
}

func NewTrie(routes []utils.Route) *Trie {
	newTrie := &Trie{
		root: &TrieNode{
			children: make(map[string]*TrieNode),
		},
	}

	for _, route := range routes {
		newTrie.Insert(route)
	}

	return newTrie
}

func (trie *Trie) Insert(route utils.Route) {
	prefixes := strings.Split(route.Prefix, "/")
	curr := trie.root
	for _, prefix := range prefixes {
		if prefix == "" {
			continue
		}
		next, exists := curr.children[prefix]
		if exists {
			curr = next
		} else {
			curr.children[prefix] = &TrieNode{
				children: make(map[string]*TrieNode),
			}
			curr = curr.children[prefix]
		}

	}
	curr.Route = &route
}

func (trie *Trie) Match(path string) *utils.Route {
	prefixes := strings.Split(path, "/")
	curr := trie.root
	for _, prefix := range prefixes {
		if prefix == "" {
			continue
		}

		next, exists := curr.children[prefix]
		if exists {
			curr = next
		} else {
			return curr.Route
		}
	}
	return curr.Route
}
