package x12

import "strings"

// Separators control X12 delimiters. Zero values use element * and segment ~.
type Separators struct {
	Element string // element separator (default *)
	Segment string // segment terminator (default ~)
}

func (s Separators) element() string {
	if s.Element == "" {
		return "*"
	}
	return s.Element
}

func (s Separators) segment() string {
	if s.Segment == "" {
		return "~"
	}
	return s.Segment
}

// Segment is one X12 segment (tag + data elements).
type Segment struct {
	Tag      string
	Elements []string
}

// Serialize renders the segment including terminator.
func (seg Segment) Serialize(sep Separators) string {
	parts := make([]string, 0, 1+len(seg.Elements))
	parts = append(parts, seg.Tag)
	parts = append(parts, seg.Elements...)
	return strings.Join(parts, sep.element()) + sep.segment()
}
