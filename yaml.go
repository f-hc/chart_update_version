// SPDX-License-Identifier: GPL-3.0-only
//
// Copyright (C) 2026 f-hc <207619282+f-hc@users.noreply.github.com>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, version 3 of the License.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func readYAMLDocuments(path string) ([]*yaml.Node, error) {
	f, err := os.Open(filepath.Clean(path)) // #nosec G703
	if err != nil {
		return nil, fmt.Errorf("open yaml file: %w", err)
	}

	docs, err := decodeStream(yaml.NewDecoder(f))
	closeFile(f, &err)

	return docs, err
}

func closeFile(c io.Closer, err *error) {
	if closeErr := c.Close(); closeErr != nil && *err == nil {
		*err = closeErr
	}
}

func decodeStream(dec *yaml.Decoder) ([]*yaml.Node, error) {
	var n yaml.Node
	if err := dec.Decode(&n); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, nil
		}

		return nil, fmt.Errorf("decode yaml: %w", err)
	}

	rest, err := decodeStream(dec)
	if err != nil {
		return nil, err
	}

	return append([]*yaml.Node{&n}, rest...), nil
}

const (
	yamlIndent        = 2
	mappingNodeStep   = 2
	artifactHubPrefix = "# artifacthub:"
	KindApplication   = "Application"
)

func writeYAMLDocuments(_ context.Context, _ io.Writer, path string, docs []*yaml.Node) error {
	f, err := os.Create(filepath.Clean(path))
	if err != nil {
		return fmt.Errorf("create yaml file: %w", err)
	}

	var writeErr error
	defer closeFile(f, &writeErr)

	enc := yaml.NewEncoder(f)

	enc.SetIndent(yamlIndent)
	defer closeFile(enc, &writeErr)

	nodes := docs
	if len(docs) > 0 {
		first, comment := extractComment(docs[0])
		if comment != "" {
			if _, writeErr = fmt.Fprintf(f, "%s\n---\n", comment); writeErr != nil {
				writeErr = fmt.Errorf("write yaml comment: %w", writeErr)
				return writeErr
			}

			nodes = append([]*yaml.Node{first}, docs[1:]...)
		}
	}

	if writeErr = encodeStream(enc, nodes); writeErr != nil {
		return writeErr
	}

	return writeErr
}

func extractComment(n *yaml.Node) (*yaml.Node, string) {
	root := docRoot(n)
	if root.Kind != yaml.MappingNode || len(root.Content) == 0 {
		return n, ""
	}

	firstKey := root.Content[0]
	if !strings.HasPrefix(firstKey.HeadComment, artifactHubPrefix) {
		return n, ""
	}

	comment := firstKey.HeadComment

	// Clone the structure to modify it safely
	newFirstKey := *firstKey
	newFirstKey.HeadComment = ""

	newContent := make([]*yaml.Node, len(root.Content))
	copy(newContent, root.Content)
	newContent[0] = &newFirstKey

	newRoot := *root
	newRoot.Content = newContent

	// If n was DocumentNode, we need to wrap newRoot
	if n.Kind == yaml.DocumentNode {
		newDoc := *n
		newDoc.Content = make([]*yaml.Node, len(n.Content))
		copy(newDoc.Content, n.Content)
		newDoc.Content[0] = &newRoot

		return &newDoc, comment
	}

	return &newRoot, comment
}

func encodeStream(enc *yaml.Encoder, docs []*yaml.Node) error {
	if len(docs) == 0 {
		return nil
	}

	if err := enc.Encode(docs[0]); err != nil {
		return fmt.Errorf("encode yaml: %w", err)
	}

	return encodeStream(enc, docs[1:])
}

func docRoot(n *yaml.Node) *yaml.Node {
	if n.Kind == yaml.DocumentNode && len(n.Content) > 0 {
		return n.Content[0]
	}

	return n
}

func kind(n *yaml.Node) string {
	return lookup(docRoot(n), "kind")
}

func getTargetRevision(n *yaml.Node) string {
	return lookup(docRoot(n), "spec", "source", "targetRevision")
}

func setTargetRevision(n *yaml.Node, v string) {
	set(docRoot(n), v, "spec", "source", "targetRevision")
}

// getArtifactHubRepo extracts the ArtifactHub repository path from a YAML comment.
// It looks for a comment in the format "# artifacthub: org/repo" at the top of the file.
// In yaml.v3, this comment is attached to the first key of the root mapping node.
func getArtifactHubRepo(n *yaml.Node) string {
	root := docRoot(n)

	// The comment is attached to the first key in a mapping node
	if root.Kind == yaml.MappingNode && len(root.Content) > 0 {
		firstKey := root.Content[0]
		if after, ok := strings.CutPrefix(firstKey.HeadComment, artifactHubPrefix); ok {
			return strings.TrimSpace(after)
		}
	}

	return ""
}

func lookup(n *yaml.Node, path ...string) string {
	if n == nil {
		return ""
	}

	if len(path) == 0 {
		return n.Value
	}

	head, tail := path[0], path[1:]

	return lookup(mapGet(n, head), tail...)
}

func set(n *yaml.Node, value string, path ...string) {
	if len(path) == 0 {
		n.Value = value
		return
	}

	head, tail := path[0], path[1:]

	next := mapGet(n, head)
	if next == nil {
		next = &yaml.Node{Kind: yaml.MappingNode}
		mapSet(n, head, next)
	}

	set(next, value, tail...)
}

func mapGet(n *yaml.Node, key string) *yaml.Node {
	if n == nil || n.Kind != yaml.MappingNode {
		return nil
	}

	return findInContent(n.Content, key)
}

func findInContent(content []*yaml.Node, key string) *yaml.Node {
	if len(content) < mappingNodeStep {
		return nil
	}

	k, v := content[0], content[1]
	if k.Value == key {
		return v
	}

	return findInContent(content[mappingNodeStep:], key)
}

func mapSet(n *yaml.Node, key string, val *yaml.Node) {
	n.Content = append(
		n.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: key},
		val,
	)
}
