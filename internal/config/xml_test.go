package config

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestXmlDecode(t *testing.T) {
	inp := `
<lexer>
  <config>
    <name>C</name>
    <alias>c</alias>
    <filename>*.c</filename>
    <filename>*.h</filename>
    <filename>*.idc</filename>
    <filename>*.x[bp]m</filename>
    <mime_type>text/x-chdr</mime_type>
    <mime_type>text/x-csrc</mime_type>
    <mime_type>image/x-xbitmap</mime_type>
    <mime_type>image/x-xpixmap</mime_type>
    <ensure_nl>true</ensure_nl>
  </config>
  <rules>
    <state name="statement">
      <rule>
        <include state="whitespace"/>
      </rule>
      <rule>
        <include state="statements"/>
      </rule>
      <rule pattern="[{}]">
        <token type="Punctuation"/>
      </rule>
      <rule pattern=";">
        <token type="Punctuation"/>
        <pop depth="1"/>
      </rule>
			<rule pattern="&#34;">
        <bygroups>
          <usingself state="root"/>
          <token type="NameFunction"/>
          <usingself state="root"/>
          <usingself state="root"/>
          <token type="Punctuation"/>
        </bygroups>
        <push state="function"/>
      </rule>
    </state>
  </rules>
</lexer>`

	buf := bytes.NewBuffer([]byte(inp))
	assert := assert.New(t)

	lex, err := DecodeLexer(buf)
	if err != nil {
		t.Fatalf("Decoding XML failed: %v\n", err)
	}

	assert.Equal("C", lex.Config.Name)

	aliases := []string{"c"}
	assert.Equal(aliases, lex.Config.Aliases)

	filenames := []string{
		"*.c",
		"*.h",
		"*.idc",
		"*.x[bp]m",
	}
	assert.Equal(filenames, lex.Config.Filenames)

	mtypes := []string{
		"text/x-chdr",
		"text/x-csrc",
		"image/x-xbitmap",
		"image/x-xpixmap",
	}
	assert.Equal(mtypes, lex.Config.MimeTypes)

	assert.Equal(true, lex.Config.EnsureNL)

	expected := Rules{
		States: []State{
			{
				Name: "statement",
				Rules: []Rule{
					{
						Include: &Include{State: "whitespace"},
					},
					{
						Include: &Include{State: "statements"},
					},
					{
						Pattern: "[{}]",
						Token:   &Token{Type: "Punctuation"},
					},
					{
						Pattern: ";",
						Token:   &Token{Type: "Punctuation"},
						Pop:     &Pop{Depth: 1},
					},
					{
						Pattern: "\"",
						ByGroups: &ByGroups{
							ByGroupsElements: []ByGroupsElement{
								{
									V: &UsingSelf{State: "root"},
								},
								{
									V: &Token{Type: "NameFunction"},
								},
								{
									V: &UsingSelf{State: "root"},
								},
								{
									V: &UsingSelf{State: "root"},
								},
								{
									V: &Token{Type: "Punctuation"},
								},
							},
						},
						Push: &Push{State: "function"},
					},
				},
			},
		},
	}

	assert.Equal(expected, lex.Rules)

}
