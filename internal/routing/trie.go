package routing

import "strings"

type Trie struct {
	root *TrieNode
}

type TrieNode struct {
	children map[string]*TrieNode
	Route    *Route
}

func NewTrie() *Trie {
	return &Trie{
		root: &TrieNode{
			children: make(map[string]*TrieNode),
		},
	}
}

func BuildTrie(config *Config) *Trie {
	newTrie := NewTrie()

	for _, route := range config.Routes {
		newTrie.Insert(route)
	}

	return newTrie
}

func (trie *Trie) Insert(route Route) {
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

func (trie *Trie) Match(path string) *Route {
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
