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
	"github.com/google/gapid/gapis/api/gles/glsl/ast"
	pp "github.com/google/gapid/gapis/api/gles/glsl/preprocessor"
)

func (b *builder) parseFunctionCall(callee ast.Expression) (ret *ast.CallExpr) {
	ret = &ast.CallExpr{Callee: callee}
	if ret.LParenCst = b.expect(pp.OpLParen).Cst; ret.LParenCst == nil {
		return
	}

	if b.peekToken() == ast.TVoid {
		ret.VoidPresent = true
		ret.VoidCst = b.nextCst()
		ret.RParenCst = b.expectSkip(pp.OpRParen, pp.OpRParen).Cst
		return
	}

	if b.peekToken() != pp.OpRParen {
	LOOP:
		for {
			ret.Args = append(ret.Args, b.parseAssignmentExpression())
			switch b.peekToken() {
			case ast.BoComma:
				ret.CommaCst = append(ret.CommaCst, b.nextCst())
			case pp.OpRParen:
				ret.RParenCst = b.nextCst()
				break LOOP
			default:
				b.unexpectedTokenErrorSkip(pp.OpRParen, ast.BoComma, pp.OpRParen)
				break LOOP
			}
		}
	} else {
		ret.RParenCst = b.nextCst()
	}

	return
}

func (b *builder) parseAssignmentExpression() ast.Expression {
	// assignment_expression
	//   conditional_expression
	//   unary_expression assignment_operator assignment_expression

	left := b.parseConditionalExpression()
	if op, ok := b.peekToken().(ast.BinaryOp); ok && ast.IsAssignmentOp(op) {
		opCst := b.nextCst()
		right := b.parseAssignmentExpression()
		return &ast.BinaryExpr{Left: left, Op: op, Right: right, OpCst: opCst}
	}
	return left
}

func (b *builder) parseConditionalExpression() ast.Expression {
	// conditional_expression
	//   logical_or_expression
	//   logical_or_expression QUESTION expression COLON assignment_expression

	expr := b.parseLogicalOrExpression()
	if b.peekToken() == pp.OpQuestion {
		ret := &ast.ConditionalExpr{
			Cond:        expr,
			QuestionCst: b.nextCst(),
		}
		ret.TrueExpr = b.parseExpression()
		if ret.ColonCst = b.expect(pp.OpColon).Cst; ret.ColonCst == nil {
			return ret
		}
		ret.FalseExpr = b.parseAssignmentExpression()
		return ret
	}
	return expr
}

// binaryOpSequence is a list of binary operators, grouped according to operator priority.
var binaryOpSequence = [...][]ast.BinaryOp{
	{ast.BoLor},
	{ast.BoLxor},
	{ast.BoLand},
	{ast.BoBor},
	{ast.BoBxor},
	{ast.BoBand},
	{ast.BoEq, ast.BoNotEq},
	{ast.BoLess, ast.BoMore, ast.BoLessEq, ast.BoMoreEq},
	{ast.BoShl, ast.BoShr},
	{ast.BoAdd, ast.BoSub},
	{ast.BoMul, ast.BoDiv, ast.BoMod},
}

// getSemanticOp is a helper function to find a binary operator given a preprocessor
// token and a mapping.
func getSemanticOp(lex pp.Token, list []ast.BinaryOp) (sem ast.BinaryOp, found bool) {
	found = false
	if lex == nil {
		return
	}
	for _, op := range list {
		if op.String() == lex.String() {
			sem = op
			found = true
		}
	}
	return
}

// All binary expressions are parsed in the same way. We just need to get the operator precedence
// right. For this reason the operators are split into levels, as specified by the
// binaryOpSequence array.
func (b *builder) parseBinaryExpression(level int) ast.Expression {
	var left ast.Expression
	if level+1 >= len(binaryOpSequence) {
		left = b.parseUnaryExpression()
	} else {
		left = b.parseBinaryExpression(level + 1)
	}
	if op, ok := getSemanticOp(b.peekToken(), binaryOpSequence[level]); ok {
		opCst := b.nextCst()
		right := b.parseBinaryExpression(level)
		return &ast.BinaryExpr{Left: left, Op: op, Right: right, OpCst: opCst}
	}
	return left
}

func (b *builder) parseLogicalOrExpression() ast.Expression {
	return b.parseBinaryExpression(0)
}

func (b *builder) parseUnaryExpression() ast.Expression {
	if b.peekToken() == nil {
		b.unexpectedTokenError("primary expression")
		return nil
	}
	if op, present := ast.GetPrefixOp(b.peekToken().String()); present {
		opCst := b.nextCst()
		expr := b.parseUnaryExpression()
		return &ast.UnaryExpr{Op: op, Expr: expr, OpCst: opCst}
	}
	return b.parsePostfixExpression()
}

func (b *builder) parsePostfixExpression() ast.Expression {
	// postfix_expression
	//   postfix_expression LBRACKET integer_expression RBRACKET
	//   postfix_expression DOT FIELD_SELECTION
	//   postfix_expression INC|DEC
	//   primary_expression
	//     variable_identifier
	//     INTCONSTANT|UINTCONSTANT|FLOATCONSTANT|BOOLCONSTANT
	//     LPAREN expression RPAREN
	//   function_call
	expr := b.parsePrimaryOrFunctionCallExpression()
LOOP:
	for {
		switch {
		case b.peekToken() == nil:
			break LOOP
		case b.peekToken().String() == ast.UoPostinc.String():
			expr = &ast.UnaryExpr{Op: ast.UoPostinc, Expr: expr, OpCst: b.nextCst()}
		case b.peekToken().String() == ast.UoPostdec.String():
			expr = &ast.UnaryExpr{Op: ast.UoPostdec, Expr: expr, OpCst: b.nextCst()}
		case b.peekToken() == pp.OpDot:
			dotCst := b.nextCst()
			if _, ok := b.peekToken().(pp.Identifier); !ok {
				b.unexpectedTokenError("identifier")
				return expr
			}
			field := b.peekToken().String()
			expr = &ast.DotExpr{
				Expr:     expr,
				Field:    field,
				DotCst:   dotCst,
				FieldCst: b.nextCst(),
			}
			if b.peekToken() == pp.OpLParen {
				expr = b.parseFunctionCall(expr)
			}
		case b.peekToken() == pp.OpLBracket:
			lbracketCst := b.nextCst()
			index := b.parseExpression()
			rbracketCst := b.expectSkip(pp.OpRBracket, pp.OpRBracket).Cst
			expr = &ast.IndexExpr{expr, index, lbracketCst, rbracketCst, nil}
		default:
			break LOOP
		}
	}
	return expr
}

func (b *builder) parsePrimaryOrFunctionCallExpression() ast.Expression {
	// primary_expression
	//   variable_identifier
	//   INTCONSTANT|UINTCONSTANT|FLOATCONSTANT|BOOLCONSTANT
	//   LPAREN expression RPAREN
	// function_call
	//   function_call_generic

	//  The expression leafs. These are easy, though lengthy. Just pick out the individual
	//  constants, variable references and function calls.
	switch b.peekToken() {
	case pp.OpLParen:
		lparenCst := b.nextCst()
		var expr ast.Expression
		if b.language == ast.LangPreprocessor {
			expr = b.parseLogicalOrExpression()
		} else {
			expr = b.parseExpression()
		}
		rparenCst := b.expect(pp.OpRParen).Cst
		return &ast.ParenExpr{expr, lparenCst, rparenCst}
	}

	switch t := b.peekToken().(type) {
	case ast.IntValue, ast.UintValue:
		return &ast.ConstantExpr{t.(ast.Value), b.nextCst()}
	}

	if b.language != ast.LangPreprocessor {
		return b.parseShaderPrimaryExpression()
	} else {
		return b.parsePreprocessorPrimaryExpression()
	}
}

func (b *builder) parsePreprocessorPrimaryExpression() ast.Expression {
	switch t := b.peekToken().(type) {
	case pp.Identifier, ast.Type, pp.Keyword:
		cst := b.nextCst()
		return &ast.VarRefExpr{&ast.VariableSym{SymName: t.String()}, cst}
	}
	b.unexpectedTokenError("constant", pp.OpLParen)
	return nil
}

func (b *builder) parseShaderPrimaryExpression() ast.Expression {
	switch b.peekToken() {
	case pp.KwFalse:
		return &ast.ConstantExpr{ast.BoolValue(false), b.nextCst()}
	case pp.KwTrue:
		return &ast.ConstantExpr{ast.BoolValue(true), b.nextCst()}
	}

	info := b.pp.Peek()
	if pp.IsPrecision(info.Token) || b.getType(info) != nil {
		declType := b.parseTypeSpecifier()
		return b.parseFunctionCall(&ast.TypeConversionExpr{declType, nil})
	}

	switch t := b.peekToken().(type) {
	case ast.FloatValue:
		return &ast.ConstantExpr{t, b.nextCst()}
	case pp.Identifier:
		cst := b.nextCst()
		sym := b.currentScope.GetValueSymbol(b, t.String())
		paren := b.peekToken() == pp.OpLParen
		_, isFun := sym.(ast.Function)
		if sym == nil {
			b.Errorf("Undeclared identifier '%s'.", t)
		} else {
			if paren != isFun {
				var s string
				if isFun {
					s = ""
				} else {
					s = " not"
				}
				b.Errorf("Syntax error: '%s' is%s a function.", sym.Name(), s)
			}
		}
		ref := &ast.VarRefExpr{sym, cst}
		if paren {
			return b.parseFunctionCall(ref)
		}
		return ref
	}

	b.unexpectedTokenError("constant", "identifier", pp.OpLParen, "type constructor")
	return nil
}

func (b *builder) parseExpression() ast.Expression {
	left := b.parseAssignmentExpression()
	if b.peekToken() == ast.BoComma {
		cst := b.nextCst()
		right := b.parseExpression()
		return &ast.BinaryExpr{Left: left, Op: ast.BoComma, Right: right, OpCst: cst}
	}
	return left
}
