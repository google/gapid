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

package jdbg_test

import (
	"context"
	"os"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/java/jdbg"
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
		if err := c.ResumeAll(); err != nil {
			log.F(ctx, true, "Failed resume VM. Error: %v", err)
			return -1
		}

		// Wait for java to load the prepare class
		t, err := c.WaitForClassPrepare(ctx, "Calculator")
		if err != nil {
			log.F(ctx, true, "Failed to wait for Calculator prepare. Error: %v", err)
			return -1
		}

		conn, thread = c, t
		return m.Run()
	}))
}

var conn *jdwp.Connection
var thread jdwp.ThreadID

func TestInvokeStringFormat(t *testing.T) {
	assert := assert.To(t)
	err := jdbg.Do(conn, thread, func(j *jdbg.JDbg) error {
		res := j.Class("java/lang/String").Call("format", "%s says '%s' %d times",
			[]interface{}{"bob", "hello world", 5}).Get()
		assert.For("res").That(res).Equals("bob says 'hello world' 5 times")
		return nil
	})
	assert.For("err").That(err).Equals(nil)
}

func TestInvokeStaticMethod(t *testing.T) {
	assert := assert.To(t)
	err := jdbg.Do(conn, thread, func(j *jdbg.JDbg) error {
		res := j.Class("Calculator").Call("Add", 3, 7).Get()
		assert.For("res").That(res).Equals(10)
		return nil
	})
	assert.For("err").That(err).Equals(nil)
}

func TestInvokeMethod(t *testing.T) {
	assert := assert.To(t)
	err := jdbg.Do(conn, thread, func(j *jdbg.JDbg) error {
		calcTy := j.Class("Calculator")

		calc := calcTy.New()
		for _, i := range []int{3, 6, 8} {
			calc.Call("Add", i)
		}

		res := calc.Call("Result").Get()
		assert.For("res").That(res).Equals(3 + 6 + 8)
		return nil
	})
	assert.For("err").That(err).Equals(nil)

}
