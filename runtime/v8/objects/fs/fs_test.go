package fs

import (
	"encoding/base64"
	"fmt"
	iofs "io/fs"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/fs"
	"github.com/yaoapp/gou/fs/system"
	"github.com/yaoapp/gou/query"
	"github.com/yaoapp/gou/query/gou"
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/xun/dbal/schema"
	"rogchap.com/v8go"
)

func TestFSObjectReadFile(t *testing.T) {
	testFsClear(t)
	f := testFsFiles(t)
	data := testFsMakeF1(t)

	initTestEngine()
	iso := v8go.NewIsolate()
	defer iso.Dispose()

	fs := &Object{}
	global := v8go.NewObjectTemplate(iso)
	global.Set("FS", fs.ExportFunction(iso))

	ctx := v8go.NewContext(iso, global)
	defer ctx.Close()

	// ReadFile
	v, err := ctx.RunScript(fmt.Sprintf(`
	function ReadFile() {
		var fs = new FS("system")
		var data = fs.ReadFile("%s");
		return data
	}
	ReadFile()
	`, f["F1"]), "")

	if err != nil {
		t.Fatal(err)
	}

	res, err := bridge.GoValue(v, ctx)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, string(data), res)

	// ReadFileBuffer
	v, err = ctx.RunScript(fmt.Sprintf(`
		function ReadFileBuffer() {
			var fs = new FS("system")
			var data = fs.ReadFileBuffer("%s");
			return data
		}
		ReadFileBuffer()
		`, f["F1"]), "")

	assert.True(t, v.IsUint8Array())

	res, err = bridge.GoValue(v, ctx)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, data, res)

	// ReadFileBuffer
	v, err = ctx.RunScript(fmt.Sprintf(`
	function ReadFileBuffer() {
		var fs = new FS("system")
		var data = fs.ReadFileBuffer("%s");
		return data
	}
	ReadFileBuffer()
	`, f["F1"]), "")

	if err != nil {
		t.Fatal(err)
	}

	res, err = bridge.GoValue(v, ctx)
	assert.True(t, v.IsUint8Array())
	assert.Equal(t, data, res)

	// ReadCloser
	v, err = ctx.RunScript(fmt.Sprintf(`
		function ReadCloser() {
			var fs = new FS("system")
			var hd = fs.ReadCloser("%s");
			return hd
		}
		ReadCloser()
		`, f["F1"]), "")

	if err != nil {
		t.Fatal(err)
	}

	res, err = bridge.GoValue(v, ctx)
	readCloser, ok := res.(*os.File)
	if !ok {
		t.Fatal("res is not *os.File")
	}

	err = readCloser.Close()
	assert.Nil(t, err)

	// Download
	v, err = ctx.RunScript(fmt.Sprintf(`
		function Download() {
			var fs = new FS("system")
			var hd = fs.Download("%s");
			return hd
		}
		Download()
		`, f["F1"]), "")

	if err != nil {
		t.Fatal(err)
	}

	res, err = bridge.GoValue(v, ctx)
	download, ok := res.(map[string]interface{})
	if !ok {
		t.Fatal("res is not not map[string]interface{}")
	}

	readCloser, ok = download["content"].(*os.File)
	if !ok {
		t.Fatal("res is not *os.File")
	}
	err = readCloser.Close()
	assert.Nil(t, err)

	mimetype, ok := download["type"].(string)
	if !ok {
		t.Fatal("res is not string")
	}

	assert.Equal(t, "text/plain; charset=utf-8", mimetype)
}

func TestFSObjectRootFs(t *testing.T) {
	testFsClear(t)
	f := testFsFiles(t)
	data := testFsMakeF1(t)

	initTestEngine()
	iso := v8go.NewIsolate()
	defer iso.Dispose()

	fs := &Object{}
	global := v8go.NewObjectTemplate(iso)
	global.Set("FS", fs.ExportFunction(iso))

	ctx := v8go.NewContext(iso, global)
	defer ctx.Close()

	goData := map[string]interface{}{"ROOT": true}
	jsData, err := bridge.JsValue(ctx, goData)
	if err != nil {
		t.Fatal(err)
	}

	err = ctx.Global().Set("__yao_data", jsData)
	if err != nil {
		t.Fatal(err)
	}

	// WriteFile
	v, err := ctx.RunScript(fmt.Sprintf(`
	function WriteFile() {
		var fs = new FS("dsl")
		return fs.WriteFile("%s", "%s", 0644);
	}
	WriteFile()
	`, f["F1"], string(data)), "")

	if err != nil {
		t.Fatal(err)
	}

	res, err := bridge.GoValue(v, ctx)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, len(data), res)
}

func TestFSObjectRootFsError(t *testing.T) {
	testFsClear(t)
	f := testFsFiles(t)
	data := testFsMakeF1(t)

	initTestEngine()
	iso := v8go.NewIsolate()
	defer iso.Dispose()

	fs := &Object{}
	global := v8go.NewObjectTemplate(iso)
	global.Set("FS", fs.ExportFunction(iso))

	ctx := v8go.NewContext(iso, global)
	defer ctx.Close()

	// WriteFile
	_, err := ctx.RunScript(fmt.Sprintf(`
	function WriteFile() {
		var fs = new FS("dsl")
		return fs.WriteFile("%s", "%s", 0644);
	}
	WriteFile()
	`, f["F1"], string(data)), "")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "dsl does not loaded")

	// WriteFile SU root
	v, err := ctx.RunScript(fmt.Sprintf(`
	__yao_data = { "ROOT": true }
	function WriteFile() {
		var fs = new FS("dsl")
		return fs.WriteFile("%s", "%s", 0644);
	}
	WriteFile()
	`, f["F1"], string(data)), "")

	if err != nil {
		t.Fatal(err)
	}

	res, err := bridge.GoValue(v, ctx)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, len(data), res)

}

func TestFSObjectWriteFile(t *testing.T) {
	testFsClear(t)
	f := testFsFiles(t)
	data := testFsMakeF1(t)

	initTestEngine()
	iso := v8go.NewIsolate()
	defer iso.Dispose()

	fs := &Object{}
	global := v8go.NewObjectTemplate(iso)
	global.Set("FS", fs.ExportFunction(iso))

	ctx := v8go.NewContext(iso, global)
	defer ctx.Close()

	// WriteFile
	v, err := ctx.RunScript(fmt.Sprintf(`
	function WriteFile() {
		var fs = new FS("system")
		return fs.WriteFile("%s", "%s", 0644);
	}
	WriteFile()
	`, f["F1"], string(data)), "")

	if err != nil {
		t.Fatal(err)
	}

	res, err := bridge.GoValue(v, ctx)
	if err != nil {
		t.Fatal(err)
	}

	// WriteFileBuffer
	v, err = ctx.RunScript(fmt.Sprintf(`
		function WriteFileBuffer() {
			var fs = new FS("system")
			var data = fs.ReadFileBuffer("%s")
			return fs.WriteFileBuffer("%s", data, 0644);
		}
		WriteFileBuffer()
		`, f["F1"], f["F2"]), "")

	if err != nil {
		t.Fatal(err)
	}

	res, err = bridge.GoValue(v, ctx)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, len(data), res)

	// WriteFileBase64
	v, err = ctx.RunScript(fmt.Sprintf(`
	function WriteFileBase64() {
		var fs = new FS("system")
		var data = fs.ReadFileBase64("%s")
		return fs.WriteFileBase64("%s", data, 0644);
	}
	WriteFileBase64()
	`, f["F1"], f["F2"]), "")

	if err != nil {
		t.Fatal(err)
	}

	res, err = bridge.GoValue(v, ctx)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, len(data), res)
}

func TestAppendFile(t *testing.T) {

	testFsClear(t)
	f := testFsFiles(t)
	data := testFsMakeF1(t)

	initTestEngine()
	iso := v8go.NewIsolate()
	defer iso.Dispose()

	fsobj := &Object{}
	global := v8go.NewObjectTemplate(iso)
	global.Set("FS", fsobj.ExportFunction(iso))

	ctx := v8go.NewContext(iso, global)
	defer ctx.Close()

	// AppendFile
	v, err := ctx.RunScript(fmt.Sprintf(`
	function AppendFile() {
		var fs = new FS("system")
		const name = "%s"
		const data = "%s"
		fs.Remove(name)
		fs.AppendFile(name, data, 0644);

		const base64 = fs.ReadFileBase64(name)
		const buffer = fs.ReadFileBuffer(name)
		fs.AppendBuffer(name, buffer, 0644);
		fs.AppendBase64(name, base64, 0644);
		return fs.ReadFileBase64(name)
	}
	AppendFile()
	`, f["F1"], string(data)), "")

	if err != nil {
		t.Fatal(err)
	}

	res, err := bridge.GoValue(v, ctx)
	if err != nil {
		t.Fatal(err)
	}
	str, ok := res.(string)
	if !ok {
		t.Fatal("res is not string")
	}
	content, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		t.Fatal(err)
	}
	contentShouldBe := append(data, append(data, data...)...)
	assert.Equal(t, contentShouldBe, content)
}

func TestFSObjectInsertFile(t *testing.T) {

	testFsClear(t)
	f := testFsFiles(t)
	data := testFsMakeF1(t)

	initTestEngine()
	iso := v8go.NewIsolate()
	defer iso.Dispose()

	fsobj := &Object{}
	global := v8go.NewObjectTemplate(iso)
	global.Set("FS", fsobj.ExportFunction(iso))

	ctx := v8go.NewContext(iso, global)
	defer ctx.Close()

	// InsertFile
	v, err := ctx.RunScript(fmt.Sprintf(`
	function InsertFile() {
		var fs = new FS("system")
		const name = "%s"
		const data = "%s"
		fs.Remove(name)
		
		fs.InsertFile(name, 0, data, 0644);
		const base64 = fs.ReadFileBase64(name)
		const buffer = fs.ReadFileBuffer(name)
		fs.InsertBuffer(name, 2, buffer, 0644);
		fs.InsertBase64(name, 2, base64, 0644);
		return fs.ReadFileBase64(name)
	}
	InsertFile()
	`, f["F1"], string(data)), "")

	if err != nil {
		t.Fatal(err)
	}

	res, err := bridge.GoValue(v, ctx)
	if err != nil {
		t.Fatal(err)
	}
	str, ok := res.(string)
	if !ok {
		t.Fatal("res is not string")
	}
	content, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		t.Fatal(err)
	}

	contentShouldBe := append(data[:2], append(data, data[2:]...)...)
	contentShouldBe = append(contentShouldBe[:2], append(data, contentShouldBe[2:]...)...)
	assert.Equal(t, contentShouldBe, content)
}

func TestFSObjectExistRemove(t *testing.T) {

	testFsClear(t)
	testFsMakeF1(t)
	testFsMakeD1D2F1(t)
	f := testFsFiles(t)

	initTestEngine()
	iso := v8go.NewIsolate()
	defer iso.Dispose()

	fs := &Object{}
	global := v8go.NewObjectTemplate(iso)
	global.Set("FS", fs.ExportFunction(iso))

	ctx := v8go.NewContext(iso, global)
	defer ctx.Close()

	// WriteFile
	v, err := ctx.RunScript(fmt.Sprintf(`
	function ExistRemove() {
		var res = {}
		var fs = new FS("system")
		res["ExistsTrue"] = fs.Exists("%s");
		res["ExistsFalse"] = fs.Exists("%s");
		res["IsDirTrue"] = fs.IsDir("%s");
		res["IsDirFalse"] = fs.IsDir("%s");
		res["IsFileTrue"] = fs.IsFile("%s");
		res["IsFileFalse"] = fs.IsFile("%s");
		res["Remove"] = fs.Remove("%s");
		res["RemoveNotExists"] = fs.Remove("%s");
		try {
			fs.Remove("%s");
		} catch( err ) {
			res["RemoveError"] = err.message
		}
		res["RemoveAll"] = fs.RemoveAll("%s");
		res["RemoveAllNotExists"] = fs.RemoveAll("%s");
		return res
	}
	ExistRemove()
	`,
		f["F1"], f["F2"], f["D1"], f["F1"], f["F1"], f["D1"], f["F1"], f["F2"], f["D1"], f["D1"], f["D1_D2"],
	), "")

	if err != nil {
		t.Fatal(err)
	}

	res, err := bridge.GoValue(v, ctx)
	if err != nil {
		t.Fatal(err)
	}

	retval := res.(map[string]interface{})
	assert.Equal(t, true, retval["ExistsTrue"])
	assert.Equal(t, false, retval["ExistsFalse"])
	assert.Equal(t, true, retval["IsDirTrue"])
	assert.Equal(t, false, retval["IsDirFalse"])
	assert.Equal(t, true, retval["IsFileTrue"])
	assert.Equal(t, false, retval["IsFileFalse"])
	assert.Nil(t, retval["Remove"])
	assert.Nil(t, retval["RemoveNotExists"])
	assert.Contains(t, retval["RemoveError"], "directory not empty")
	assert.Nil(t, retval["RemoveAll"])
	assert.Nil(t, retval["RemoveAllNotExists"])
}

func TestFSObjectFileInfo(t *testing.T) {

	testFsClear(t)
	data := testFsMakeF1(t)
	testFsMakeD1D2F1(t)
	f := testFsFiles(t)

	initTestEngine()
	iso := v8go.NewIsolate()
	defer iso.Dispose()

	fs := &Object{}
	global := v8go.NewObjectTemplate(iso)
	global.Set("FS", fs.ExportFunction(iso))

	ctx := v8go.NewContext(iso, global)
	defer ctx.Close()

	// WriteFile
	v, err := ctx.RunScript(fmt.Sprintf(`
	function FileInfo() {
		var res = {}
		var fs = new FS("system")
		res["BaseName"] = fs.BaseName("%s");
		res["DirName"] = fs.DirName("%s");
		res["ExtName"] = fs.ExtName("%s");
		res["MimeType"] = fs.MimeType("%s");
		res["Size"] = fs.Size("%s");
		res["ModTime"] = fs.ModTime("%s");
		res["Mode"] = fs.Mode("%s");
		res["Chmod"] = fs.Chmod("%s", 0755);
		res["ModeAfter"] = fs.Mode("%s");
		return res
	}
	FileInfo()
	`,
		f["F1"], f["F1"], f["F1"], f["F1"], f["F1"], f["F1"], f["F1"], f["F1"], f["F1"],
	), "")

	if err != nil {
		t.Fatal(err)
	}

	res, err := bridge.GoValue(v, ctx)
	if err != nil {
		t.Fatal(err)
	}

	ret := res.(map[string]interface{})
	assert.Equal(t, "f1.file", ret["BaseName"])
	assert.Equal(t, f["root"], ret["DirName"])
	assert.Equal(t, "file", ret["ExtName"])
	assert.Equal(t, "text/plain; charset=utf-8", ret["MimeType"])
	assert.Equal(t, len(data), int(ret["Size"].(float64)))
	assert.Equal(t, iofs.FileMode(0644), iofs.FileMode(int(ret["Mode"].(float64))))
	assert.Equal(t, iofs.FileMode(0755), iofs.FileMode(int(ret["ModeAfter"].(float64))))
	assert.Equal(t, true, int(time.Now().Unix()) >= int(ret["ModTime"].(float64)))
	assert.Nil(t, ret["Chmod"])

}

func TestFSObjectMove(t *testing.T) {

	testFsClear(t)
	testFsMakeData(t)
	f := testFsFiles(t)

	initTestEngine()
	iso := v8go.NewIsolate()
	defer iso.Dispose()

	fsobj := &Object{}
	global := v8go.NewObjectTemplate(iso)
	global.Set("FS", fsobj.ExportFunction(iso))

	ctx := v8go.NewContext(iso, global)
	defer ctx.Close()

	// WriteFile
	v, err := ctx.RunScript(fmt.Sprintf(`
	function Move() {
		var res = {}
		var fs = new FS("system")
		return fs.Move("%s", "%s")
	
	}
	Move()
	`, f["D1_D2"], f["D2"]), "")

	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, v.IsNull())
	stor := fs.FileSystems["system"]
	dirs, err := fs.ReadDir(stor, f["D2"], true)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(dirs))
	checkFileNotExists(t, f["D1_D2"])
	checkFileExists(t, f["D2_F1"])
	checkFileExists(t, f["D2_F2"])
}

func TestFSObjectMoveAppend(t *testing.T) {
	testFsClear(t)
	testFsMakeData(t)
	f := testFsFiles(t)

	initTestEngine()
	iso := v8go.NewIsolate()
	defer iso.Dispose()

	fsobj := &Object{}
	global := v8go.NewObjectTemplate(iso)
	global.Set("FS", fsobj.ExportFunction(iso))

	ctx := v8go.NewContext(iso, global)
	defer ctx.Close()

	// WriteFile
	data := testFsMakeF1(t)
	v, err := ctx.RunScript(fmt.Sprintf(`
	function Move() {
		var res = {}
		const src = "%s"
		const dst = "%s"
		const data = "%s"
		var fs = new FS("system")
		fs.WriteFile(src, data, 0644)
		fs.WriteFile(dst, data, 0644)
		return fs.MoveAppend(src, dst)
	
	}
	Move()
	`, f["D1_D2_F1"], f["D1_D2_F2"], data), "")

	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, v.IsNull())

	// Check file
	content, err := fs.FileSystems["system"].ReadFile(f["D1_D2_F2"])
	contentDataShouldBe := append(data, data...)
	assert.Equal(t, contentDataShouldBe, content)
}

func TestFSObjectMoveAppendError(t *testing.T) {

	testFsClear(t)
	testFsMakeData(t)
	f := testFsFiles(t)

	initTestEngine()
	iso := v8go.NewIsolate()
	defer iso.Dispose()

	fsobj := &Object{}
	global := v8go.NewObjectTemplate(iso)
	global.Set("FS", fsobj.ExportFunction(iso))

	ctx := v8go.NewContext(iso, global)
	defer ctx.Close()

	// WriteFile
	v, err := ctx.RunScript(fmt.Sprintf(`
	function Move() {
		var res = {}
		const src = "%s"
		const dst = "%s"
		var fs = new FS("system")
		try {
			return fs.MoveAppend(src, dst)
		}catch(err) {
			return err.message
		}
	}
	Move()
	`, f["D1_D2"], f["D1_D2"]), "")

	if err != nil {
		t.Fatal(err)
	}
	assert.Contains(t, v.String(), "is not a file")
}

func TestFSObjectMoveInsertError(t *testing.T) {

	testFsClear(t)
	testFsMakeData(t)
	f := testFsFiles(t)

	initTestEngine()
	iso := v8go.NewIsolate()
	defer iso.Dispose()

	fsobj := &Object{}
	global := v8go.NewObjectTemplate(iso)
	global.Set("FS", fsobj.ExportFunction(iso))

	ctx := v8go.NewContext(iso, global)
	defer ctx.Close()

	// WriteFile
	v, err := ctx.RunScript(fmt.Sprintf(`
	function Move() {
		var res = {}
		const src = "%s"
		const dst = "%s"
		var fs = new FS("system")
		try {
			return fs.MoveInsert(src, dst, 2)
		}catch(err) {
			return err.message
		}
	}
	Move()
	`, f["D1_D2"], f["D1_D2"]), "")

	if err != nil {
		t.Fatal(err)
	}
	assert.Contains(t, v.String(), "is not a file")
}

func TestFSObjectMoveInsert(t *testing.T) {
	testFsClear(t)
	testFsMakeData(t)
	f := testFsFiles(t)

	initTestEngine()
	iso := v8go.NewIsolate()
	defer iso.Dispose()

	fsobj := &Object{}
	global := v8go.NewObjectTemplate(iso)
	global.Set("FS", fsobj.ExportFunction(iso))

	ctx := v8go.NewContext(iso, global)
	defer ctx.Close()

	// WriteFile
	data := testFsMakeF1(t)
	v, err := ctx.RunScript(fmt.Sprintf(`
	function Move() {
		var res = {}
		const src = "%s"
		const dst = "%s"
		const data = "%s"
		var fs = new FS("system")
		fs.WriteFile(src, data, 0644)
		fs.WriteFile(dst, data, 0644)
		return fs.MoveInsert(src, dst, 2)
	
	}
	Move()
	`, f["D1_D2_F1"], f["D1_D2_F2"], data), "")

	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, v.IsNull())

	// Check file
	content, err := fs.FileSystems["system"].ReadFile(f["D1_D2_F2"])
	contentDataShouldBe := append(data[:2], append(data, data[2:]...)...)
	assert.Equal(t, contentDataShouldBe, content)
}

func TestFSObjectZip(t *testing.T) {

	testFsClear(t)
	testFsMakeData(t)
	f := testFsFiles(t)

	initTestEngine()
	iso := v8go.NewIsolate()
	defer iso.Dispose()

	fsobj := &Object{}
	global := v8go.NewObjectTemplate(iso)
	global.Set("FS", fsobj.ExportFunction(iso))

	ctx := v8go.NewContext(iso, global)
	defer ctx.Close()

	zipfile := filepath.Join(os.Getenv("GOU_TEST_APP_ROOT"), "data", "test.zip")
	unzipdir := filepath.Join(os.Getenv("GOU_TEST_APP_ROOT"), "data", "test")

	v, err := ctx.RunScript(fmt.Sprintf(`
	function Zip() {
		var res = {}
		var fs = new FS("system")
		return fs.Zip("%s", "%s")
	
	}
	Zip()
	`, f["D1_D2"], zipfile), "")

	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, v.IsNull())

	v, err = ctx.RunScript(fmt.Sprintf(`
	function UnZip() {
		var res = {}
		var fs = new FS("system")
		return fs.Unzip("%s", "%s")
	
	}
	UnZip()
	`, zipfile, unzipdir), "")

	if err != nil {
		t.Fatal(err)
	}

	files, err := bridge.GoValue(v, ctx)
	assert.Len(t, files, 2)
}

func TestFSObjectCopy(t *testing.T) {

	testFsClear(t)
	testFsMakeData(t)
	f := testFsFiles(t)

	initTestEngine()
	iso := v8go.NewIsolate()
	defer iso.Dispose()

	fsobj := &Object{}
	global := v8go.NewObjectTemplate(iso)
	global.Set("FS", fsobj.ExportFunction(iso))

	ctx := v8go.NewContext(iso, global)
	defer ctx.Close()

	// WriteFile
	v, err := ctx.RunScript(fmt.Sprintf(`
	function Copy() {
		var res = {}
		var fs = new FS("system")
		return fs.Copy("%s", "%s")
	
	}
	Copy()
	`, f["D1_D2"], f["D2"]), "")

	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, v.IsNull())
	stor := fs.FileSystems["system"]
	dirs, err := fs.ReadDir(stor, f["D2"], true)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(dirs))
	checkFileExists(t, f["D1_D2"])
	checkFileExists(t, f["D2_F1"])
	checkFileExists(t, f["D2_F2"])
}

func TestFSObjectDir(t *testing.T) {
	testFsClear(t)
	f := testFsFiles(t)
	initTestEngine()
	iso := v8go.NewIsolate()
	defer iso.Dispose()

	fs := &Object{}
	global := v8go.NewObjectTemplate(iso)
	global.Set("FS", fs.ExportFunction(iso))

	ctx := v8go.NewContext(iso, global)
	defer ctx.Close()

	// DirTest
	v, err := ctx.RunScript(fmt.Sprintf(`
	function DirTest() {
		var fs = new FS("system");
		fs.Mkdir("%s");
		fs.MkdirAll("%s");
		fs.MkdirTemp()
		fs.MkdirTemp("%s")
		fs.MkdirTemp("%s", "*-logs")
		return fs.ReadDir("%s", true)
	}
	DirTest()
	`, f["D2"], f["D1_D2"], f["D1"], f["D1"], f["root"]), "")

	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, v.IsArray())
	res, err := bridge.GoValue(v, ctx)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 5, len(res.([]interface{})))
}

func TestFSObjectGlob(t *testing.T) {
	testFsClear(t)
	f := testFsFiles(t)
	initTestEngine()
	iso := v8go.NewIsolate()
	defer iso.Dispose()

	fs := &Object{}
	global := v8go.NewObjectTemplate(iso)
	global.Set("FS", fs.ExportFunction(iso))

	ctx := v8go.NewContext(iso, global)
	defer ctx.Close()

	// GlobTest
	v, err := ctx.RunScript(fmt.Sprintf(`
	function GlobTest() {
		var fs = new FS("system");
		fs.Mkdir("%s");
		fs.MkdirAll("%s");
		fs.MkdirTemp()
		fs.MkdirTemp("%s")
		fs.MkdirTemp("%s", "*-logs")
		return fs.Glob("%s/*/*")
	}
	GlobTest()
	`, f["D2"], f["D1_D2"], f["D1"], f["D1"], f["root"]), "")

	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, v.IsArray())
	res, err := bridge.GoValue(v, ctx)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 3, len(res.([]interface{})))
}

func testFsMakeF1(t *testing.T) []byte {
	data := testFsData(t)
	f := testFsFiles(t)

	// Write
	_, err := fs.WriteFile(fs.FileSystems["system"], f["F1"], data, 0644)
	if err != nil {
		t.Fatal(err)
	}

	return data
}

func checkFileExists(t assert.TestingT, path string) {
	stor := fs.FileSystems["system"]
	exist, _ := fs.Exists(stor, path)
	assert.True(t, exist)
}

func checkFileNotExists(t assert.TestingT, path string) {
	stor := fs.FileSystems["system"]
	exist, _ := fs.Exists(stor, path)
	assert.False(t, exist)
}
func testFsMakeData(t *testing.T) {

	stor := fs.FileSystems["system"]
	f := testFsFiles(t)
	data := testFsData(t)

	err := fs.MkdirAll(stor, f["D1_D2"], uint32(os.ModePerm))
	if err != nil {
		t.Fatal(err)
	}

	// Write
	_, err = fs.WriteFile(stor, f["D1_F1"], data, 0644)
	if err != nil {
		t.Fatal(err)
	}

	_, err = fs.WriteFile(stor, f["D1_F2"], data, 0644)
	if err != nil {
		t.Fatal(err)
	}

	_, err = fs.WriteFile(stor, f["D1_D2_F1"], data, 0644)
	if err != nil {
		t.Fatal(err)
	}

	_, err = fs.WriteFile(stor, f["D1_D2_F2"], data, 0644)
	if err != nil {
		t.Fatal(err)
	}
}

func testFsData(t *testing.T) []byte {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	rand.Seed(time.Now().UnixNano())
	n := rand.Intn(10) + 1
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return []byte(fmt.Sprintf("HELLO WORLD %s", string(b)))
}

func testFsFiles(t *testing.T) map[string]string {

	root := filepath.Join(os.Getenv("GOU_TEST_APP_ROOT"), "data")
	return map[string]string{
		"root":     root,
		"F1":       filepath.Join(root, "f1.file"),
		"F2":       filepath.Join(root, "f2.file"),
		"F3":       filepath.Join(root, "f3.js"),
		"D1_F1":    filepath.Join(root, "d1", "f1.file"),
		"D1_F2":    filepath.Join(root, "d1", "f2.file"),
		"D2_F1":    filepath.Join(root, "d2", "f1.file"),
		"D2_F2":    filepath.Join(root, "d2", "f2.file"),
		"D1_D2_F1": filepath.Join(root, "d1", "d2", "f1.file"),
		"D1_D2_F2": filepath.Join(root, "d1", "d2", "f2.file"),
		"D1":       filepath.Join(root, "d1"),
		"D2":       filepath.Join(root, "d2"),
		"D1_D2":    filepath.Join(root, "d1", "d2"),
	}

}

func testFsClear(t *testing.T) {

	fs.Register("system", system.New())
	fs.RootRegister("dsl", system.New())

	stor := fs.FileSystems["system"]
	root := filepath.Join(os.Getenv("GOU_TEST_APP_ROOT"), "data")
	err := os.RemoveAll(root)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	err = fs.MkdirAll(stor, root, uint32(os.ModePerm))
	if err != nil {
		t.Fatal(err)
	}
}

func testFsMakeD1D2F1(t *testing.T) []byte {
	data := testFsData(t)
	f := testFsFiles(t)
	_, err := fs.WriteFile(fs.FileSystems["system"], f["D1_D2_F1"], data, 0644)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func initTestEngine() {

	if capsule.Global == nil {

		var TestDriver = os.Getenv("GOU_TEST_DB_DRIVER")
		var TestDSN = os.Getenv("GOU_TEST_DSN")

		// Connect DB
		switch TestDriver {
		case "sqlite3":
			capsule.AddConn("primary", "sqlite3", TestDSN).SetAsGlobal()
			break
		default:
			capsule.AddConn("primary", "mysql", TestDSN).SetAsGlobal()
			break
		}
	}

	sch := capsule.Schema()
	sch.MustDropTableIfExists("queryobj_test")
	sch.MustCreateTable("queryobj_test", func(table schema.Blueprint) {
		table.ID("id")
		table.String("name", 20)
	})

	qb := capsule.Query()
	qb.Table("queryobj_test").MustInsert([][]interface{}{
		{1, "Lucy"},
		{2, "Join"},
		{3, "Lily"},
	}, []string{"id", "name"})

	query.Register("query-test", &gou.Query{
		Query: capsule.Query(),
		GetTableName: func(s string) string {
			return s
		},
	})
}
