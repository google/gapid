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
	"os"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/context/jot"
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
	os.Exit(test.BuildRunAndConnect(source, func(ctx log.Context, c *jdwp.Connection) int {
		if err := c.ResumeAll(); err != nil {
			jot.Fatal(ctx, err, "Failed resume VM")
			return -1
		}

		// Wait for java to load the prepare class
		t, err := c.WaitForClassPrepare(ctx, "Calculator")
		if err != nil {
			jot.Fatal(ctx, err, "Failed to wait for Calculator prepare")
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
		jot.Fail(ctx, err, "GetClassesBySignature returned error")
	}

	assert.For(ctx, "Calculator kind").That(calculator.Kind).Equals(jdwp.Class)
	assert.For(ctx, "Calculator kind").That(calculator.Signature).Equals("LCalculator;")
}

func TestInvokeStaticMethod(t *testing.T) {
	ctx := log.Testing(t)

	class, err := connection.GetClassBySignature("LCalculator;")
	if err != nil {
		jot.Fail(ctx, err, "GetClassesBySignature returned error")
		return
	}

	methods, err := connection.GetMethods(class.TypeID)
	if err != nil {
		jot.Fail(ctx, err, "GetMethods returned error")
		return
	}

	add := methods.FindBySignature("Add", "(II)I")
	if add == nil {
		jot.Fail(ctx, err, "Couldn't find method Add")
		return
	}

	result, err := connection.InvokeStaticMethod(class.ClassID(), add.ID, thread, jdwp.InvokeSingleThreaded, 3, 7)
	if err != nil {
		jot.Fail(ctx, err, "InvokeStaticMethod returned error")
		return
	}
	assert.For(ctx, "Add(3, 7)").That(result.Result).Equals(10)
}

func TestInvokeMethod(t *testing.T) {
	ctx := log.Testing(t)

	class, err := connection.GetClassBySignature("LCalculator;")
	if err != nil {
		jot.Fail(ctx, err, "GetClassesBySignature returned error")
		return
	}

	methods, err := connection.GetMethods(class.TypeID)
	if err != nil {
		jot.Fail(ctx, err, "GetMethods returned error")
		return
	}

	constructor := methods.FindBySignature("<init>", "()V")
	if constructor == nil {
		jot.Fail(ctx, err, "Couldn't find constructor")
		ctx.Printf("Available methods:\n%v", methods)
		return
	}

	instance, err := connection.NewInstance(class.ClassID(), constructor.ID, thread, jdwp.InvokeSingleThreaded)
	if err != nil {
		jot.Fail(ctx, err, "NewInstance returned error")
		return
	}

	add := methods.FindBySignature("Add", "(I)V")
	if add == nil {
		jot.Fail(ctx, err, "Couldn't find method Add")
		ctx.Printf("Available methods:\n%v", methods)
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
			jot.Fail(ctx, err, "InvokeMethod returned error")
			return
		}
	}

	resultf := methods.FindBySignature("Result", "()I")
	if resultf == nil {
		jot.Fail(ctx, err, "Couldn't find method Result")
		ctx.Printf("Available methods:\n%v", methods)
		return
	}

	result, err := connection.InvokeMethod(
		instance.Result.Object,
		class.ClassID(),
		resultf.ID,
		thread,
		jdwp.InvokeSingleThreaded)
	if err != nil {
		jot.Fail(ctx, err, "InvokeMethod returned error")
		return
	}
	assert.For(ctx, "Result").That(result.Result).Equals(3 + 6 + 8)
}
