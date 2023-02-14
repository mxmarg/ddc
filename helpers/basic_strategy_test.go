/*
   Copyright 2022 Ryan SVIHLA

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

// helpers package provides different functionality

package helpers

import (
	"path/filepath"
	"testing"
	"time"
)

// Tests the constructor is setting a basedir dir
func TestBaseDirDF(t *testing.T) {
	ddcfs := NewFakeFileSystem()
	testStrat := NewBACopyStrategy(ddcfs)
	expected := time.Now().Format("20060102-150405-DDC")
	actual := testStrat.BaseDir
	// Check the base dir is set on creation
	if expected != actual {
		t.Errorf("ERROR: base directory name on create: \nexpected:\t%v\nactual:\t\t%v\n", expected, actual)
	}
}

// Tests the constructor is setting a temp dir
func TestTmpDirDF(t *testing.T) {
	ddcfs := NewFakeFileSystem()
	testStrat := NewBACopyStrategy(ddcfs)
	expected := filepath.Join("tmp", "dir1", "random")
	actual := testStrat.TmpDir
	// Check the base dir is set on creation
	if expected != actual {
		t.Errorf("ERROR: tmp directory on create: \nexpected:\t%v\nactual:\t\t%v\n", expected, actual)
	}
}

// Tests the method returns the correct path
func TestGetPathDF(t *testing.T) {
	ddcfs := NewFakeFileSystem()
	testStrat := NewBACopyStrategy(ddcfs)
	// Test path for coordinators
	expected := filepath.Join("tmp", "dir1", "random", testStrat.BaseDir, "coordinators", "node1", "log")
	actual, _ := testStrat.CreatePath("log", "node1", "coordinator")
	if expected != actual {
		t.Errorf("\nERROR: returned path: \nexpected:\t%v\nactual:\t\t%v\n", expected, actual)
	}
	// Test path for executors
	expected = filepath.Join("tmp", "dir1", "random", testStrat.BaseDir, "executors", "node1", "log")
	actual, _ = testStrat.CreatePath("log", "node1", "executors")
	if expected != actual {
		t.Errorf("\nERROR: returned path: \nexpected:\t%v\nactual:\t\t%v\n", expected, actual)
	}
}

// Tests the method returns the correct path
func TestGzipFilesDF(t *testing.T) {
	ddcfs := NewFakeFileSystem()
	testStrat := NewBACopyStrategy(ddcfs)
	// Test gzip is a noop essentially, but we can still check for a nil response
	_, actual := testStrat.GzipAllFiles("/tmp")
	if actual != nil {
		t.Errorf("\nERROR: gzip file: \nexpected:\t%v\nactual:\t\t%v\n", nil, actual)
	}
}

// Test archiving of a file (which is also tested elsewhere) but in addition
// it tests the call via the selected strategy
func TestArchiveDiagDF(t *testing.T) {
	ddcfs := NewRealFileSystem()
	testStrat := NewBACopyStrategy(ddcfs)
	tmpDir := t.TempDir()
	testFileRaw := filepath.Join("testdata", "test.txt")
	testFile, err := filepath.Abs(testFileRaw)
	if err != nil {
		t.Fatalf("not able to get absolute path for test file %v", err)
	}
	fi, err := ddcfs.Stat(testFile)
	if err != nil {
		t.Fatalf("unexpected error getting file size for file %v due to error %v", testFile, err)
	}
	archiveFile := tmpDir + ".zip"
	if err != nil {
		t.Fatalf("not able to get absolute path for testdata dir %v", err)
	}
	testFiles := []CollectedFile{
		{
			Path: testFile,
			Size: int64(fi.Size()),
		},
	}
	// Test Archive, pushes a teal test file into a zip archive
	err = testStrat.ArchiveDiag("test", archiveFile, testFiles)
	if err != nil {
		t.Errorf("\nERROR: gzip file: \nexpected:\t%v\nactual:\t\t%v\n", nil, err)
	}
}