package data

import (
	"bytes"
	"fmt"
)

const (
	SectionFrontEnd = "frontend"
	SectionBackEnd  = "backend"
)

type SectionId string
type Section struct {
	Type       string
	Attributes []string
	Name       string
}

func (s *Section) AppendAttributeF(format string, args ...string) {
	if s.Attributes == nil {
		s.Attributes = []string{}
	}
	s.Attributes = append(s.Attributes, args...)
}

func (s *Section) AppendAttribute(attribute string) {
	if s.Attributes == nil {
		s.Attributes = []string{}
	}
	s.Attributes = append(s.Attributes, attribute)
}

func (s *Section) Serialize(buf *bytes.Buffer) *bytes.Buffer {
	if buf == nil {
		buf = &bytes.Buffer{}
	}
	buf.WriteString(fmt.Sprintf("\n%s %s\n", s.Type, s.Name))
	for _, attribute := range s.Attributes {
		buf.WriteString(fmt.Sprintf("  %s\n", attribute))
	}
	return buf
}
