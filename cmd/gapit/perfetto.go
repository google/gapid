// Copyright (C) 2018 Google Inc.
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

package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/client"
	"github.com/google/gapid/gapis/service/path"
)

type perfettoVerb struct{ PerfettoFlags }

func init() {
	verb := &perfettoVerb{}
	app.AddVerb(&app.Verb{
		Name:      "perfetto",
		ShortHelp: "Run metrics and interact with system profiler trace.",
		Action:    verb,
	})
}

func (verb *perfettoVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	if flags.NArg() != 1 {
		app.Usage(ctx, "Exactly one perfetto trace file expected, got %d", flags.NArg())
		return nil
	}

	trace := flags.Arg(0)
	if _, err := os.Stat(trace); os.IsNotExist(err) {
		return fmt.Errorf("Could not find trace file: %v", trace)
	}

	run_metrics := true
	switch verb.Mode {
	case ModeMetrics:
		// default: run_metrics
	case ModeInteractive:
		run_metrics = false
	default:
		app.Usage(ctx, "Run mode should be 'metrics' or 'interactive', got '%s'.", verb.Mode)
	}

	var input string
	if verb.In != "" {
		input, _ = filepath.Abs(verb.In)
		if _, err := os.Stat(input); os.IsNotExist(err) {
			return fmt.Errorf("Could not find input queries file: %v", input)
		}
	}

	var categories []string
	if run_metrics && verb.Categories != "" {
		categories = strings.Split(verb.Categories, ",")
	}

	output := ""
	if verb.Out != "" {
		output, _ = filepath.Abs(verb.Out)
	}

	outputFormat := OutputText
	fmt.Println("verb.Format: ", verb.Format)
	switch verb.Format {
	case OutputDefault:
		if output == "" {
			outputFormat = OutputText
		} else {
			outputFormat = OutputJson
		}
	case OutputText:
		break
	case OutputJson:
		outputFormat = OutputJson
	default:
		app.Usage(ctx, "Output format should be 'text' or 'json', got '%s'.", verb.Format)
	}

	if run_metrics {
		return RunMetrics(ctx, trace, input, categories, output, outputFormat)
	} else {
		return RunInteractive(ctx, trace, input, output, outputFormat)
	}
}

func RunQuery(ctx context.Context, cclient client.Client, capture *path.Capture, query string) (numColumns int, columnNames []string, columnTypes []string, numRecords uint64, dataStrings [][]string, dataLongs [][]int64, dataDoubles [][]float64, err error) {
	queryResult, error := cclient.PerfettoQuery(ctx, capture, query)
	if error != nil {
		err = fmt.Errorf("Error while running query %s : %v.", query, err)
		return
	}
	columns := queryResult.GetColumns()
	numColumns = len(columns)
	numRecords = queryResult.GetNumRecords()
	columnNames = make([]string, numColumns)
	columnTypes = make([]string, numColumns)
	columnDescriptors := queryResult.GetColumnDescriptors()
	dataStrings = make([][]string, numColumns)
	dataLongs = make([][]int64, numColumns)
	dataDoubles = make([][]float64, numColumns)
	for i := 0; i < numColumns; i++ {
		columnNames[i] = columnDescriptors[i].GetName()
		columnTypes[i] = columnDescriptors[i].GetType().String()
		switch columnTypes[i] {
		case "UNKNOWN":
			fallthrough
		case "STRING":
			dataStrings[i] = columns[i].GetStringValues()
		case "LONG":
			dataLongs[i] = columns[i].GetLongValues()
		case "DOUBLE":
			dataDoubles[i] = columns[i].GetDoubleValues()
		}
	}
	return
}

func RunMetrics(ctx context.Context, trace string, input string, categories []string, output string, format PerfettoOutputFormat) error {
	return nil
}

func RunInteractive(ctx context.Context, trace string, inputPath string, outputPath string, format PerfettoOutputFormat) error {
	// Load the preparation queries from input file. For these queries we do not report the output.
	var prepQueries []string
	if inputPath != "" {
		queriesFile, err := os.Open(inputPath)
		if err != nil {
			app.Usage(ctx, "Cannot open file %s: %v.", inputPath, err)
		}
		defer queriesFile.Close()
		scanner := bufio.NewScanner(queriesFile)
		for scanner.Scan() {
			prepQueries = append(prepQueries, scanner.Text())
		}
	}

	// Load main queries from stdin if any exist. We report the result of these queries.
	pipeMode := false
	var queries []string
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		pipeMode = true
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			queries = append(queries, scanner.Text())
		}
	}

	// Load the trace
	client, capture, err := getGapisAndLoadCapture(ctx, GapisFlags{}, GapirFlags{}, trace, CaptureFileFlags{})
	if err != nil {
		return fmt.Errorf("Error while loading the trace file %s: %v.", trace, err)
	}
	defer client.Close()

	// Run the preparation queries
	for _, query := range prepQueries {
		log.I(ctx, "Running query: %s", query)
		_, err := client.PerfettoQuery(ctx, capture, query)
		if err != nil {
			return fmt.Errorf("Error while running query: %v.", err)
		}
	}

	// If queries are passed as a pipe, run the queries, report the results and finish the execution
	if pipeMode {
		if outputPath != "" {
			InitilizeOutputFile(ctx, outputPath, format)
		}
		for qindex, query := range queries {
			log.I(ctx, "Running query: %s", query)
			numColumns, columnNames, columnTypes, numRecords, dataStrings, dataLongs, dataDoubles, err := RunQuery(ctx, client, capture, query)
			if err != nil {
				return err
			}
			if format == OutputText {
				err := ReportQueryResultsInTextFormat(query, numColumns, columnNames, columnTypes, numRecords, dataStrings, dataLongs, dataDoubles, outputPath)
				if err != nil {
					return err
				}
			} else {
				lastQuery := qindex == len(queries)-1
				err := ReportQueryResultsInJSONFormat(query, lastQuery, numColumns, columnNames, columnTypes, numRecords, dataStrings, dataLongs, dataDoubles, outputPath)
				if err != nil {
					return err
				}
			}
		}
		if outputPath != "" {
			FinalizeOutputFile(ctx, outputPath, format)
		}
		return nil
	}

	// Start the interactive mode
	log.I(ctx, "Starting interactive mode ...")
	valid_results := false
	var numColumns int
	var columnNames []string
	var columnTypes []string
	var numRecords uint64
	var dataStrings [][]string
	var dataLongs [][]int64
	var dataDoubles [][]float64
	var query string

	for {
		valid_command := false
		fmt.Print(">> ")
		reader := bufio.NewReader(os.Stdin)
		cmdString, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		if strings.HasPrefix(cmdString, "run") {
			valid_results = false
			query = strings.TrimSuffix(strings.TrimPrefix(cmdString, "run"), "\n")
			var err error
			numColumns, columnNames, columnTypes, numRecords, dataStrings, dataLongs, dataDoubles, err = RunQuery(ctx, client, capture, query)
			if err != nil {
				fmt.Println("Error while running query: ", err)
			} else {
				valid_results = true
				err = ReportQueryResultsInTextFormat(query, numColumns, columnNames, columnTypes, numRecords, dataStrings, dataLongs, dataDoubles, "")
				if err != nil {
					fmt.Println(err)
				}
			}
			valid_command = true
			// Run the query here
		} else if strings.HasPrefix(cmdString, "save") {
			if !valid_results {
				fmt.Println("No valid query results available to save.")
			} else {
				cmdString := strings.TrimSpace(strings.TrimPrefix(cmdString, "save"))
				if strings.HasPrefix(cmdString, "text") || strings.HasPrefix(cmdString, "json") {
					valid_command = true
					prefix := cmdString[:4]
					path := strings.TrimSpace(cmdString[4:])
					if strings.HasPrefix(path, "~/") {
						usr, _ := user.Current()
						path = filepath.Join(usr.HomeDir, path[2:])
					}
					var err error
					switch prefix {
					case "text":
						err = InitilizeOutputFile(ctx, path, OutputText)
						if err == nil {
							err = ReportQueryResultsInTextFormat(query, numColumns, columnNames, columnTypes, numRecords, dataStrings, dataLongs, dataDoubles, path)
						}
					case "json":
						err = InitilizeOutputFile(ctx, path, OutputJson)
						if err == nil {
							lastQuery := true
							err = ReportQueryResultsInJSONFormat(query, lastQuery, numColumns, columnNames, columnTypes, numRecords, dataStrings, dataLongs, dataDoubles, path)
						}
					}
					if err != nil {
						fmt.Println(err)
					} else {
						fmt.Println("Results saved to ", path)
					}
				}
			}
		} else if strings.HasPrefix(cmdString, "quit") {
			break
		}
		if !valid_command {
			fmt.Println("Please use one of the following commands:")
			fmt.Println("\t>> run <sql_query>")
			fmt.Println("\t>> save text <filename>")
			fmt.Println("\t>> save json <filename>")
			fmt.Println("\t>> quit")
		}
	}
	return nil
}

func InitilizeOutputFile(ctx context.Context, outputPath string, outputFormat PerfettoOutputFormat) error {
	output, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("Error while creating the output file %s: %v.", outputPath, err)
	}
	defer output.Close()
	if outputFormat == OutputJson {
		if _, err := output.Write([]byte("{\"Queries\":[\n")); err != nil {
			return fmt.Errorf("Error writing to the output file %s: %v.", outputPath, err)
		}
	}
	return nil
}

func FinalizeOutputFile(ctx context.Context, outputPath string, outputFormat PerfettoOutputFormat) error {
	output, err := os.OpenFile(outputPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("Error while opening the output file %s: %v.", outputPath, err)
	}
	defer output.Close()
	if outputFormat == OutputJson {
		if _, err := output.Write([]byte("]}")); err != nil {
			return fmt.Errorf("Error writing to the output file %s: %v.", outputPath, err)
		}
	}
	return nil
}

func ReportQueryResultsInTextFormat(query string, numColumns int, columnNames []string, columnTypes []string, numRecords uint64, dataStrings [][]string, dataLongs [][]int64, dataDoubles [][]float64, outputPath string) error {
	var output *os.File
	var err error
	if outputPath != "" {
		output, err = os.OpenFile(outputPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("Error while opening the output file %s: %v.", outputPath, err)
		}
		defer output.Close()
	} else {
		output = os.Stdout
	}
	const padding = 2
	writer := tabwriter.NewWriter(output, 0, 0, padding, ' ', 0)
	fmt.Fprintln(writer, "Query: "+query)
	for i := 0; i < numColumns; i++ {
		fmt.Fprint(writer, columnNames[i]+"\t")
	}
	fmt.Fprintln(writer)

	for i := uint64(0); i < numRecords; i++ {
		for j := 0; j < numColumns; j++ {
			switch columnTypes[j] {
			case "UNKNOWN":
				fmt.Fprintf(writer, "NULL\t")
			case "STRING":
				fmt.Fprintf(writer, "%s\t", dataStrings[j][i])
			case "LONG":
				fmt.Fprintf(writer, "%d\t", dataLongs[j][i])
			case "DOUBLE":
				fmt.Fprintf(writer, "%f\t", dataDoubles[j][i])
			}
		}
		fmt.Fprintln(writer)
	}
	fmt.Fprintln(writer)
	writer.Flush()
	return nil
}

func ReportQueryResultsInJSONFormat(query string, lastQuery bool, numColumns int, columnNames []string, columnTypes []string, numRecords uint64, dataStrings [][]string, dataLongs [][]int64, dataDoubles [][]float64, outputPath string) error {
	var output *os.File
	var err error
	if outputPath != "" {
		output, err = os.OpenFile(outputPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("Error while opening the output file %s: %v.", outputPath, err)
		}
		defer output.Close()
	} else {
		output = os.Stdout
	}

	writer := bufio.NewWriter(output)
	const padding1 = "  "
	const padding2 = "    "
	const padding3 = "      "
	const padding4 = "        "
	writer.WriteString(padding1 + "{ \"query\": \"" + strings.ReplaceAll(query, "\"", "\\\"") + "\",\n")
	writer.WriteString(padding2 + "\"result\": [\n")

	for i := uint64(0); i < numRecords; i++ {
		for j := 0; j < numColumns; j++ {
			if j == 0 {
				writer.WriteString(padding3 + "{ ")
			} else {
				writer.WriteString(padding4)
			}
			writer.WriteString("\"" + columnNames[j] + "\": \"")
			switch columnTypes[j] {
			case "UNKNOWN":
				writer.WriteString("NULL\"")
			case "STRING":
				writer.WriteString(fmt.Sprintf("%s", dataStrings[j][i]) + "\"")
			case "LONG":
				writer.WriteString(fmt.Sprintf("%d", dataLongs[j][i]) + "\"")
			case "DOUBLE":
				writer.WriteString(fmt.Sprintf("%f", dataDoubles[j][i]) + "\"")
			}
			if j != numColumns-1 {
				writer.WriteString(",\n")
			} else {
				writer.WriteString("\n")
			}
		}
		if i != numRecords-1 {
			writer.WriteString(padding3 + "},\n")
		} else {
			writer.WriteString(padding3 + "}\n" + padding2 + "]\n")
		}
	}
	if lastQuery {
		writer.WriteString(padding1 + "}\n")

	} else {
		writer.WriteString(padding1 + "},\n")
	}

	writer.Flush()
	return nil
}
