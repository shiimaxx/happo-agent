package db

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/heartbeatsjp/happo-agent/util"
	"github.com/stretchr/testify/assert"
)

func copyTestDB(t *testing.T) string {
	t.Helper()

	copied, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal("create temp directory failed:", err)
	}

	files, err := ioutil.ReadDir("./testdata/test.db")
	if err != nil {
		t.Fatal("read testdata directory fialed:", err)
	}

	for _, f := range files {
		src, err := os.Open(path.Join("./testdata/test.db", f.Name()))
		if err != nil {
			t.Fatal("open file failed: ", err)
		}

		dest, err := os.Create(path.Join(copied, f.Name()))
		if err != nil {
			t.Fatal("create file failed: ", err)
		}

		if _, err := io.Copy(dest, src); err != nil {
			t.Fatal("copy file failed:", err)
		}
	}

	return copied
}

func makeCorrupt(t *testing.T, dbfile string) {
	t.Helper()

	// make corrupt manifest file
	manifest := path.Join(dbfile, "MANIFEST-000000")
	if err := os.Truncate(manifest, 1); err != nil {
		t.Fatal("truncate file failed:", err)
	}
}

func TestOpen(t *testing.T) {
	// open the db file without faital error

	logger := util.HappoAgentLogger()
	stream := new(bytes.Buffer)
	logger.Out = stream

	testDB := copyTestDB(t)
	Open(testDB)

	assert.Empty(t, stream.String())
}

func TestOpen_corrupted(t *testing.T) {
	// recovery and open the db file without faital error

	logger := util.HappoAgentLogger()
	stream := new(bytes.Buffer)
	logger.Out = stream

	testDB := copyTestDB(t)
	makeCorrupt(t, testDB)
	Open(testDB)

	assert.Contains(t, stream.String(), fmt.Sprintf("[error] detect corrupted manifest file in %s", testDB))
	assert.Contains(t, stream.String(), fmt.Sprintf("[error] attempt recover for %s", testDB))
	assert.Contains(t, stream.String(), "recover corrupted manifest file succeeded")
}

func TestOpen_ReadWriteDBAfterRecover(t *testing.T) {
	testDB := copyTestDB(t)
	makeCorrupt(t, testDB)
	Open(testDB)

	if err := DB.Put([]byte("test-key"), []byte("test-value"), nil); err != nil {
		t.Fatal("wirte database failed:", err)
	}

	got, err := DB.Get([]byte("test-key"), nil)
	if err != nil {
		t.Fatal("read database failed:", err)
	}

	assert.Equal(t, []byte("test-value"), got)
}
