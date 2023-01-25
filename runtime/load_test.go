package runtime

import (
	"context"
	"fmt"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRuntimeRootLoad(t *testing.T) {

	runtime, TestScriptRoot := prepare(t)
	ctx := context.Background()
	err := runtime.RootLoad(path.Join(TestScriptRoot, "test.js"), "test")
	assert.Nil(t, err)

	isRoot := runtime.New("test", "IsRoot").
		WithGlobal(map[string]interface{}{"foo": "bar"}).
		WithSid("1").
		WithContext(ctx)

	v, err := isRoot.Call()
	assert.NotNil(t, err)

	v, err = isRoot.RootCall()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, true, v.(bool))
}

func TestRuntimeLoad(t *testing.T) {
	runtime, TestScriptRoot := prepare(t)
	ctx := context.Background()
	err := runtime.Load(path.Join(TestScriptRoot, "test.js"), "test")
	assert.Nil(t, err)
	isRoot := runtime.New("test", "IsRoot").
		WithGlobal(map[string]interface{}{"foo": "bar"}).
		WithSid("1").
		WithContext(ctx)

	v, err := isRoot.Call()
	assert.Nil(t, err)
	assert.Equal(t, false, v.(bool))
}

func TestRuntimeExec(t *testing.T) {
	runtime, TestScriptRoot := prepare(t)
	ctx := context.Background()
	err := runtime.Load(path.Join(TestScriptRoot, "test.js"), "test")
	assert.Equal(t, nil, err)
	getArgs := runtime.New("test", "getArgs").
		WithGlobal(map[string]interface{}{"foo": "bar"}).
		WithSid("1").
		WithContext(ctx)

	v, err := getArgs.Call("world", 1, 0.618, []interface{}{"foo", "bar"}, map[string]interface{}{"foo": "bar", "int": 1})
	assert.Nil(t, err)
	fmt.Println(v)

	getArgs = runtime.New("test", "getArgs").
		WithGlobal(map[string]interface{}{"foo": "bar"}).
		WithSid("1").
		WithContext(ctx)
	v, err = getArgs.Call("yao", 2, 1.618, []interface{}{"code", "ping"}, map[string]interface{}{"one": "two", "int": 5})
	assert.Nil(t, err)
	fmt.Println(v)
}

func TestRuntimeExecES6(t *testing.T) {
	runtime, TestScriptRoot := prepare(t)
	ctx := context.Background()
	err := runtime.Load(path.Join(TestScriptRoot, "es6.js"), "es6")
	assert.Equal(t, nil, err)
	now := runtime.New("es6", "now").
		WithGlobal(map[string]interface{}{"foo": "bar"}).
		WithSid("1").
		WithContext(ctx)
	v, err := now.Call("world", 1, 0.618, []interface{}{"foo", "bar"}, map[string]interface{}{"foo": "bar", "int": 1})
	assert.Nil(t, err)
	fmt.Println(v)

	promiseTest := runtime.New("es6", "promiseTest").
		WithGlobal(map[string]interface{}{"foo": "bar"}).
		WithSid("1").
		WithContext(ctx)
	v, err = promiseTest.Call()
	assert.Nil(t, err)
	fmt.Println(v)

	asyncTest := runtime.New("es6", "asyncTest").
		WithGlobal(map[string]interface{}{"foo": "bar"}).
		WithSid("1").
		WithContext(ctx)
	v, err = asyncTest.Call()
	assert.Nil(t, err)
	fmt.Println(v)

	processTest := runtime.New("es6", "processTest").
		WithGlobal(map[string]interface{}{"foo": "bar"}).
		WithSid("1").
		WithContext(ctx)
	v, err = processTest.Call()
	assert.Nil(t, err)
	fmt.Println(v)
}

func prepare(t *testing.T) (*Runtime, string) {
	return LoadRuntime(Option{}), "scripts"
}
