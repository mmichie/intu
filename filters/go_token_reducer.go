package filters

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strings"
)

type GoASTCompressFilter struct{}

func (f *GoASTCompressFilter) Process(content string) string {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "", content, parser.ParseComments)
	if err != nil {
		return content // Return original content if parsing fails
	}

	// Remove all comments
	node.Comments = nil

	// Compress the AST
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.GenDecl:
			compressGenDecl(x)
		case *ast.FuncDecl:
			compressFuncDecl(x)
		case *ast.BlockStmt:
			compressBlockStmt(x)
		}
		return true
	})

	// Format the compressed AST
	var buf bytes.Buffer
	format.Node(&buf, fset, node)

	// Additional string-based compression
	compressed := compressString(buf.String())

	return compressed
}

func (f *GoASTCompressFilter) Name() string {
	return "goASTCompress"
}

func compressGenDecl(d *ast.GenDecl) {
	d.Lparen = token.NoPos // Remove parentheses for single-line declarations
}

func compressFuncDecl(f *ast.FuncDecl) {
	if f.Body != nil {
		f.Body.Lbrace = token.NoPos // Remove newline before opening brace
	}
}

func compressBlockStmt(b *ast.BlockStmt) {
	b.Lbrace = token.NoPos // Remove newline before opening brace
	b.Rbrace = token.NoPos // Remove newline before closing brace
}

func compressString(s string) string {
	// Remove unnecessary whitespace
	s = strings.ReplaceAll(s, "\n", "")
	s = strings.ReplaceAll(s, "\t", "")

	// Compress multiple spaces into a single space
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}

	// Remove spaces around certain punctuation
	punctuation := []string{"(", ")", "{", "}", "[", "]", ",", ";"}
	for _, p := range punctuation {
		s = strings.ReplaceAll(s, " "+p, p)
		s = strings.ReplaceAll(s, p+" ", p)
	}

	// Special case for ':='
	s = strings.ReplaceAll(s, " := ", ":=")

	return s
}

func init() {
	Register(&GoASTCompressFilter{})
}
