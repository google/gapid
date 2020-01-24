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
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
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
	} else if run_metrics == true {
		log.I(ctx, "No input file is given to read the metric definitions. Default metrics will be run. You can use '-in <metrics-json-file>' to provide custom metric definitions or  '-mode interactive' to use the interactive mode.")
	}

	var empty struct{}
	var categories map[string]struct{}
	if run_metrics && verb.Categories != "" {
		catstrs := strings.Split(verb.Categories, ",")
		for _, s := range catstrs {
			categories[s] = empty
		}
	}

	output := ""
	if verb.Out != "" {
		output, _ = filepath.Abs(verb.Out)
	}

	outputFormat := OutputText
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

func RunPrepQuery(ctx context.Context, cclient client.Client, capture *path.Capture, query string) (err error) {
	log.I(ctx, "Running query: %s", query)
	_, error := cclient.PerfettoQuery(ctx, capture, query)
	if error != nil {
		err = fmt.Errorf("Error while running query %s : %v.", query, err)
	}
	return
}

func RunQuery(ctx context.Context, cclient client.Client, capture *path.Capture, query string) (numColumns int, columnNames []string, columnTypes []string, numRecords uint64, dataStrings [][]string, dataLongs [][]int64, dataDoubles [][]float64, err error) {
	log.I(ctx, "Running query: %s", query)
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

type MetricResult struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type MetricResults struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Results     []MetricResult `json:"results"`
}

type CategoryResults struct {
	Name    string          `json:"name"`
	Metrics []MetricResults `json:"metrics"`
}

type MetricsResults struct {
	Categories []CategoryResults `json:"categories"`
}

func RunMetrics(ctx context.Context, trace string, inputPath string, categories map[string]struct{}, outputPath string, format PerfettoOutputFormat) error {

	type Query struct {
		Query string `json:"query"`
	}

	type Metric struct {
		Name        string  `json:"name"`
		Description string  `json:"description"`
		Queries     []Query `json:"queries"`
	}

	type MetricCategory struct {
		Name    string   `json:"category"`
		Metrics []Metric `json:"metric"`
	}

	type MetricsInfo struct {
		PrepQueries      []Query          `json:"initialize"`
		MetricCategories []MetricCategory `json:"metrics"`
	}

	metricsFile, err := os.Open(inputPath)
	if err != nil {
		app.Usage(ctx, "Cannot open file %s: %v.", inputPath, err)
	}
	defer metricsFile.Close()
	byteValue, _ := ioutil.ReadAll(metricsFile)
	var metricsInfo MetricsInfo
	json.Unmarshal(byteValue, &metricsInfo)

	// Load the trace
	client, capture, err := getGapisAndLoadCapture(ctx, GapisFlags{}, GapirFlags{}, trace, CaptureFileFlags{})
	if err != nil {
		return fmt.Errorf("Error while loading the trace file %s: %v.", trace, err)
	}
	defer client.Close()

	// Run the preparation queries
	for _, query := range metricsInfo.PrepQueries {
		if err := RunPrepQuery(ctx, client, capture, query.Query); err != nil {
			return err
		}
	}

	if format == OutputText {
		if err := InitilizeOutputFile(ctx, ModeMetrics, outputPath, format); err != nil {
			return err
		}
	}

	var metricsResults MetricsResults
	runCustomCategories := len(categories) > 0
	for i := 0; i < len(metricsInfo.MetricCategories); i++ {
		categoryName := metricsInfo.MetricCategories[i].Name
		if runCustomCategories {
			if _, ok := categories[categoryName]; !ok {
				continue
			}
		}
		log.I(ctx, "Running Category: %s", categoryName)
		categoryResults := CategoryResults{Name: categoryName}
		metrics := &(metricsInfo.MetricCategories[i].Metrics)
		for j := 0; j < len(*metrics); j++ {
			log.I(ctx, "Running Metric: %s", (*metrics)[j].Name)
			queries := &(*metrics)[j].Queries
			if len(*queries) == 0 {
				log.I(ctx, "No queires found to run for this metric.")
			} else {
				for k := 0; k < len(*queries)-1; k++ {
					if err := RunPrepQuery(ctx, client, capture, (*queries)[k].Query); err != nil {
						return err
					}
				}
				numColumns, columnNames, columnTypes, numRecords, dataStrings, dataLongs, dataDoubles, err := RunQuery(ctx, client, capture, (*queries)[len(*queries)-1].Query)
				if err != nil {
					return err
				}
				if numRecords == 0 {
					log.I(ctx, "No records returned by this query.")
				} else if format == OutputText {
					err := ReportMetricResultsInTextFormat(categoryName, (*metrics)[j].Name, (*metrics)[j].Description, numColumns, columnNames, columnTypes, dataStrings, dataLongs, dataDoubles, outputPath)
					if err != nil {
						return err
					}
				} else {
					categoryResults.Metrics = append(categoryResults.Metrics, CreateMetricResult((*metrics)[j].Name, (*metrics)[j].Description, numColumns, columnNames, columnTypes, dataStrings, dataLongs, dataDoubles))
				}
			}
		}
		metricsResults.Categories = append(metricsResults.Categories, categoryResults)
	}
	if format == OutputJson {
		if err := WriteResultsToJSONOutput(metricsResults, outputPath); err != nil {
			return err
		}
	}
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
		if err := RunPrepQuery(ctx, client, capture, query); err != nil {
			return err
		}
	}

	// If queries are passed as a pipe, run the queries, report the results and finish the execution
	if pipeMode {
		err := InitilizeOutputFile(ctx, ModeInteractive, outputPath, format)
		if err != nil {
			return err
		}
		for qindex, query := range queries {
			log.I(ctx, "Running query: %s", query)
			numColumns, columnNames, columnTypes, numRecords, dataStrings, dataLongs, dataDoubles, err := RunQuery(ctx, client, capture, query)
			if err != nil {
				return err
			}
			if format == OutputText {
				err := ReportAllQueryResultsInTextFormat(query, numColumns, columnNames, columnTypes, numRecords, dataStrings, dataLongs, dataDoubles, outputPath)
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
		if format == OutputJson {
			if err := FinalizeJSONOutput(ctx, outputPath); err != nil {
				return err
			}
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
				startIndex := uint64(0)
				recordCount := uint64(10)
				for {
					err = ReportPartialQueryResultsOnConsole(query, numColumns, columnNames, columnTypes, numRecords, dataStrings, dataLongs, dataDoubles, startIndex, recordCount)
					if err != nil {
						fmt.Println(err)
					}
					if startIndex+recordCount >= numRecords {
						break
					}
					fmt.Print("Press any key for more results; press 'q' to quit... ")
					reader := bufio.NewReaderSize(os.Stdin, 1)
					input, _ := reader.ReadByte()
					if input == 'q' {
						break
					}
					startIndex = startIndex + recordCount
				}
			}
			valid_command = true
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
						err = InitilizeOutputFile(ctx, ModeInteractive, path, OutputText)
						if err != nil {
							err = ReportAllQueryResultsInTextFormat(query, numColumns, columnNames, columnTypes, numRecords, dataStrings, dataLongs, dataDoubles, path)
						}
					case "json":
						err = InitilizeOutputFile(ctx, ModeInteractive, path, OutputJson)
						if err == nil {
							lastQuery := true
							err = ReportQueryResultsInJSONFormat(query, lastQuery, numColumns, columnNames, columnTypes, numRecords, dataStrings, dataLongs, dataDoubles, path)
						}
						if err == nil {
							if err = FinalizeJSONOutput(ctx, outputPath); err != nil {
								return err
							}
						}
					}
					fmt.Println("Results saved to ", path)
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

func InitilizeOutputFile(ctx context.Context, runMode PerfettoMode, outputPath string, outputFormat PerfettoOutputFormat) error {
	var output *os.File
	var err error
	if outputPath != "" {
		output, err = os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("Error while creating the output file %s: %v.", outputPath, err)
		}
		defer output.Close()
	} else {
		output = os.Stdout
	}
	if runMode == ModeInteractive && outputFormat == OutputJson {
		if _, err := output.Write([]byte("{\"queries\":[\n")); err != nil {
			return fmt.Errorf("Error writing to the output file %s: %v.", outputPath, err)
		}
	} else if runMode == ModeMetrics && outputFormat == OutputText {
		if _, err := output.Write([]byte("GAPID System Profiler Metrics\n")); err != nil {
			return fmt.Errorf("Error writing to the output file %s: %v.", outputPath, err)
		}
	}
	return nil
}

func FinalizeJSONOutput(ctx context.Context, outputPath string) error {
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
	if _, err := output.Write([]byte("]}")); err != nil {
		return fmt.Errorf("Error writing to the output file %s: %v.", outputPath, err)
	}
	return nil
}

func ReportAllQueryResultsInTextFormat(query string, numColumns int, columnNames []string, columnTypes []string, numRecords uint64, dataStrings [][]string, dataLongs [][]int64, dataDoubles [][]float64, outputPath string) error {
	return ReportQueryResultsInTextFormat(query, numColumns, columnNames, columnTypes, numRecords, dataStrings, dataLongs, dataDoubles, outputPath, true, 0, 0)
}

func ReportPartialQueryResultsOnConsole(query string, numColumns int, columnNames []string, columnTypes []string, numRecords uint64, dataStrings [][]string, dataLongs [][]int64, dataDoubles [][]float64, startIndex uint64, count uint64) error {
	return ReportQueryResultsInTextFormat(query, numColumns, columnNames, columnTypes, numRecords, dataStrings, dataLongs, dataDoubles, "", false, startIndex, count)
}

func ReportQueryResultsInTextFormat(query string, numColumns int, columnNames []string, columnTypes []string, numRecords uint64, dataStrings [][]string, dataLongs [][]int64, dataDoubles [][]float64, outputPath string, showAll bool, startIndex uint64, count uint64) error {
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
	for i := 0; i < numColumns; i++ {
		fmt.Fprint(writer, columnNames[i]+"\t")
	}
	fmt.Fprintln(writer)

	if showAll {
		fmt.Fprintln(writer, "Query: "+query)
		startIndex = 0
		count = numRecords
	}
	endIndex := numRecords
	if endIndex > startIndex+count {
		endIndex = startIndex + count
	}
	for i := startIndex; i < endIndex; i++ {
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

func ReportMetricResultsInTextFormat(categoryName string, metric string, description string, numColumns int, columnNames []string, columnTypes []string, dataStrings [][]string, dataLongs [][]int64, dataDoubles [][]float64, outputPath string) error {
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
	writer.WriteString("\nCategory: " + categoryName + "\n")
	writer.WriteString("Metric name: " + metric + "\n")
	writer.WriteString("Description: " + description + "\n")
	for i := 0; i < numColumns; i++ {
		switch columnTypes[i] {
		case "UNKNOWN":
			writer.WriteString(columnNames[i] + ": NULL\n")
		case "STRING":
			writer.WriteString(columnNames[i] + ": " + dataStrings[i][0] + "\n")
		case "LONG":
			writer.WriteString(columnNames[i] + ": " + strconv.FormatInt(dataLongs[i][0], 10) + "\n")
		case "DOUBLE":
			writer.WriteString(columnNames[i] + ": " + strconv.FormatFloat(dataDoubles[i][0], 'E', -1, 64) + "\n")
		}
	}
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

func CreateMetricResult(metric string, description string, numColumns int, columnNames []string, columnTypes []string, dataStrings [][]string, dataLongs [][]int64, dataDoubles [][]float64) (metricResults MetricResults) {
	metricResults.Name = metric
	metricResults.Description = description
	for i := 0; i < numColumns; i++ {
		var result string
		switch columnTypes[i] {
		case "UNKNOWN":
			result = "NULL"
		case "STRING":
			result = dataStrings[i][0]
		case "LONG":
			result = strconv.FormatInt(dataLongs[i][0], 10)
		case "DOUBLE":
			result = strconv.FormatFloat(dataDoubles[i][0], 'E', -1, 64)
		}
		metricResults.Results = append(metricResults.Results, MetricResult{Name: columnNames[i], Value: result})
	}
	return
}

func WriteResultsToJSONOutput(metricsResults MetricsResults, outputPath string) error {
	var output *os.File
	var err error
	if outputPath != "" {
		output, err = os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("Error while opening the output file %s: %v.", outputPath, err)
		}
		defer output.Close()
	} else {
		output = os.Stdout
	}

	jsonResult, err := json.MarshalIndent(&metricsResults, "", "  ")
	if err != nil {
		return fmt.Errorf("Error converting the metrics results to Json. Please file a bug and provide as much as information as possbile to address this issue. Error: %v", err)
	}
	if _, err := output.Write(jsonResult); err != nil {
		return fmt.Errorf("Error writing to the output file %s: %v.", outputPath, err)
	}
	output.Sync()
	return nil
}
