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

package jdwp_test

import (
	"context"
	"os"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/java/jdwp"
	"github.com/google/gapid/core/java/jdwp/test"
	"github.com/google/gapid/core/log"
)

var source = map[string]string{
	"main.java": `
public class main {
	static {
		Calculator.register();
	}
	public static void main(String[] args) throws Exception {
		System.out.print("Doing stuff\n");
		Thread.sleep(1000);
	}
}`,
	"Calculator.java": `
public class Calculator {
	public static void register() {}

	private int value = 0;

	public static int Add(int a, int b) { return a + b; }
	public void Add(int a) { value += a; }

	public int Result() { return value; }
}
`,
}

func TestMain(m *testing.M) {
	ctx := context.Background()
	ctx = log.PutHandler(ctx, log.Normal.Handler(log.Std()))
	os.Exit(test.BuildRunAndConnect(ctx, source, func(ctx context.Context, c *jdwp.Connection) int {
		// Wait for java to load the prepare class
		t, err := c.WaitForClassPrepare(ctx, "Calculator")
		if err != nil {
			log.F(ctx, true, "Failed to wait for Calculator prepare. Error: %v", err)
			return -1
		}

		connection, thread = c, t
		return m.Run()
	}))
}

var connection *jdwp.Connection
var thread jdwp.ThreadID

func TestGetClassBySignature(t *testing.T) {
	ctx := log.Testing(t)

	calculator, err := connection.GetClassBySignature("LCalculator;")
	if err != nil {
		log.F(ctx, true, "GetClassesBySignature returned error: %v", err)
	}

	assert.For(ctx, "Calculator kind").That(calculator.Kind).Equals(jdwp.Class)
	assert.For(ctx, "Calculator kind").That(calculator.Signature).Equals("LCalculator;")
}

func TestInvokeStaticMethod(t *testing.T) {
	ctx := log.Testing(t)

	class, err := connection.GetClassBySignature("LCalculator;")
	if err != nil {
		log.F(ctx, true, "GetClassesBySignature returned error: %v", err)
		return
	}

	methods, err := connection.GetMethods(class.TypeID)
	if err != nil {
		log.F(ctx, true, "GetMethods returned error: %v", err)
		return
	}

	add := methods.FindBySignature("Add", "(II)I")
	if add == nil {
		log.F(ctx, true, "Couldn't find method Add: %v", err)
		return
	}

	result, err := connection.InvokeStaticMethod(class.ClassID(), add.ID, thread, jdwp.InvokeSingleThreaded, 3, 7)
	if err != nil {
		log.F(ctx, true, "InvokeStaticMethod returned error: %v", err)
		return
	}
	assert.For(ctx, "Add(3, 7)").That(result.Result).Equals(10)
}

func TestInvokeMethod(t *testing.T) {
	ctx := log.Testing(t)

	class, err := connection.GetClassBySignature("LCalculator;")
	if err != nil {
		log.F(ctx, true, "GetClassesBySignature returned error: %v", err)
		return
	}

	methods, err := connection.GetMethods(class.TypeID)
	if err != nil {
		log.F(ctx, true, "GetMethods returned error: %v", err)
		return
	}

	constructor := methods.FindBySignature("<init>", "()V")
	if constructor == nil {
		log.F(ctx, true, "Couldn't find constructor: %v", err)
		log.I(ctx, "Available methods:\n%v", methods)
		return
	}

	instance, err := connection.NewInstance(class.ClassID(), constructor.ID, thread, jdwp.InvokeSingleThreaded)
	if err != nil {
		log.F(ctx, true, "NewInstance returned error: %v", err)
		return
	}

	add := methods.FindBySignature("Add", "(I)V")
	if add == nil {
		log.F(ctx, true, "Couldn't find method Add: %v", err)
		log.I(ctx, "Available methods:\n%v", methods)
		return
	}

	for _, i := range []int{3, 6, 8} {
		_, err = connection.InvokeMethod(
			instance.Result.Object,
			class.ClassID(),
			add.ID,
			thread,
			jdwp.InvokeSingleThreaded,
			i)
		if err != nil {
			log.F(ctx, true, "InvokeMethod returned error: %v", err)
			return
		}
	}

	resultf := methods.FindBySignature("Result", "()I")
	if resultf == nil {
		log.F(ctx, true, "Couldn't find method Result: %v", err)
		log.I(ctx, "Available methods:\n%v", methods)
		return
	}

	result, err := connection.InvokeMethod(
		instance.Result.Object,
		class.ClassID(),
		resultf.ID,
		thread,
		jdwp.InvokeSingleThreaded)
	if err != nil {
		log.F(ctx, true, "InvokeMethod returned error: %v", err)
		return
	}
	assert.For(ctx, "Result").That(result.Result).Equals(3 + 6 + 8)
}
