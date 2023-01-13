# Syn—Syntax Highlighting for Text Editors

Syn helps perform syntax highlighting by lexing source code text. It is based on, and large portions of the source code are taken from, Alec Thomas' [Chroma](https://github.com/alecthomas/chroma). 

Compared to Chroma, Syn does not provide formatters or styles. It properly lexes text with Windows line endings (CRLF). It also provides a mechanism to save the state of lexing midway through iteration so that a new lexer can be created to lex from that point in the text, which is useful for text editors.

