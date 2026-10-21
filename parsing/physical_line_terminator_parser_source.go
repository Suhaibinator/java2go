package parsing

import (
	"bytes"
	"fmt"
	"strconv"

	sitter "github.com/smacker/go-tree-sitter"
)

// physicalLineTerminatorParserSource changes only a private, equal-width parser
// view. A CRLF keeps its existing LF; a bare CR becomes LF. Original source and
// Unicode offset maps are never passed by mutable alias to this view.
func physicalLineTerminatorParserSource(source []byte) []byte {
	var view []byte
	for index, value := range source {
		if value != '\r' {
			continue
		}
		if view == nil {
			view = append([]byte(nil), source...)
		}
		if index+1 < len(source) && source[index+1] == '\n' {
			view[index] = ' '
		} else {
			view[index] = '\n'
		}
	}
	if view == nil {
		return source
	}
	return view
}

// The bundled grammar admits line terminators in ordinary literal fragments
// and does not require the opening line break of a text block. Validate only
// these actual literal nodes, using the same private bytes that produced them.
func physicalLineTerminatorLiteralError(root *sitter.Node, parserSource []byte) error {
	if root == nil || root.HasError() {
		return nil
	}
	var visit func(*sitter.Node) error
	visit = func(node *sitter.Node) error {
		switch node.Type() {
		case "string_literal", "character_literal":
			start, end := node.StartByte(), node.EndByte()
			if start >= end || uint64(end) > uint64(len(parserSource)) {
				return fmt.Errorf("invalid Java literal source span")
			}
			literal := parserSource[start:end]
			if node.Type() == "character_literal" {
				if literal[0] != '\'' || literal[len(literal)-1] != '\'' {
					return fmt.Errorf("invalid Java character literal delimiter")
				}
				if bytes.ContainsAny(literal, "\r\n") {
					return fmt.Errorf("invalid Java character literal: line terminator")
				}
			} else {
				opening := node.Child(0)
				if opening == nil || opening.StartByte() != start {
					return fmt.Errorf("invalid Java string literal delimiter")
				}
				switch opening.Type() {
				case "\"\"\"":
					if opening.EndByte() != start+3 || !bytes.HasPrefix(literal, []byte("\"\"\"")) {
						return fmt.Errorf("invalid Java text block delimiter")
					}
					if !physicalTextBlockOpening(literal[3:]) {
						return fmt.Errorf("invalid Java text block opening delimiter")
					}
				case "\"":
					if opening.EndByte() != start+1 || literal[0] != '"' {
						return fmt.Errorf("invalid Java string literal delimiter")
					}
					if bytes.ContainsAny(literal, "\r\n") {
						return fmt.Errorf("invalid Java string literal: line terminator")
					}
				default:
					return fmt.Errorf("invalid Java string literal delimiter")
				}
			}
		}
		for index := 0; index < int(node.NamedChildCount()); index++ {
			if err := visit(node.NamedChild(index)); err != nil {
				return err
			}
		}
		return nil
	}
	return visit(root)
}

// Before the opening line break a text block permits only space, tab and form
// feed. The actual opening quote resets Unicode backslash eligibility. Every
// accepted prefix unit is horizontal whitespace, so eligibility remains reset:
// an encountered Unicode escape can be decoded locally without scanning the
// literal body or reconstructing a whole-source Unicode state.
func physicalTextBlockOpening(prefix []byte) bool {
	for index := 0; index < len(prefix); {
		switch prefix[index] {
		case ' ', '\t', '\f':
			index++
		case '\r', '\n':
			return true
		case '\\':
			digits := index + 1
			for digits < len(prefix) && prefix[digits] == 'u' {
				digits++
			}
			if digits == index+1 || len(prefix)-digits < 4 {
				return false
			}
			value, err := strconv.ParseUint(string(prefix[digits:digits+4]), 16, 16)
			if err != nil || (value != ' ' && value != '\t' && value != '\f') {
				return false
			}
			index = digits + 4
		default:
			return false
		}
	}
	return false
}
