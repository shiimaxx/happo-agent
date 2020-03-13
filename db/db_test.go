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
	// make corrupt manifest file
	manifest := path.Join(testDB, "MANIFEST-000000")
	if err := os.Truncate(manifest, 1); err != nil {
		t.Fatal("truncate file failed:", err)
	}
	Open(testDB)

	assert.Contains(t, stream.String(), fmt.Sprintf("[error] detect corrupted manifest file in %s", testDB))
	assert.Contains(t, stream.String(), fmt.Sprintf("[error] attempt recover for %s", testDB))
	assert.Contains(t, stream.String(), "recover corrupted manifest file succeeded")
}
