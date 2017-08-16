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

package eval

import (
	"context"
	"reflect"
	"regexp"

	"github.com/google/gapid/core/event"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/test/robot/search"
)

type eval func(context.Context, interface{}) interface{}

var (
	boolType     = reflect.TypeOf(true)
	signedType   = reflect.TypeOf(int64(0))
	unsignedType = reflect.TypeOf(uint64(0))
	doubleType   = reflect.TypeOf(float64(0))
	stringType   = reflect.TypeOf("")
)

// Compile takes a search query and a a value type and produces a function that will perform the
// query against values of that type.
func Compile(ctx context.Context, query *search.Query, klass reflect.Type) (event.Predicate, error) {
	m, res, err := compileExpression(ctx, query.Expression, klass)
	if res.Kind() != reflect.Bool {
		return nil, log.Err(ctx, nil, "Expression result not a bool")
	}
	return func(ctx context.Context, value interface{}) bool {
		got := reflect.TypeOf(value)
		if got != klass {
			return false
		}
		return m(ctx, value).(bool)
	}, err
}

// Filter uses a search query and returns a handler that filters events before passing them to
// the output handler.
func Filter(ctx context.Context, query *search.Query, klass reflect.Type, handler event.Handler) event.Handler {
	pred, err := Compile(ctx, query, klass)
	if err != nil {
		return func(ctx context.Context, event interface{}) error {
			return err
		}
	}
	return event.Filter(ctx, pred, handler)
}

func compileExpression(ctx context.Context, expr *search.Expression, t reflect.Type) (eval, reflect.Type, error) {
	if expr == nil {
		return func(context.Context, interface{}) interface{} { return true }, boolType, nil
	}
	switch et := expr.Is.(type) {
	case nil:
		return func(context.Context, interface{}) interface{} { return true }, boolType, nil
	case *search.Expression_Boolean:
		return func(context.Context, interface{}) interface{} { return et.Boolean }, boolType, nil
	case *search.Expression_String_:
		return func(context.Context, interface{}) interface{} { return et.String_ }, stringType, nil
	case *search.Expression_Signed:
		return func(context.Context, interface{}) interface{} { return et.Signed }, signedType, nil
	case *search.Expression_Unsigned:
		return func(context.Context, interface{}) interface{} { return et.Unsigned }, unsignedType, nil
	case *search.Expression_Double:
		return func(context.Context, interface{}) interface{} { return et.Double }, doubleType, nil
	case *search.Expression_Name:
		return compileGetMember(ctx, et.Name, t)
	case *search.Expression_And:
		return compileBinaryBool(ctx, et.And, t, func(lhs, rhs bool) bool { return lhs && rhs })
	case *search.Expression_Or:
		return compileBinaryBool(ctx, et.Or, t, func(lhs, rhs bool) bool { return lhs || rhs })
	case *search.Expression_Equal:
		return compileEqual(ctx, et.Equal, t)
	case *search.Expression_Greater:
		return compileBinaryNumeric(ctx, et.Greater, t,
			func(lhs, rhs int64) bool { return lhs > rhs },
			func(lhs, rhs uint64) bool { return lhs > rhs },
			func(lhs, rhs float64) bool { return lhs > rhs },
		)
	case *search.Expression_GreaterOrEqual:
		return compileBinaryNumeric(ctx, et.GreaterOrEqual, t,
			func(lhs, rhs int64) bool { return lhs >= rhs },
			func(lhs, rhs uint64) bool { return lhs >= rhs },
			func(lhs, rhs float64) bool { return lhs >= rhs },
		)
	case *search.Expression_Subscript:
		return compileSubscript(ctx, et.Subscript, t)
	case *search.Expression_Regex:
		return compileRegex(ctx, et.Regex, t)
	case *search.Expression_Member:
		return compileMember(ctx, et.Member, t)
	case *search.Expression_Not:
		return compileNot(ctx, et.Not, t)
	default:
		return nil, boolType, log.Errf(ctx, nil, "Invalid expression %v", et)
	}
}

func compileBinaryBool(ctx context.Context, expr *search.Binary, t reflect.Type, test func(bool, bool) bool) (eval, reflect.Type, error) {
	lhs, res, err := compileExpression(ctx, expr.Lhs, t)
	if err != nil {
		return nil, boolType, err
	}
	if res.Kind() != reflect.Bool {
		return nil, boolType, log.Err(ctx, nil, "lhs was not bool")
	}
	rhs, res, err := compileExpression(ctx, expr.Rhs, t)
	if err != nil {
		return nil, boolType, err
	}
	if res.Kind() != reflect.Bool {
		return nil, boolType, log.Err(ctx, nil, "rhs was not bool")
	}
	return func(ctx context.Context, value interface{}) interface{} {
		return test(lhs(ctx, value).(bool), rhs(ctx, value).(bool))
	}, boolType, nil
}

func compileEqual(ctx context.Context, expr *search.Binary, t reflect.Type) (eval, reflect.Type, error) {
	lhs, lt, err := compileExpression(ctx, expr.Lhs, t)
	if err != nil {
		return nil, boolType, err
	}
	rhs, rt, err := compileExpression(ctx, expr.Rhs, t)
	if err != nil {
		return nil, boolType, err
	}
	if lt != rt {
		return nil, boolType, log.Err(ctx, nil, "Types to equal do not match")
	}
	return func(ctx context.Context, value interface{}) interface{} {
		return reflect.DeepEqual(lhs(ctx, value), rhs(ctx, value))
	}, boolType, nil
}

func compileBinaryNumeric(ctx context.Context, expr *search.Binary, t reflect.Type,
	testS func(int64, int64) bool,
	testU func(uint64, uint64) bool,
	testD func(float64, float64) bool,
) (eval, reflect.Type, error) {
	lhs, lt, err := compileExpression(ctx, expr.Lhs, t)
	if err != nil {
		return nil, boolType, err
	}
	rhs, rt, err := compileExpression(ctx, expr.Rhs, t)
	if err != nil {
		return nil, boolType, err
	}
	if lt.AssignableTo(signedType) && rt.AssignableTo(signedType) {
		return func(ctx context.Context, value interface{}) interface{} {
			return testS(lhs(ctx, value).(int64), rhs(ctx, value).(int64))
		}, boolType, nil
	}
	if lt.AssignableTo(unsignedType) && rt.AssignableTo(unsignedType) {
		return func(ctx context.Context, value interface{}) interface{} {
			return testU(lhs(ctx, value).(uint64), rhs(ctx, value).(uint64))
		}, boolType, nil
	}
	if lt.AssignableTo(doubleType) && rt.AssignableTo(doubleType) {
		return func(ctx context.Context, value interface{}) interface{} {
			return testD(lhs(ctx, value).(float64), rhs(ctx, value).(float64))
		}, boolType, nil
	}
	return nil, boolType, log.Err(ctx, nil, "no numeric comparison possible")
}

func compileNot(ctx context.Context, expr *search.Expression, t reflect.Type) (eval, reflect.Type, error) {
	rhs, res, err := compileExpression(ctx, expr, t)
	if err != nil {
		return nil, boolType, err
	}
	if res.Kind() != reflect.Bool {
		return nil, boolType, log.Err(ctx, nil, "Not only applies to bool")
	}
	return func(ctx context.Context, value interface{}) interface{} {
		return !rhs(ctx, value).(bool)
	}, boolType, nil
}

func compileSubscript(ctx context.Context, expr *search.Subscript, t reflect.Type) (eval, reflect.Type, error) {
	container, ct, err := compileExpression(ctx, expr.Container, t)
	if err != nil {
		return nil, boolType, err
	}
	key, kt, err := compileExpression(ctx, expr.Key, t)
	if err != nil {
		return nil, boolType, err
	}
	vt := ct.Elem()
	switch ct.Kind() {
	case reflect.Slice:
		if !kt.AssignableTo(signedType) {
			return nil, boolType, log.Err(ctx, nil, "Slices must be indexed by int64")
		}
		zeroValue := reflect.Zero(vt).Interface()
		return func(ctx context.Context, value interface{}) interface{} {
			cv := reflect.ValueOf(container(ctx, value))
			index := (int)(key(ctx, value).(int64))
			if index >= cv.Len() {
				return zeroValue
			}
			return cv.Index(index).Interface()
		}, vt, nil
	case reflect.Map:
		if !kt.AssignableTo(ct.Key()) {
			return nil, boolType, log.Err(ctx, nil, "Map index type does not match")
		}
		return func(ctx context.Context, value interface{}) interface{} {
			return reflect.ValueOf(container(ctx, value)).MapIndex(reflect.ValueOf(key(ctx, value))).Interface()
		}, vt, nil
	default:
		return nil, boolType, log.Err(ctx, nil, "Can only subscript map or slice")
	}
}

func compileRegex(ctx context.Context, expr *search.Regex, t reflect.Type) (eval, reflect.Type, error) {
	val, vt, err := compileExpression(ctx, expr.Value, t)
	if err != nil {
		return nil, boolType, err
	}
	if vt.Kind() != reflect.String {
		return nil, boolType, log.Err(ctx, nil, "Regex only applies to string")
	}
	re, err := regexp.Compile(expr.Pattern)
	if err != nil {
		return nil, boolType, err
	}
	return func(ctx context.Context, value interface{}) interface{} {
		return re.MatchString(val(ctx, value).(string))
	}, boolType, nil
}

func compileMember(ctx context.Context, expr *search.Member, t reflect.Type) (eval, reflect.Type, error) {
	obj, ot, err := compileExpression(ctx, expr.Object, t)
	if err != nil {
		return nil, boolType, err
	}
	get, mt, err := compileGetMember(ctx, expr.Name, ot)
	if err != nil {
		return nil, boolType, err
	}
	return func(ctx context.Context, value interface{}) interface{} {
		return get(ctx, obj(ctx, value))
	}, mt, nil
}

func compileGetMember(ctx context.Context, name string, t reflect.Type) (eval, reflect.Type, error) {
	wasPtr := t.Kind() == reflect.Ptr
	if wasPtr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, boolType, log.Err(ctx, nil, "Field access on non struct type"+t.String())
	}
	field, found := t.FieldByName(name)
	if !found {
		return nil, boolType, log.Errf(ctx, nil, "No field '%v' found in %v", name, t)
	}
	if wasPtr {
		return func(ctx context.Context, value interface{}) interface{} {
			return reflect.ValueOf(value).Elem().FieldByIndex(field.Index).Interface()
		}, field.Type, nil
	}
	return func(ctx context.Context, value interface{}) interface{} {
		return reflect.ValueOf(value).FieldByIndex(field.Index).Interface()
	}, field.Type, nil
}
