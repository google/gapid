// Copyright (C) 2017 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package parser

import (
	"github.com/google/gapid/core/text/parse"
	"github.com/google/gapid/gapis/gfxapi/gles/glsl/ast"
	"github.com/google/gapid/gapis/gfxapi/gles/glsl/builtins"
	pp "github.com/google/gapid/gapis/gfxapi/gles/glsl/preprocessor"
)

// newParamDirection constructs a ParamDirection given a preprocessor keyword.
func newParamDirection(kw pp.Keyword) ast.ParamDirection {
	switch kw {
	case pp.KwIn:
		return ast.DirIn
	case pp.KwOut:
		return ast.DirOut
	case pp.KwInout:
		return ast.DirInout
	default:
		return ast.DirNone
	}
}

func newPrecision(kw pp.Keyword) ast.Precision {
	switch kw {
	case pp.KwHighp:
		return ast.HighP
	case pp.KwMediump:
		return ast.MediumP
	case pp.KwLowp:
		return ast.LowP
	default:
		return ast.NoneP
	}
}

// PreprocessorExpressionEvaluator is the type of the function arugment of the Parse function. These
// functions are supposed to evaluate preprocessor expressions, given an ast.Expression and
// return an IntValue and possibly a list of errors.
type PreprocessorExpressionEvaluator func(expr ast.Expression) (val ast.IntValue, err []error)

type symbols []ast.Symbol

// Given a name, find a symbol in the list of symbols.
func (decls symbols) Get(name string) ast.Symbol {
	for _, d := range decls {
		if d.Name() == name {
			return d
		}
	}
	return nil
}

// A scope consist of a list of symbols and a pointer to a parent scope.
type scope struct {
	parent *scope
	decls  symbols
}

// Find a symbol in this scope or any of the enclosing scopes.
func (s *scope) GetSymbol(name string) (d ast.Symbol) {
	d = s.decls.Get(name)
	if d == nil && s.parent != nil {
		d = s.parent.GetSymbol(name)
	}
	return
}

// Find a value symbol in this scope or any of the enclosing scopes.
func (s *scope) GetValueSymbol(b *builder, name string) ast.ValueSymbol {
	sym := s.GetSymbol(name)
	if sym == nil {
		return nil
	}
	if vs, ok := sym.(ast.ValueSymbol); ok {
		return vs
	}
	b.Errorf("%q used in a value context.", sym.Name())
	return nil
}

// Add a symbol to this scope. It is an error if a symbol with the same name already exists in
// this scope, except if it is a function overload.
func (s *scope) AddDecl(b *builder, d ast.Symbol) {
	_, dfun := d.(ast.Function)
	if d2 := s.decls.Get(d.Name()); d2 != nil {
		_, d2fun := d2.(ast.Function)
		// Function overloads are ok
		if !(d2fun && dfun) {
			b.Errorf("Declaration of '%s' already exists in this scope.", d.Name())
		}
	}
	s.decls = append(s.decls, d)
}

// builder is a struct holding the state of the AST construction process.
type builder struct {
	err          ast.ErrorCollector
	pp           *pp.Preprocessor
	currentScope *scope
	language     ast.Language
	ppEval       PreprocessorExpressionEvaluator
}

// Peeks at the next token in the input stream.
func (b *builder) peekToken() pp.Token { return b.pp.Peek().Token }

// Returns the Cst of the next token and advances the input stream.
func (b *builder) nextCst() *parse.Leaf { return b.pp.Next().Cst }

func (b *builder) parseTranslationUnit() *ast.Ast {
	ast := &ast.Ast{}
	for b.pp.Peek().Token != nil {
		decl := b.parseExternalDeclaration()
		ast.Decls = append(ast.Decls, decl)
	}
	return ast
}

func (b *builder) parseExternalDeclaration() interface{} {
	//declaration
	//  PRECISION ...
	//  type_qualifier IDENTIFIER ...
	//  type_qualifier SEMICOLON
	//  function_prototype SEMICOLON
	//    function_declarator RPAREN
	//      function_header
	//        fully_specified_type IDENTIFIER LPAREN
	//      function_header_with_parameters
	//        function_header parameter_declaration ...
	//  init_declarator_list SEMICOLON
	//    single_declaration (COMMA ...)?
	//      INVARIANT IDENTIFIER
	//      fully_specified_type (IDENTIFIER (LBRACKET|EQ)? )?
	//        type_qualifier? type_specifier

	// As we can see, the parsing tree for a declaration is quite complex.
	// First, we pick off the easy targets.
	if b.peekToken() == pp.KwPrecision {
		return b.parsePrecisionDeclaration()
	}

	// A lot of declarations can begin with type qualifiers. Store these and then decide what
	// to do with them.
	var quals *ast.TypeQualifiers
	if pp.IsQualifier(b.peekToken()) {
		quals = b.parseTypeQualifiers()

		if b.peekToken() == pp.OpSemicolon {
			return b.parseLayoutDeclaration(quals)
		} else if b.isIdentifier(b.peekToken()) {
			return b.parseInterfaceOrInvariantDeclaration(quals)
		}
	}

	// Stash the type specifier too.
	declType := b.parseTypeSpecifier()

	if b.peekToken() == pp.OpSemicolon {
		// A semicolon at this point signals a variable declaration declaring zero
		// variables.
		return &ast.MultiVarDecl{
			Vars:         nil,
			Quals:        quals,
			Type:         declType,
			SemicolonCst: b.nextCst(),
		}
	}

	var funnyCommaCst *parse.Leaf
	if b.peekToken() == ast.BoComma {
		// Funky "int ,foo" declarations work for variables but not for functions. Skip
		// the comma.
		funnyCommaCst = b.nextCst()
	}

	// Stash the declared identifier.
	if !b.isIdentifier(b.peekToken()) {
		b.unexpectedTokenErrorSkip(pp.OpSemicolon, "identifier")
		return nil
	}
	name := b.peekToken().String()
	nameCst := b.nextCst()

	if funnyCommaCst == nil && b.peekToken() == pp.OpLParen {
		// We are parsing a function declaration.
		lparenCst := b.nextCst()

		if quals != nil {
			b.Errorf("Type qualifiers are not allowed on function return types.")
		}

		return b.parseFunctionDeclaration(name, declType, nameCst, lparenCst)
	}

	// It's a variable declaration.
	return b.parseInitDeclaratorList(name, quals, declType, nameCst, funnyCommaCst)
}

func (b *builder) parseLayoutDeclaration(quals *ast.TypeQualifiers) *ast.LayoutDecl {
	if quals.Storage != ast.StorUniform || quals.Interpolation != ast.IntNone ||
		quals.Invariant || quals.Layout == nil {

		b.Errorf("Invalid combination of qualifiers for a layout qualifier declaration.")
	}
	return &ast.LayoutDecl{
		Layout:       quals.Layout,
		UniformCst:   quals.StorageCst,
		SemicolonCst: b.nextCst(),
	}
}

func (b *builder) parseInterfaceOrInvariantDeclaration(quals *ast.TypeQualifiers) interface{} {
	if quals.Storage == ast.StorNone && quals.Interpolation == ast.IntNone &&
		quals.Invariant && quals.Layout == nil {

		return b.parseInvariantDeclaration(quals.InvariantCst)
	}
	if quals.Storage == ast.StorUniform && quals.Interpolation == ast.IntNone &&
		!quals.Invariant {

		return b.parseInterfaceDeclaration(quals.Layout, quals.StorageCst)
	}
	b.Errorf("Invalid combination of qualifiers: %v.", quals)
	return nil
}

func (b *builder) parseInvariantDeclaration(invariantCst *parse.Leaf) (ret *ast.InvariantDecl) {
	ret = &ast.InvariantDecl{InvariantCst: invariantCst}
	for {
		if !b.isIdentifier(b.peekToken()) {
			b.unexpectedTokenError("identifier")
			return
		}
		name := b.peekToken().String()
		sym := b.currentScope.GetValueSymbol(b, name)
		nameCst := b.nextCst()
		if sym == nil {
			b.Errorf("Undeclared identifier '%s'.", name)
		} else {
			ret.Vars = append(ret.Vars, &ast.VarRefExpr{Sym: sym, SymCst: nameCst})
		}
		switch b.peekToken() {
		case ast.BoComma:
			ret.CommaCst = append(ret.CommaCst, b.nextCst())
		case pp.OpSemicolon:
			ret.SemicolonCst = b.nextCst()
			if len(ret.Vars) == 0 {
				b.Errorf("Empty invariant declaration.")
			}
			return
		}
	}
}

// Guess whether the following tokens are a part of an expression or a declaration. Basically, if
// it begins with a type, it is a declaration, unless it is followed by a '('.
func (b *builder) guessIsDeclaration() bool {
	if pp.IsQualifier(b.peekToken()) ||
		b.peekToken() == pp.KwPrecision || b.peekToken() == pp.KwStruct {

		return true
	}

	var info pp.TokenInfo
	i := 0
	for {
		info = b.pp.PeekN(i)
		if info.Token == nil || (!pp.IsPrecision(info.Token) && b.getType(info) == nil) {
			break
		}
		i++
	}

	return i > 0 && info.Token != pp.OpLParen
}

func (b *builder) parseStatement(newScope bool) interface{} {
	//statement
	//  compound_statement_with_scope
	//  simple_statement
	if b.peekToken() == pp.OpLBrace {
		if newScope {
			oldScope := b.currentScope
			b.currentScope = &scope{parent: oldScope}
			defer func() { b.currentScope = oldScope }()
		}
		return b.parseCompoundStatement()
	}
	return b.parseSimpleStatement()
}

func (b *builder) parseIfStatement() (stmt *ast.IfStmt) {
	stmt = &ast.IfStmt{}
	if stmt.IfCst = b.expect(pp.KwIf).Cst; stmt.IfCst == nil {
		return
	}

	if stmt.LParenCst = b.expect(pp.OpLParen).Cst; stmt.LParenCst == nil {
		return
	}

	stmt.IfExpr = b.parseExpression()
	stmt.RParenCst = b.expectSkip(pp.OpRParen, pp.OpRParen).Cst

	oldScope := b.currentScope
	b.currentScope = &scope{parent: oldScope}
	defer func() { b.currentScope = oldScope }()

	stmt.ThenStmt = b.parseStatement(false)

	if b.peekToken() == pp.KwElse {
		stmt.ElseCst = b.nextCst()

		b.currentScope = &scope{parent: oldScope}
		stmt.ElseStmt = b.parseStatement(false)
	}
	return
}

func (b *builder) parseWhileStatement() (stmt *ast.WhileStmt) {
	stmt = &ast.WhileStmt{}
	if stmt.WhileCst = b.expect(pp.KwWhile).Cst; stmt.WhileCst == nil {
		return
	}

	if stmt.LParenCst = b.expect(pp.OpLParen).Cst; stmt.LParenCst == nil {
		return
	}

	oldScope := b.currentScope
	b.currentScope = &scope{parent: oldScope}
	defer func() { b.currentScope = oldScope }()

	stmt.Cond = b.parseCondition()
	stmt.RParenCst = b.expectSkip(pp.OpRParen, pp.OpRParen).Cst
	stmt.Stmt = b.parseStatement(false)
	return
}

func (b *builder) parseDoStatement() (stmt *ast.DoStmt) {
	stmt = &ast.DoStmt{}
	if stmt.DoCst = b.expect(pp.KwDo).Cst; stmt.DoCst == nil {
		return
	}

	oldScope := b.currentScope
	b.currentScope = &scope{parent: oldScope}
	defer func() { b.currentScope = oldScope }()

	stmt.Stmt = b.parseStatement(false)

	b.currentScope = oldScope

	if stmt.WhileCst = b.expect(pp.KwWhile).Cst; stmt.WhileCst == nil {
		return
	}
	if stmt.LParenCst = b.expect(pp.OpLParen).Cst; stmt.LParenCst == nil {
		return
	}
	stmt.Expr = b.parseExpression()
	stmt.RParenCst = b.expectSkip(pp.OpRParen, pp.OpRParen).Cst
	stmt.SemicolonCst = b.expectSkip(pp.OpSemicolon, pp.OpSemicolon).Cst
	return
}

func (b *builder) parseForStatement() (stmt *ast.ForStmt) {
	stmt = &ast.ForStmt{}
	if stmt.ForCst = b.expect(pp.KwFor).Cst; stmt.ForCst == nil {
		return
	}

	if stmt.LParenCst = b.expect(pp.OpLParen).Cst; stmt.LParenCst == nil {
		return
	}

	oldScope := b.currentScope
	b.currentScope = &scope{parent: oldScope}
	defer func() { b.currentScope = oldScope }()

	stmt.Init = b.parseExpressionOrDeclarationStatement()
	stmt.Cond = b.parseCondition()
	if stmt.Semicolon2Cst = b.expect(pp.OpSemicolon).Cst; stmt.Semicolon2Cst == nil {
		return
	}
	stmt.Loop = b.parseExpression()
	stmt.RParenCst = b.expectSkip(pp.OpRParen, pp.OpRParen).Cst
	stmt.Body = b.parseStatement(false)
	return
}

func (b *builder) parseSwitchStatement() (stmt *ast.SwitchStmt) {
	stmt = &ast.SwitchStmt{}
	stmt.SwitchCst = b.nextCst()
	if stmt.LParenCst = b.expect(pp.OpLParen).Cst; stmt.LParenCst == nil {
		return
	}
	stmt.Expr = b.parseExpression()
	stmt.RParenCst = b.expectSkip(pp.OpRParen, pp.OpRParen).Cst
	stmt.Stmts = b.parseCompoundStatement()
	return
}

func (b *builder) parseSimpleStatement() interface{} {
	//simple_statement
	//  selection_statement
	//    IF LPAREN expression RPAREN selection_rest_statement
	//  switch_statement
	//    SWITCH LPAREN expression RPAREN LBRACE switch_statement_list RBRACE
	//  case_label
	//    CASE expression COLON
	//    DEFAULT COLON
	//  iteration_statement
	//    WHILE LPAREN condition RPAREN statement_no_new_scope
	//    DO statement_with_scope WHILE LPAREN expression RPAREN SEMICOLON
	//    FOR LPAREN for_init_statement for_rest_statement RPAREN statement_no_new_scope
	//  jump_statement
	//    CONTINUE SEMICOLON
	//    BREAK SEMICOLON
	//    RETURN expression? SEMICOLON
	//    DISCARD SEMICOLON
	//  declaration_statement
	//    declaration --> pED
	//  expression_statement
	//    expression SEMICOLON
	//      ...
	//        (INC|DEC|PLUS|DASH|BANG|TILDE) unary_expression
	//        function_call
	//          ...
	//            function_identifier LPAREN
	//              type_specifier
	//              IDENTIFIER
	//              FIELD_SELECTION
	//        primary_expression
	//          variable_identifier
	//            IDENTIFIER
	//          INTCONSTANT
	//          UINTCONSTANT
	//          FLOATCONSTANT
	//          BOOLCONSTANT
	//          LPAREN expression RPAREN
	//    SEMICOLON

	// A grammar for the simple statement is quite complex. However, most statements begin
	// with a keyword and can be easily separated. Then we are left just with the
	// declaration-or-expression case, which we have a function for.
	switch b.peekToken() {
	case pp.OpSemicolon:
		return &ast.EmptyStmt{SemicolonCst: b.nextCst()}
	case pp.KwIf:
		return b.parseIfStatement()
	case pp.KwDo:
		return b.parseDoStatement()
	case pp.KwFor:
		return b.parseForStatement()
	case pp.KwWhile:
		return b.parseWhileStatement()
	case pp.KwSwitch:
		return b.parseSwitchStatement()
	case pp.KwCase:
		stmt := &ast.CaseStmt{}
		stmt.CaseCst = b.nextCst()
		stmt.Expr = b.parseExpression()
		stmt.ColonCst = b.expectSkip(pp.OpColon, pp.OpColon).Cst
		return stmt
	case pp.KwDefault:
		stmt := &ast.DefaultStmt{}
		stmt.DefaultCst = b.nextCst()
		stmt.ColonCst = b.expectSkip(pp.OpColon, pp.OpColon).Cst
		return stmt
	case pp.KwContinue:
		stmt := &ast.ContinueStmt{}
		stmt.ContinueCst = b.nextCst()
		stmt.SemicolonCst = b.expectSkip(pp.OpSemicolon, pp.OpSemicolon).Cst
		return stmt
	case pp.KwBreak:
		stmt := &ast.BreakStmt{}
		stmt.BreakCst = b.nextCst()
		stmt.SemicolonCst = b.expectSkip(pp.OpSemicolon, pp.OpSemicolon).Cst
		return stmt
	case pp.KwDiscard:
		stmt := &ast.DiscardStmt{}
		stmt.DiscardCst = b.nextCst()
		stmt.SemicolonCst = b.expectSkip(pp.OpSemicolon, pp.OpSemicolon).Cst
		return stmt
	case pp.KwReturn:
		stmt := &ast.ReturnStmt{}
		stmt.ReturnCst = b.nextCst()
		if b.peekToken() != pp.OpSemicolon {
			stmt.Expr = b.parseExpression()
		}
		stmt.SemicolonCst = b.expectSkip(pp.OpSemicolon, pp.OpSemicolon).Cst
		return stmt
	}
	return b.parseExpressionOrDeclarationStatement()
}

func (b *builder) parseCondition() (ret interface{}) {
	if b.guessIsDeclaration() {
		vd := &ast.VariableSym{SymType: b.parseTypeSpecifier()}
		ret = &ast.VarDeclCond{Sym: vd}

		if _, ok := b.peekToken().(pp.Identifier); !ok {
			b.unexpectedTokenError("identifier")
			return
		}
		vd.SymName = b.peekToken().String()
		vd.NameCst = b.nextCst()

		if vd.EqualCst = b.expect(ast.BoAssign).Cst; vd.EqualCst == nil {
			return
		}
		vd.Init = b.parseInitializer()
		return
	} else {
		expr := b.parseExpression()
		return &ast.ExpressionCond{Expr: expr}
	}
}

func (b *builder) parseExpressionOrDeclarationStatement() interface{} {
	if b.guessIsDeclaration() {
		decl := b.parseExternalDeclaration()
		return &ast.DeclarationStmt{Decl: decl}
	} else {
		expr := &ast.ExpressionStmt{Expr: b.parseExpression()}
		expr.SemicolonCst = b.expectSkip(pp.OpSemicolon, pp.OpSemicolon).Cst
		return expr
	}
}

func (b *builder) parseInitDeclaratorList(name string, quals *ast.TypeQualifiers,
	declType ast.Type, nameCst, funnyCommaCst *parse.Leaf) (ret *ast.MultiVarDecl) {

	ret = &ast.MultiVarDecl{
		Quals:         quals,
		Type:          declType,
		FunnyComma:    funnyCommaCst != nil,
		FunnyCommaCst: funnyCommaCst,
	}
	for {
		declType = ret.Type
		if b.peekToken() == pp.OpLBracket {
			left := b.nextCst()
			var size ast.Expression
			if b.peekToken() != pp.OpRBracket {
				size = b.parseConstantExpression()
			}
			if right := b.expect(pp.OpRBracket).Cst; right != nil {
				declType = &ast.ArrayType{Base: declType, Size: size, LBracketCst: left, RBracketCst: right}
			} else {
				return
			}
		}
		d := &ast.VariableSym{
			SymType: declType,
			SymName: name,
			Quals:   quals,
			NameCst: nameCst,
		}
		if array, ok := declType.(*ast.ArrayType); ok {
			d.SymType = array.Copy()
		}
		if b.peekToken() == ast.BoAssign {
			d.EqualCst = b.nextCst()
			d.Init = b.parseInitializer()
		}
		b.currentScope.AddDecl(b, d)
		ret.Vars = append(ret.Vars, d)

		switch b.peekToken() {
		case pp.OpSemicolon:
			ret.SemicolonCst = b.nextCst()
			return
		case ast.BoComma:
			ret.CommaCst = append(ret.CommaCst, b.nextCst())
		default:
			b.unexpectedTokenErrorSkip(pp.OpSemicolon, ast.BoComma, pp.OpSemicolon)
			return
		}

		if !b.isIdentifier(b.peekToken()) {
			b.unexpectedTokenError("identifier")
			return
		}
		name = b.peekToken().String()
		nameCst = b.nextCst()
	}
}

func (b *builder) parseInitializer() ast.Expression {
	return b.parseAssignmentExpression()
}

func (b *builder) parseFunctionDeclaration(name string, ret ast.Type,
	nameCst, lparenCst *parse.Leaf) (fun *ast.FunctionDecl) {

	if ret, ok := ret.(*ast.StructType); ok && ret.StructDef {
		b.Errorf("Structure definitions are not allowed in function return values.")
	}

	fun = &ast.FunctionDecl{
		SymName:   name,
		RetType:   ret,
		NameCst:   nameCst,
		LParenCst: lparenCst,
	}
	switch b.peekToken() {
	case ast.TVoid:
		fun.VoidPresent = true
		fun.VoidCst = b.nextCst()
		fun.RParenCst = b.expectSkip(pp.OpRParen, pp.OpRParen).Cst
	case pp.OpRParen:
		fun.RParenCst = b.nextCst()
	default:
	LOOP:
		for {
			fun.Params = append(fun.Params, b.parseParameterDeclaration())
			switch b.peekToken() {
			case ast.BoComma:
				fun.CommaCst = append(fun.CommaCst, b.nextCst())
			case pp.OpRParen:
				fun.RParenCst = b.nextCst()
				break LOOP
			default:
				b.unexpectedTokenError(ast.BoComma, pp.OpRParen)
				break LOOP
			}
		}
	}

	b.currentScope.AddDecl(b, fun)

	definition := b.peekToken() != pp.OpSemicolon

	oldScope := b.currentScope
	b.currentScope = &scope{parent: oldScope}
	defer func() { b.currentScope = oldScope }()

	for i, p := range fun.Params {
		if p.SymName == "" {
			if definition {
				b.Errorf("Formal parameter %d of function '%s' lacks a name.", i+1, name)
			}
		} else {
			b.currentScope.AddDecl(b, p)
		}
	}

	if !definition {
		fun.SemicolonCst = b.nextCst()
		return
	}

	fun.Stmts = b.parseCompoundStatement()
	return
}

func (b *builder) parseCompoundStatement() (ret *ast.CompoundStmt) {
	ret = &ast.CompoundStmt{}
	if ret.LBraceCst = b.expect(pp.OpLBrace).Cst; ret.LBraceCst == nil {
		return
	}

	for b.peekToken() != pp.OpRBrace && b.peekToken() != nil {
		ret.Stmts = append(ret.Stmts, b.parseStatement(true))
	}
	ret.RBraceCst = b.nextCst()
	return
}

func (b *builder) parseParameterDeclaration() (ret *ast.FuncParameterSym) {
	ret = &ast.FuncParameterSym{}
	// CONST? (IN|OUT|INOUT)? type_specifier (IDENTIFIER (LBRACKET constant_expression RBRACKET)? )?

	if b.peekToken() == pp.KwConst {
		ret.Const = true
		ret.ConstCst = b.nextCst()
	}

	switch b.peekToken() {
	case pp.KwIn, pp.KwOut, pp.KwInout:
		ret.Direction = newParamDirection(b.peekToken().(pp.Keyword))
		ret.DirectionCst = b.nextCst()
	}

	ret.SymType = b.parseTypeSpecifier()
	if ret.SymType == nil {
		return
	}
	if declType, ok := ret.SymType.(*ast.StructType); ok && declType.StructDef {
		b.Errorf("Structure definitions are not allowed in function prototypes.")
	}

	if array, ok := ret.SymType.(*ast.ArrayType); ok {
		if array.Size == nil {
			b.Errorf("Array size must be specified for function parameters.")
		}
	}

	if !b.isIdentifier(b.peekToken()) {
		return
	}
	ret.SymName = b.peekToken().String()
	ret.NameCst = b.nextCst()

	if b.peekToken() != pp.OpLBracket {
		return
	}
	ret.ArrayAfterVar = true
	if _, ok := ret.SymType.(*ast.ArrayType); ok {
		b.Errorf("Declaring an array of arrays.")
	}

	left := b.nextCst()
	size := b.parseConstantExpression()
	right := b.expectSkip(pp.OpRBracket, pp.OpRBracket).Cst

	ret.SymType = &ast.ArrayType{Base: ret.SymType, Size: size, LBracketCst: left, RBracketCst: right}
	return
}

func setPrecision(t ast.Type, prec ast.Precision, cst *parse.Leaf) bool {
	switch t := t.(type) {
	case *ast.BuiltinType:
		t.Precision = prec
		t.PrecisionCst = cst
		return true
	case *ast.ArrayType:
		return setPrecision(t.Base, prec, cst)
	default:
		return false
	}
}

func (b *builder) parseTypeSpecifier() (t ast.Type) {
	// precision_specifier
	//   (LOWP|MEDIUMP|HIGHP)? (VOID|FLOAT|...) (LBRACKET constant_expression? RBRACKET)?
	precision := ast.NoneP
	var cst *parse.Leaf
	if pp.IsPrecision(b.peekToken()) {
		precision = newPrecision(b.peekToken().(pp.Keyword))
		cst = b.nextCst()
	}
	t = b.parseTypeSpecifierNoPrec()
	if t == nil {
		return
	}
	if precision != ast.NoneP {
		if !setPrecision(t, precision, cst) {
			b.Errorf("Type %v cannot be used with a precision specifier.", Formatter(t))
		}
	}
	return
}

func (b *builder) parseInterfaceDeclaration(layout *ast.LayoutQualifier,
	uniformCst *parse.Leaf) (ret *ast.UniformDecl) {

	ret = &ast.UniformDecl{}
	if !b.isIdentifier(b.peekToken()) {
		b.unexpectedTokenError("identifier")
		return
	}
	ret.Block = &ast.UniformBlock{
		SymName:    b.peekToken().String(),
		Layout:     layout,
		UniformCst: uniformCst,
	}
	ret.Block.NameCst = b.nextCst()

	ret.Block.LBraceCst = b.expect(pp.OpLBrace).Cst
	if ret.Block.LBraceCst == nil {
		return
	}

	b.currentScope.AddDecl(b, ret.Block)

	// We assume the interface declaration will be named and put the contained variables in a
	// separate scope.
	oldScope := b.currentScope
	b.currentScope = &scope{parent: oldScope}
	defer func() { b.currentScope = oldScope }()

	for b.peekToken() != pp.OpRBrace && b.peekToken() != nil {
		ret.Block.Vars = append(ret.Block.Vars, b.parseStructDeclaration())
	}
	if b.peekToken() == nil {
		return
	}
	ret.Block.RBraceCst = b.nextCst()

	newScope := b.currentScope
	b.currentScope = oldScope

	if b.isIdentifier(b.peekToken()) {
		ret.SymName = b.peekToken().String()
		ret.NameCst = b.nextCst()

		if b.peekToken() == pp.OpLBracket {
			ret.LBracketCst = b.nextCst()
			ret.Size = b.parseConstantExpression()
			ret.RBracketCst = b.expect(pp.OpRBracket).Cst
		}

		b.currentScope.AddDecl(b, ret)
	} else {
		// Oops, it is an unnamed interface declaration. We now manually move all the
		// declarations into the outer scope.
		for _, d := range newScope.decls {
			b.currentScope.AddDecl(b, d)
		}
	}
	ret.SemicolonCst = b.expect(pp.OpSemicolon).Cst
	return
}

// A token is a type if it is a type keyword or if a type with this name has been declared already.
func (b *builder) getType(t pp.TokenInfo) ast.Type {
	if typeKw, ok := t.Token.(ast.BareType); ok {
		return &ast.BuiltinType{Type: typeKw, TypeCst: t.Cst}
	}
	if _, ok := t.Token.(pp.Identifier); !ok {
		return nil
	}
	if d, ok := b.currentScope.GetSymbol(t.Token.String()).(*ast.StructSym); ok {
		return &ast.StructType{Sym: d, StructDef: false, NameCst: t.Cst}
	}
	return nil
}

// A token is an identifier if it was parsed as an identifier by the preprocessor and this name
// hasn't been assigned to a type already.
func (b *builder) isIdentifier(t pp.Token) (r bool) {
	_, r = t.(pp.Identifier)
	if r {
		r = b.getType(pp.TokenInfo{Token: t}) == nil
	}
	return
}

var storageQualifierMap = map[pp.Token]ast.StorageQualifier{
	pp.KwConst:     ast.StorConst,
	pp.KwUniform:   ast.StorUniform,
	pp.KwIn:        ast.StorIn,
	pp.KwOut:       ast.StorOut,
	pp.KwAttribute: ast.StorAttribute,
	pp.KwVarying:   ast.StorVarying,
}

func (b *builder) parseTypeQualifiers() (ret *ast.TypeQualifiers) {
	// The specification is very strict about the possible order and combination of type
	// qualifiers. These checks emulate that. They could potentially be relaxed or moved to a
	// later stage of the analysis.
	ret = &ast.TypeQualifiers{}
	for {
		switch b.peekToken() {
		case pp.KwInvariant:
			if ret.Invariant {
				b.Errorf("Invariant qualifier specified twice.")
			}
			if ret.Layout != nil {
				b.Errorf("Incorrect combination of type qualifiers. 'invariant' " +
					"cannot be combined with layout qualifiers.")

			}
			if ret.Interpolation != ast.IntNone || ret.Storage != ast.StorNone {
				b.Errorf("Incorrect order of type qualifiers. 'invariant' " +
					"must go before interpolation and storage qualifiers.")
			}
			ret.Invariant = true
			ret.InvariantCst = b.nextCst()
		case pp.KwSmooth, pp.KwFlat:
			if ret.Interpolation != ast.IntNone {
				b.Errorf("Interpolation qualifier specified twice.")
			}
			if ret.Layout != nil {
				b.Errorf("Incorrect combination of type qualifiers. Interpolation " +
					"qualifiers cannot be combined with layout qualifiers.")
			}
			if ret.Storage != ast.StorNone {
				b.Errorf("Incorrect order of type qualifiers. Interpolation " +
					"qualifiers must go before storage qualifiers.")
			}
			if b.peekToken() == pp.KwSmooth {
				ret.Interpolation = ast.IntSmooth
			} else {
				ret.Interpolation = ast.IntFlat
			}
			ret.InterpolationCst = b.nextCst()
		case pp.KwLayout:
			if ret.Layout != nil {
				b.Errorf("Layout qualifier specified twice.")
			}
			if ret.Interpolation != ast.IntNone || ret.Invariant {
				b.Errorf("Incorrect combination of type qualifiers. 'layout' " +
					"cannot be combined with interpolation and invariant " +
					"qualifiers.")
			}
			if ret.Storage != ast.StorNone {
				b.Errorf("Incorrect order of type qualifiers. 'layout' " +
					"must go before storage qualifiers.")
			}
			ret.Layout = b.parseLayoutQualifier()
		case pp.KwCentroid:
			if ret.Storage != ast.StorNone {
				b.Errorf("Storage qualifier specified twice.")
			}
			ret.CentroidStorageCst = b.nextCst()
			switch b.peekToken() {
			case pp.KwIn:
				ret.StorageCst = b.nextCst()
				ret.Storage = ast.StorCentroidIn
			case pp.KwOut:
				ret.StorageCst = b.nextCst()
				ret.Storage = ast.StorCentroidOut
			default:
				b.unexpectedTokenError(pp.KwIn, pp.KwOut)
			}
		case pp.KwConst, pp.KwIn, pp.KwOut, pp.KwUniform, pp.KwAttribute, pp.KwVarying:
			if ret.Storage != ast.StorNone {
				b.Errorf("Storage qualifier specified twice.")
			}
			ret.Storage = storageQualifierMap[b.peekToken()]
			ret.StorageCst = b.nextCst()
		default:
			return
		}
	}
}

func (b *builder) parseLayoutQualifier() (ret *ast.LayoutQualifier) {
	ret = &ast.LayoutQualifier{}
	if ret.LayoutCst = b.expect(pp.KwLayout).Cst; ret.LayoutCst == nil {
		return
	}

	if ret.LParenCst = b.expect(pp.OpLParen).Cst; ret.LParenCst == nil {
		return
	}
	if b.peekToken() != pp.OpRParen {
	LOOP:
		for {
			if _, ok := b.peekToken().(pp.Identifier); !ok {
				b.unexpectedTokenErrorSkip(pp.OpRParen, "identifier")
			}
			id := ast.LayoutQualifierID{}
			id.Name = b.peekToken().String()
			id.NameCst = b.nextCst()

			if b.peekToken() == ast.BoAssign {
				id.EqualCst = b.nextCst()
				switch la := b.peekToken().(type) {
				case ast.UintValue, ast.IntValue:
					id.Value = la.(ast.Value)
					id.ValueCst = b.nextCst()
				default:
					b.unexpectedTokenErrorSkip(pp.OpRParen, "intconstant", "uintconstant")
					return
				}
			}

			ret.Ids = append(ret.Ids, id)
			switch b.peekToken() {
			case ast.BoComma:
				ret.CommaCst = append(ret.CommaCst, b.nextCst())
			case pp.OpRParen:
				break LOOP
			default:
				b.unexpectedTokenErrorSkip(pp.OpRParen, ast.BoComma, pp.OpRParen)
				return
			}
		}
	}
	ret.RParenCst = b.nextCst()
	return
}

func (b *builder) parsePrecisionDeclaration() (ret *ast.PrecisionDecl) {
	//PRECISION precision_qualifier type_specifier_no_prec SEMICOLON
	ret = &ast.PrecisionDecl{}
	ret.PrecisionCst = b.expect(pp.KwPrecision).Cst
	if ret.PrecisionCst == nil {
		return
	}

	retType, ok := b.parseTypeSpecifier().(*ast.BuiltinType)

	ret.SemicolonCst = b.expectSkip(pp.OpSemicolon, pp.OpSemicolon).Cst

	if ok {
		ret.Type = retType
		if t, _ := ast.GetPrecisionType(retType.Type); t == retType.Type {
			return
		}
	}
	b.Errorf("Invalid type for a precision declaration.")
	return
}

func (b *builder) parseTypeSpecifierNoPrec() (t ast.Type) {
	//type_specifier_nonarray
	//type_specifier_nonarray [ ]
	//type_specifier_nonarray [ constant_expression ]

	t = b.parseTypeSpecifierNonarray()

	if b.peekToken() != pp.OpLBracket {
		return
	}

	array := &ast.ArrayType{Base: t}
	t = array
	array.LBracketCst = b.nextCst()

	if b.peekToken() != pp.OpRBracket {
		array.Size = b.parseConstantExpression()
	}
	array.RBracketCst = b.expectSkip(pp.OpRBracket, pp.OpRBracket).Cst
	return
}

func (b *builder) parseTypeSpecifierNonarray() (t ast.Type) {
	//VOID|FLOAT|...
	//struct_specifier
	// STRUCT ...
	//TYPE_NAME

	if b.peekToken() == pp.KwStruct {
		return b.parseStructSpecifier()
	} else if t = b.getType(b.pp.Peek()); t != nil {
		b.nextCst()
		return t
	} else {
		b.unexpectedTokenError("type", pp.KwStruct)
		return nil
	}
}

func (b *builder) parseStructSpecifier() (ret *ast.StructType) {
	sym := &ast.StructSym{}
	ret = &ast.StructType{Sym: sym, StructDef: true}
	if sym.StructCst = b.expect(pp.KwStruct).Cst; sym.StructCst == nil {
		return
	}

	if b.isIdentifier(b.peekToken()) {
		sym.SymName = b.peekToken().String()
		sym.NameCst = b.nextCst()
	}

	if sym.LBraceCst = b.expect(pp.OpLBrace).Cst; sym.LBraceCst == nil {
		return
	}

	if sym.SymName != "" {
		b.currentScope.AddDecl(b, sym)
	}

	oldScope := b.currentScope
	b.currentScope = &scope{parent: oldScope}
	defer func() { b.currentScope = oldScope }()

	for b.peekToken() != pp.OpRBrace && b.peekToken() != nil {
		sym.Vars = append(sym.Vars, b.parseStructDeclaration())
	}
	if b.peekToken() == pp.OpRBrace {
		sym.RBraceCst = b.nextCst()
	}

	if len(sym.Vars) == 0 {
		b.Errorf("Structs must not be empty, but struct '%s' contains no declarations.", sym.SymName)
	}
	return
}

func (b *builder) parseStructDeclaration() *ast.MultiVarDecl {
	var quals *ast.TypeQualifiers
	if pp.IsQualifier(b.peekToken()) {
		quals = b.parseTypeQualifiers()
	}

	declType := b.parseTypeSpecifier()

	if declType, ok := declType.(*ast.StructType); ok && declType.StructDef {
		b.Errorf("Nested structure definitions are not supported.")
	}

	if !b.isIdentifier(b.peekToken()) {
		b.unexpectedTokenError()
	}
	name := b.peekToken().String()
	nameCst := b.nextCst()

	return b.parseInitDeclaratorList(name, quals, declType, nameCst, nil)
}

func (b *builder) parseConstantExpression() ast.Expression {
	return b.parseConditionalExpression()
}

func (b *builder) Errorf(msg string, args ...interface{}) {
	b.err.Errorf(msg, args...)
}

func (b *builder) unexpectedTokenError(expected ...interface{}) {
	b.unexpectedTokenErrorSkip(nil, expected...)
}

func (b *builder) unexpectedTokenErrorSkip(skipTo pp.Token, expected ...interface{}) pp.TokenInfo {
	line, col := b.pp.Peek().Cst.Token().Cursor()
	b.Errorf("%d:%d: Unexpected token (%v), was expecting one of: %v", line, col,
		b.peekToken(), expected)
	if skipTo == nil {
		return b.pp.Next()
	}
	for ; b.peekToken() != skipTo && b.peekToken() != nil; b.pp.Next() {
	}
	return b.pp.Next()
}

func (b *builder) expect(token pp.Token) pp.TokenInfo {
	return b.expectSkip(token, nil)
}

func (b *builder) expectSkip(token pp.Token, skipTo pp.Token) pp.TokenInfo {
	if b.peekToken() == token {
		return b.pp.Next()
	}
	return b.unexpectedTokenErrorSkip(skipTo, token)
}

func (b *builder) GetErrors() []error { return ast.ConcatErrors(b.pp.Errors(), b.err.GetErrors()) }

func parseImpl(tp *pp.Preprocessor, language ast.Language,
	eval PreprocessorExpressionEvaluator) (program interface{}, err []error) {

	b := &builder{
		currentScope: &scope{},
		pp:           tp,
		language:     language,
		ppEval:       eval,
	}
	for _, s := range builtinSymbols {
		b.currentScope.AddDecl(b, s)
	}
	for i := range builtins.BuiltinFunctions {
		b.currentScope.AddDecl(b, &builtins.BuiltinFunctions[i])
	}

	var ret interface{}
	if language == ast.LangPreprocessor {
		ret = b.parseLogicalOrExpression()
	} else {
		ret = b.parseTranslationUnit()
	}

	return ret, b.GetErrors()
}

func (eval PreprocessorExpressionEvaluator) parseEvaluatePreprocessorExpression(
	tp *pp.Preprocessor) (ast.IntValue, []error) {

	ret, err := parseImpl(tp, ast.LangPreprocessor, nil)
	if len(err) > 0 {
		return 0, err
	}
	return eval(ret.(ast.Expression))
}

// Parse is the main entry point into the parser. It parses GLES Shading Language programs and
// returns their AST representations. With suitable arguments it can also parse preprocessor #if
// expressions. It's arguments are:
//
// - in: the string to parse. It is automatically preprocessed.
//
// - language: which language are we parsing. In case we are parsing #if expressions
// (ast.LangPreprocessor), the eval function can be null.
//
// - eval: An evaluator function which evaluates preprocessor #if expressions. It is only used
// when parsing full GLES Shading Language programs (ast.LangVertexShader,
// ast.LangFragmentShader).
//
// The result is an object of type *ast.Ast in case of vertex and fragment shaders. In case of
// preprocessor expressions the result is an ast.Expression interface. The function also returns
// any errors it encounters during processing.
func Parse(in string, language ast.Language, eval PreprocessorExpressionEvaluator) (program interface{}, version string, extensions []pp.Extension, err []error) {
	s := pp.PreprocessStream(in, eval.parseEvaluatePreprocessorExpression, 0)
	program, err = parseImpl(s, language, eval)
	version = s.Version()
	extensions = s.Extensions()
	return
}
