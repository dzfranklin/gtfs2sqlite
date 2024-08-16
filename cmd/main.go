package main

import (
	"fmt"
	"github.com/dzfranklin/gtfs2sqlite"
	"github.com/spf13/pflag"
	"os"
	"path"
	"strings"
)

/* Timing notes on UK rail timetable:
Import without any tuning: 34s
Import with synchronous=OFF: 30s
Export without any tuning: 11s
*/

func usageAndDie() {
	fmt.Println("Example usage:\n" +
		"    gtfs2sqlite --import <timetable.zip>\n" +
		"    gtfs2sqlite --export <timetable.db>\n" +
		"    gtfs2sqlite --clip <timetable.db> --clip-feature <feature_geojson.json>")
	os.Exit(1)
}

func main() {
	importPath := pflag.StringP("import", "i", "", "Import from a GTFS file")
	exportPath := pflag.StringP("export", "e", "", "Export to a GTFS file")
	clipPath := pflag.StringP("clip", "c", "", "Clip a database")
	primaryOptions := []*string{importPath, exportPath, clipPath}

	output := pflag.StringP("out", "o", "", "Path to write output to")
	forceMode := pflag.BoolP("force-valid", "f", false, "Whether to fix issues by deleting data during import")
	ignoreInvalidMode := pflag.Bool("ignore-invalid", false, "Ignore any issues during import")
	clipFeaturePath := pflag.String("clip-feature", "", "If --clip is specified clips to the GeoJSON feature in the file specified")

	pflag.Parse()

	primaryCount := 0
	for _, opt := range primaryOptions {
		if *opt != "" {
			primaryCount++
		}
	}
	if primaryCount > 1 {
		usageAndDie()
	}

	var err error
	if *importPath != "" {
		outputPath := outputPathOrDefault(*importPath, *output, ".zip", ".db")
		opts := &gtfs2sqlite.ImportOpts{
			ForceValid:    *forceMode,
			IgnoreInvalid: *ignoreInvalidMode,
		}
		_, err = gtfs2sqlite.Import(*importPath, outputPath, opts)
	} else if *exportPath != "" {
		outputPath := outputPathOrDefault(*exportPath, *output, ".db", ".zip")
		opts := &gtfs2sqlite.ExportOpts{}
		err = gtfs2sqlite.Export(*exportPath, outputPath, opts)
	} else if *clipPath != "" {
		if *clipFeaturePath == "" {
			usageAndDie()
		}
		var feature []byte
		feature, err = os.ReadFile(*clipFeaturePath)
		if err != nil {
			panic(err)
		}
		featureName := trimFileExt(path.Base(*clipFeaturePath))

		outputPath := outputPathOrDefault(*clipPath, *output, ".db", fmt.Sprintf("_%s.db", featureName))
		err = gtfs2sqlite.Clip(*clipPath, outputPath, string(feature))
	} else {
		usageAndDie()
	}

	if err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	} else {
		fmt.Println("All done")
	}
}

func outputPathOrDefault(inputPath string, outputPath string, suffixToTrim string, newSuffix string) string {
	if outputPath != "" {
		return outputPath
	}
	inputPath = path.Clean(inputPath)
	return strings.TrimSuffix(path.Base(inputPath), suffixToTrim) + newSuffix
}

func trimFileExt(name string) string {
	i := strings.LastIndex(name, ".")
	if i == -1 {
		return name
	} else {
		return name[:i]
	}
}
