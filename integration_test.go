package gtfs2sqlite

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/hexops/gotextdiff/span"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
)

func TestConcurrent(t *testing.T) {
	outDir := testTempdir(t)
	var wg sync.WaitGroup
	for i := range 25 {
		wg.Add(1)
		go func() {
			defer wg.Done()

			outputPath := fmt.Sprintf("%s/%d.db", outDir, i)

			_, err := Import("./sample_data/sample-feed.zip", outputPath, nil)
			require.NoError(t, err)

			err = Export(outputPath, fmt.Sprintf("%s/%d.zip", outDir, i), nil)
			require.NoError(t, err)
		}()
	}
	wg.Wait()
}

func TestStableOutput(t *testing.T) {
	inputs := []string{"sample-feed.zip"}
	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			outDir := testTempdir(t)

			_, err := Import("./sample_data/"+input, outDir+"/imported.db", nil)
			require.NoError(t, err, "import")

			err = Export(outDir+"/imported.db", outDir+"/exported.zip", nil)
			require.NoError(t, err, "export")

			assertGTFSEqual(t, "./sample_data/"+input, outDir+"/exported.zip")
		})
	}
}

func TestPreservesUnknownFiles(t *testing.T) {
	outDir := testTempdir(t)

	_, err := Import("./sample_data/unknown-file.zip", outDir+"/imported.db", nil)
	require.NoError(t, err, "import")

	err = Export(outDir+"/imported.db", outDir+"/exported.zip", nil)
	require.NoError(t, err, "export")

	gotZip, err := zip.OpenReader(outDir + "/exported.zip")
	require.NoError(t, err)

	readFile := func(name string) []byte {
		gotF, err := gotZip.Open(name)
		require.NoError(t, err)
		got, err := io.ReadAll(gotF)
		require.NoError(t, err)
		return got
	}

	require.Equal(t, "some,columns\nsome,values\n", string(readFile("something_unknown.txt")))
	require.Equal(t, "{}\n", string(readFile("unknown_other_format.json")))
}

func TestClip(t *testing.T) {
	outDir := testTempdir(t)

	feature, err := os.ReadFile("./sample_data/ne_beatty.json")
	require.NoError(t, err)

	_, err = Import("./sample_data/sample-multiagency-feed.zip", outDir+"/imported.db", nil)
	require.NoError(t, err, "import")

	err = Clip(outDir+"/imported.db", outDir+"/clipped.db", string(feature))
	require.NoError(t, err)

	err = Export(outDir+"/clipped.db", outDir+"/exported.zip", nil)
	require.NoError(t, err, "export")

	assertGTFSEqual(t, "./sample_data/sample-multiagency-feed-clipped-to-ne_beatty.zip", outDir+"/exported.zip")
}

func assertGTFSEqual(t *testing.T, expected, actual string) {
	t.Helper()

	expectedZip, err := zip.OpenReader(expected)
	if err != nil {
		panic(err)
	}
	actualZip, err := zip.OpenReader(actual)
	if err != nil {
		panic(err)
	}

	var expectedFiles []string
	for _, entry := range expectedZip.File {
		expectedFiles = append(expectedFiles, entry.Name)
	}
	var actualFiles []string
	for _, entry := range actualZip.File {
		actualFiles = append(actualFiles, entry.Name)
	}

	var removedFiles []string
	for _, file := range expectedFiles {
		if !slices.Contains(actualFiles, file) {
			removedFiles = append(removedFiles, file)
		}
	}
	slices.Sort(removedFiles)
	var addedFiles []string
	for _, file := range actualFiles {
		if !slices.Contains(expectedFiles, file) {
			addedFiles = append(addedFiles, file)
		}
	}
	slices.Sort(addedFiles)
	var filesToCheck []string
	for _, file := range actualFiles {
		if !slices.Contains(removedFiles, file) && !slices.Contains(addedFiles, file) {
			filesToCheck = append(filesToCheck, file)
		}
	}
	slices.Sort(filesToCheck)

	var out strings.Builder

	if len(addedFiles) > 0 || len(removedFiles) > 0 {
		t.Fail()
	}
	for _, name := range addedFiles {
		fmt.Fprintf(&out, "ADDED FILE %s", name)
	}
	for _, name := range removedFiles {
		fmt.Fprintf(&out, "REMOVED FILE %s", name)
	}

	for _, file := range filesToCheck {
		expectedF, err := expectedZip.Open(file)
		if err != nil {
			panic(err)
		}
		actualF, err := actualZip.Open(file)
		if err != nil {
			panic(err)
		}

		var expectedContent []byte
		var actualContent []byte
		if strings.HasSuffix(file, ".txt") {
			var baseColumns []string
			if schema, ok := gtfsSchema[strings.TrimSuffix(file, ".txt")]; ok {
				for col, _ := range schema.Columns {
					baseColumns = append(baseColumns, col)
				}
			}

			expectedContent, err = normalizeCSV(expectedF, baseColumns)
			if err != nil {
				panic(err)
			}
			actualContent, err = normalizeCSV(actualF, baseColumns)
			if err != nil {
				panic(err)
			}
		} else {
			expectedContent, err = io.ReadAll(expectedF)
			if err != nil {
				panic(err)
			}
			actualContent, err = io.ReadAll(actualF)
			if err != nil {
				panic(err)
			}
		}

		edits := myers.ComputeEdits(span.URIFromPath(file), string(expectedContent), string(actualContent))
		if len(edits) > 0 {
			t.Fail()
			fmt.Fprint(&out, gotextdiff.ToUnified("expected/"+file, "actual/"+file, string(expectedContent), edits))
		}
	}

	if out.Len() > 0 {
		t.Log(expected, "!=", actual, "\n", out.String())
	}
}

func normalizeCSV(input io.Reader, baseColumns []string) ([]byte, error) {
	r := csv.NewReader(input)
	r.FieldsPerRecord = -1

	var out bytes.Buffer
	w := csv.NewWriter(&out)

	srcHeader, err := r.Read()
	if err != nil {
		return nil, err
	}

	headerOccurrences := make(map[string]int)
	for _, col := range srcHeader {
		headerOccurrences[col]++
	}
	for _, count := range headerOccurrences {
		if count > 1 {
			return nil, errors.New("normalizeCSV doesn't currently support duplicated column names")
		}
	}

	header := make([]string, len(srcHeader))
	copy(header, srcHeader)
	for _, col := range baseColumns {
		if !slices.Contains(header, col) {
			header = append(header, col)
		}
	}
	slices.Sort(header)

	headerSort := make([]int, len(srcHeader))
	for srcI, col := range srcHeader {
		dstI := slices.Index(header, col)
		if dstI == -1 {
			panic("unreachable")
		}
		headerSort[srcI] = dstI
	}

	if err := w.Write(header); err != nil {
		return nil, err
	}

	for {
		srcRow, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return nil, err
		}

		row := make([]string, len(header))
		for srcI := range srcRow {
			row[headerSort[srcI]] = srcRow[srcI]
		}

		if err := w.Write(row); err != nil {
			return nil, err
		}
	}

	w.Flush()
	return out.Bytes(), w.Error()
}

func TestHelperNormalizeCSV(t *testing.T) {
	sample := "a,c,b\n1,3,2\n1,0,1"
	expected := "a,b,c,d\n1,2,3,\n1,1,0,\n"

	got, err := normalizeCSV(bytes.NewReader([]byte(sample)), []string{"a", "b", "c", "d"})
	require.NoError(t, err)
	assert.Equal(t, expected, string(got))
}

func TestHelperAssertGTFSEqual(t *testing.T) {
	assertGTFSEqual(t, "./sample_data/sample-feed.zip", "./sample_data/sample-feed.zip")
}
