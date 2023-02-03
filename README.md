# Syn—Syntax Highlighting for Text Editors

Syn helps perform syntax highlighting by lexing source code text. It is based on, and large portions of the source code are taken from, Alec Thomas' [Chroma](https://github.com/alecthomas/chroma). 

Compared to Chroma, Syn does not provide formatters or styles. It does properly lex text with Windows line endings (CRLF). It also allows incremental lexing via an iterator, rather than lexing the entire document at once and providing an iterator over the produced tokens. This can be useful for text editors.

