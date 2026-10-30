package model

import "time"

type NodeType string

const (
	NodeFile NodeType = "file"
	NodeDir  NodeType = "dir"
)

type Node struct {
	Path       string    `json:"path"`
	Name       string    `json:"name"`
	Type       NodeType  `json:"type"`
	SizeBytes  int64     `json:"sizeBytes"`
	ModTime    time.Time `json:"modTime"`
	ChildCount int       `json:"childCount"`
	Children   []Node    `json:"children,omitempty"`
}
