package config

// Note: to find all the elements that are allowed to nest in the lexer xml files do something like:
// srex -s '\n' ../../c.xml 'x/<rules.*\n(.*\n)*? *<\/rules>/ x/<[^ ]+/' | sort | uniq -c

import (
	"encoding/xml"
	"fmt"
	"io"
)

type Lexer struct {
	XMLName xml.Name `xml:"lexer"`
	Config  Config   `xml:"config"`
	Rules   Rules    `xml:"rules"`
}

type Config struct {
	Name      string   `xml:"name"`
	Aliases   []string `xml:"alias"`
	Filenames []string `xml:"filename"`
	MimeTypes []string `xml:"mime_type"`
	EnsureNL  bool     `xml:"ensure_nl"`
	Priority  float32  `xml:"priority,omitempty"`
}

type Rules struct {
	States []State `xml:"state"`
}

type State struct {
	Name  string `xml:"name,attr"`
	Rules []Rule `xml:"rule"`
}

type Rule struct {
	Pattern   string     `xml:"pattern,attr"`
	Include   *Include   `xml:"include"`
	Token     *Token     `xml:"token"`
	Pop       *Pop       `xml:"pop"`
	Push      *Push      `xml:"push"`
	ByGroups  *ByGroups  `xml:"bygroups"`
	UsingSelf *UsingSelf `xml:"usingself"`
}

type Include struct {
	State string `xml:"state,attr"`
}

type Token struct {
	Type string `xml:"type,attr"`
}

type Pop struct {
	Depth int `xml:"depth,attr"`
}

type Push struct {
	State string `xml:"state,attr"`
}

type ByGroups struct {
	ByGroupsElements []ByGroupsElement `xml:",any"`
}

// ByGroups contains usingself and token elements intermixed, and the order matters.
// We preserve the order by representing either of those elements by a ByGroupsElement
type ByGroupsElement struct {
	V interface{}
}

func (m *ByGroupsElement) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {

	switch start.Name.Local {
	case "token":
		m.V = &Token{}
	case "usingself":
		m.V = &UsingSelf{}
	default:
		return fmt.Errorf("unknown element: %s", start)
	}

	if err := d.DecodeElement(m.V, &start); err != nil {
		return err
	}
	return nil
}

type UsingSelf struct {
	State string `xml:"state,attr"`
}

func DecodeLexer(rdr io.Reader) (lex *Lexer, err error) {
	dec := xml.NewDecoder(rdr)

	err = dec.Decode(&lex)
	return
}
