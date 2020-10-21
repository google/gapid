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
	var trace string
	if verb.Mode != ModeList {
		if flags.NArg() != 1 {
			app.Usage(ctx, "Exactly one perfetto trace file expected, got %d", flags.NArg())
			return nil
		}

		trace = flags.Arg(0)
		if _, err := os.Stat(trace); os.IsNotExist(err) {
			return fmt.Errorf("Could not find trace file: %v", trace)
		}
	}

	if verb.Mode != ModeMetrics && verb.Mode != ModeInteractive && verb.Mode != ModeList {
		app.Usage(ctx, "Run mode should be 'metrics', 'interactive' or 'list', got '%s'.", verb.Mode)
	}

	var input string
	if verb.In != "" {
		input, _ = filepath.Abs(verb.In)
		if _, err := os.Stat(input); os.IsNotExist(err) {
			return fmt.Errorf("Could not find input queries file: %v", input)
		}
	} else if verb.Mode == ModeMetrics {
		log.I(ctx, "No input file is given to read the metric definitions. Default metrics will be run. You can use '-in <metrics-json-file>' to provide custom metric definitions or  '-mode interactive' to use the interactive mode.")
	}

	var empty struct{}
	categories := make(map[string]struct{})
	if verb.Mode != ModeInteractive && verb.Categories != "" {
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

	if verb.Mode == ModeMetrics {
		return RunMetrics(ctx, verb.Gapis, trace, input, categories, output, outputFormat)
	} else if verb.Mode == ModeInteractive {
		return RunInteractive(ctx, verb.Gapis, trace, input, output, outputFormat)
	} else {
		return ListMetrics(ctx, input, categories)
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

type MetricResultRecord struct {
	Results []MetricResult `json:"result"`
}

type MetricResults struct {
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Results     []MetricResultRecord `json:"results"`
}

type CategoryResults struct {
	Name    string          `json:"name"`
	Metrics []MetricResults `json:"metrics"`
}

type MetricsResults struct {
	Categories []CategoryResults `json:"categories"`
}

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

func RunMetrics(ctx context.Context, gapisFlags GapisFlags, trace string, inputPath string, categories map[string]struct{}, outputPath string, format PerfettoOutputFormat) error {
	var byteValue []byte
	if inputPath != "" {
		metricsFile, err := os.Open(inputPath)
		if err != nil {
			app.Usage(ctx, "Cannot open file %s: %v.", inputPath, err)
		}
		defer metricsFile.Close()
		byteValue, _ = ioutil.ReadAll(metricsFile)
	} else {
		byteValue = []byte(PredefinedMetrics())
	}
	var metricsInfo MetricsInfo
	err := json.Unmarshal(byteValue, &metricsInfo)
	if err != nil {
		return fmt.Errorf("Error while unmarshalling metrics from file %s: %v.", inputPath, err)
	}

	// Load the trace
	client, capture, err := getGapisAndLoadCapture(ctx, gapisFlags, GapirFlags{}, trace, CaptureFileFlags{})
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
					err := ReportMetricResultsInTextFormat(categoryName, (*metrics)[j].Name, (*metrics)[j].Description, numColumns, columnNames, columnTypes, numRecords, dataStrings, dataLongs, dataDoubles, outputPath)
					if err != nil {
						return err
					}
				} else {
					categoryResults.Metrics = append(categoryResults.Metrics, CreateMetricResult((*metrics)[j].Name, (*metrics)[j].Description, numColumns, columnNames, columnTypes, numRecords, dataStrings, dataLongs, dataDoubles))
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

func RunInteractive(ctx context.Context, gapisFlags GapisFlags, trace string, inputPath string, outputPath string, format PerfettoOutputFormat) error {
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
	client, capture, err := getGapisAndLoadCapture(ctx, gapisFlags, GapirFlags{}, trace, CaptureFileFlags{})
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
	validResults := false
	var numColumns int
	var columnNames []string
	var columnTypes []string
	var numRecords uint64
	var dataStrings [][]string
	var dataLongs [][]int64
	var dataDoubles [][]float64
	var query string
	var cmdString string
	var cmdHistory []string
	var validCommand bool
	runCommandFromHistory := false

	for {
		validCommand = false
		if !runCommandFromHistory {
			fmt.Print(">> ")
			reader := bufio.NewReader(os.Stdin)
			cmdString, err = reader.ReadString('\n')
			if err != nil {
				return err
			}
		}
		runCommandFromHistory = false
		cmdHistory = append(cmdHistory, cmdString)
		if strings.HasPrefix(cmdString, "run") {
			validResults = false
			query = strings.TrimSuffix(strings.TrimPrefix(cmdString, "run"), "\n")
			numColumns, columnNames, columnTypes, numRecords, dataStrings, dataLongs, dataDoubles, err = RunQuery(ctx, client, capture, query)
			if err != nil {
				fmt.Println("Error while running query: ", err)
			} else {
				validResults = true
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
			validCommand = true
		} else if strings.HasPrefix(cmdString, "save") {
			if !validResults {
				fmt.Println("No valid query results available to save.")
			} else {
				cmdString = strings.TrimSpace(strings.TrimPrefix(cmdString, "save"))
				if strings.HasPrefix(cmdString, "text") || strings.HasPrefix(cmdString, "json") {
					validCommand = true
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
		} else if strings.HasPrefix(cmdString, "history") {
			cmdString = strings.TrimSpace(strings.TrimPrefix(cmdString, "history"))
			validCommand = true
			cmdFirst := 0
			cmdCountr, err := strconv.Atoi(cmdString)
			if err == nil {
				if len(cmdHistory)-cmdCountr > 0 {
					cmdFirst = len(cmdHistory) - cmdCountr - 1
				}
			}
			for i := cmdFirst; i < len(cmdHistory)-1; i++ {
				fmt.Print(i, "\t", cmdHistory[i])
			}
		} else if strings.HasPrefix(cmdString, "!") {
			cmdString = strings.TrimSpace(strings.TrimPrefix(cmdString, "!"))
			cmdId, err := strconv.Atoi(cmdString)
			if err == nil && cmdId < len(cmdHistory)-1 {
				validCommand = true
				cmdString = cmdHistory[cmdId]
				runCommandFromHistory = true
				fmt.Println(cmdString)
			}
		} else if strings.HasPrefix(cmdString, "quit") {
			break
		}
		if !validCommand {
			fmt.Println("Please use one of the following commands:")
			fmt.Println("\t>> run <sql-query>")
			fmt.Println("\t>> save text <filename>")
			fmt.Println("\t>> save json <filename>")
			fmt.Println("\t>> history [num-of-last-commands]")
			fmt.Println("\t>> !<command-id-returned-in-history>")
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

func ReportMetricResultsInTextFormat(categoryName string, metric string, description string, numColumns int, columnNames []string, columnTypes []string, numRecords uint64, dataStrings [][]string, dataLongs [][]int64, dataDoubles [][]float64, outputPath string) error {
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
	fmt.Fprintln(writer, "Category: ", categoryName)
	fmt.Fprintln(writer, "Metric name: ", metric)
	fmt.Fprintln(writer, "Description: ", description)
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

func CreateMetricResult(metric string, description string, numColumns int, columnNames []string, columnTypes []string, numRecords uint64, dataStrings [][]string, dataLongs [][]int64, dataDoubles [][]float64) (metricResults MetricResults) {
	metricResults.Name = metric
	metricResults.Description = description
	for i := uint64(0); i < numRecords; i++ {
		var resultsRecord MetricResultRecord
		for j := 0; j < numColumns; j++ {
			var result string
			switch columnTypes[j] {
			case "UNKNOWN":
				result = "NULL"
			case "STRING":
				result = dataStrings[j][i]
			case "LONG":
				result = strconv.FormatInt(dataLongs[j][i], 10)
			case "DOUBLE":
				result = strconv.FormatFloat(dataDoubles[j][i], 'E', -1, 64)
			}
			resultsRecord.Results = append(resultsRecord.Results, MetricResult{Name: columnNames[j], Value: result})
		}
		metricResults.Results = append(metricResults.Results, resultsRecord)
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
		return fmt.Errorf("Error converting the metrics results to Json. Please file a bug and provide as much as information as possible to address this issue. Error: %v", err)
	}
	if _, err := output.Write(jsonResult); err != nil {
		return fmt.Errorf("Error writing to the output file %s: %v.", outputPath, err)
	}
	output.Sync()
	return nil
}

func ListMetrics(ctx context.Context, inputPath string, categories map[string]struct{}) error {
	var byteValue []byte
	if inputPath != "" {
		metricsFile, err := os.Open(inputPath)
		if err != nil {
			app.Usage(ctx, "Cannot open file %s: %v.", inputPath, err)
		}
		defer metricsFile.Close()
		byteValue, _ = ioutil.ReadAll(metricsFile)
	} else {
		byteValue = []byte(PredefinedMetrics())
	}
	var metricsInfo MetricsInfo
	err := json.Unmarshal(byteValue, &metricsInfo)
	if err != nil {
		return fmt.Errorf("Error while unmarshalling metrics from file %s: %v.", inputPath, err)
	}

	metricInfo, _ := json.MarshalIndent(&metricsInfo.PrepQueries, "", "  ")
	fmt.Println("Initialization Queries:")
	os.Stdin.Write(metricInfo)
	os.Stdin.Sync()
	fmt.Println("\nMetrics:")
	listCustomCategories := len(categories) > 0
	for i := 0; i < len(metricsInfo.MetricCategories); i++ {
		categoryName := metricsInfo.MetricCategories[i].Name
		if listCustomCategories {
			if _, ok := categories[categoryName]; !ok {
				continue
			}
		}
		metricInfo, _ := json.MarshalIndent(&metricsInfo.MetricCategories[i], "", "  ")
		os.Stdin.Write(metricInfo)
		os.Stdin.WriteString("\n")
	}
	os.Stdin.Sync()
	return nil
}

func PredefinedMetrics() string {
	metricsJsonString := `{
	  "initialize": [{
	      "query": "CREATE VIEW [DRIVER_MEM_TRACK_IDS] AS SELECT * FROM counter_track WHERE name LIKE 'vulkan.mem.driver.%'"
	    },
	    {
	      "query": "CREATE VIEW [GPU_MEM_ALLOC_TRACK_IDS] AS SELECT * FROM counter_track WHERE name LIKE 'vulkan.mem.device.%allocation'"
	    },
	    {
	      "query": "CREATE VIEW [GPU_MEM_BIND_TRACK_IDS] AS SELECT * FROM counter_track WHERE name LIKE 'vulkan.mem.device.%bind'"
	    }
	  ],
	  "metrics": [{
	      "category": "CPU",
	      "metric": [{
	        "name": "AvgCPULoadPercentage",
	        "description": "Average CPU load in percentage over all CPU cores",
	        "queries": [{
	          "query": "SELECT (100 * CAST(SUM(dur) AS FLOAT) / (COUNT(DISTINCT cpu) * CAST(MAX(ts) - MIN(ts) AS FLOAT))) AS avg_cpu_load FROM sched WHERE utid !=  0"
	        }]
	      }]
	    },
	    {
	      "category": "Memory",
	      "metric": [{
	          "name": "AvgMemoryUsageBytes",
	          "description": "Average host memory usage in bytes",
	          "queries": [{
	              "query": "CREATE VIEW [TOTALMEM] AS SELECT value AS total_mem FROM counter WHERE track_id = (SELECT id FROM counter_track WHERE name = 'MemTotal') LIMIT 1"
	            },
	            {
	              "query": "CREATE VIEW [FREEMEM] AS SELECT AVG(value) AS avg_free_mem FROM counter WHERE track_id = (SELECT id FROM counter_track WHERE name = 'MemFree')"
	            },
	            {
	              "query": "SELECT CAST((total_mem - avg_free_mem) AS INT) AS avg_used_mem FROM [TOTALMEM], [FREEMEM]"
	            }
	          ]
	        },
	        {
	          "name": "MaxMemoryUsageBytes",
	          "description": "Max host memory usage in bytes",
	          "queries": [{
	              "query": "CREATE VIEW [TOTALMEM] AS SELECT value AS total_mem FROM counter WHERE track_id = (SELECT id FROM counter_track WHERE name = 'MemTotal') LIMIT 1"
	            },
	            {
	              "query": "CREATE VIEW [MINFREEMEM] AS SELECT MIN(value) AS min_free_mem FROM counter WHERE track_id = (SELECT id FROM counter_track WHERE name = 'MemFree')"
	            },
	            {
	              "query": "SELECT CAST((total_mem - min_free_mem) AS INT) AS max_used_mem FROM [TOTALMEM], [MINFREEMEM]"
	            }
	          ]
	        }
	      ]
	    },
	    {
	      "category": "Graphics.Memory.Driver",
	      "metric": [{
	          "name": "AvgDriverMemUsagePerScope",
	          "description": "Average memory usage by driver per allocation scope in bytes",
	          "queries": [{
	            "query": "SELECT REPLACE([DRIVER_MEM_TRACK_IDS].name, 'vulkan.mem.driver.scope.', '') AS scope, CAST(AVG(counter.value) AS INT) AS avg_mem_driver_bytes FROM counter LEFT JOIN [DRIVER_MEM_TRACK_IDS] WHERE counter.track_id = [DRIVER_MEM_TRACK_IDS].id GROUP BY counter.track_id"
	          }]
	        },
	        {
	          "name": "MaxDriverMemUsagePerScope",
	          "description": "Max memory usage by driver per allocation scope in bytes",
	          "queries": [{
	            "query": "SELECT REPLACE([DRIVER_MEM_TRACK_IDS].name, 'vulkan.mem.driver.scope.', '') AS scope, CAST (MAX(counter.value) AS INT) AS max_mem_driver_bytes FROM counter LEFT JOIN [DRIVER_MEM_TRACK_IDS] WHERE counter.track_id = [DRIVER_MEM_TRACK_IDS].id GROUP BY counter.track_id"
	          }]
	        }
	      ]
	    },
	    {
	      "category": "Graphics.Memory.GPU",
	      "metric": [{
	          "name": "AvgGPUMemAllocPerMemType",
	          "description": "Average GPU memory allocation per memory type",
	          "queries": [{
	            "query": "SELECT REPLACE(REPLACE([GPU_MEM_ALLOC_TRACK_IDS].name, 'vulkan.mem.device.memory.type.', ''), '.allocation', '') AS memory_type, CAST(AVG(counter.value) AS INT) AS avg_gpu_alloc_bytes FROM counter LEFT JOIN [GPU_MEM_ALLOC_TRACK_IDS] WHERE counter.track_id = [GPU_MEM_ALLOC_TRACK_IDS].id GROUP BY counter.track_id ORDER BY memory_type"
	          }]
	        },
	        {
	          "name": "MaxGPUMemAllocPerMemType",
	          "description": "Max GPU memory allocation per memory type",
	          "queries": [{
	            "query": "SELECT REPLACE(REPLACE([GPU_MEM_ALLOC_TRACK_IDS].name, 'vulkan.mem.device.memory.type.', ''), '.allocation', '') AS memory_type, CAST(MAX(counter.value) AS INT) AS max_gpu_alloc_bytes FROM counter LEFT JOIN [GPU_MEM_ALLOC_TRACK_IDS] WHERE counter.track_id = [GPU_MEM_ALLOC_TRACK_IDS].id GROUP BY counter.track_id ORDER BY memory_type"
	          }]
	        },
	        {
	          "name": "AvgGPUMemBindPerMemType",
	          "description": "Average GPU memory bound per memory type",
	          "queries": [{
	            "query": "SELECT REPLACE(REPLACE([GPU_MEM_BIND_TRACK_IDS].name, 'vulkan.mem.device.memory.type.', ''), '.bind', '') AS memory_type, CAST(AVG(counter.value) AS INT) AS avg_gpu_bound_bytes FROM counter LEFT JOIN [GPU_MEM_BIND_TRACK_IDS] WHERE counter.track_id = [GPU_MEM_BIND_TRACK_IDS].id GROUP BY counter.track_id ORDER BY memory_type"
	          }]
	        },
	        {
	          "name": "MaxGPUMemBindPerMemType",
	          "description": "Max GPU memory bound per memory type",
	          "queries": [{
	            "query": "SELECT REPLACE(REPLACE([GPU_MEM_BIND_TRACK_IDS].name, 'vulkan.mem.device.memory.type.', ''), '.bind', '') AS memory_type, CAST(MAX(counter.value) AS INT) AS max_gpu_bound_bytes FROM counter LEFT JOIN [GPU_MEM_BIND_TRACK_IDS] WHERE counter.track_id = [GPU_MEM_BIND_TRACK_IDS].id GROUP BY counter.track_id ORDER BY memory_type"
	          }]
	        }
	      ]
	    },
	    {
	      "category": "Graphics.Vulkan",
	      "metric": [{
	          "name": "NumCreateBufferEvents",
	          "description": "Number of create buffer events",
	          "queries": [{
	            "query": "SELECT COUNT(*) AS num_create_buffers FROM vulkan_memory_allocations WHERE source = 'GPU_BUFFER' AND operation = 'CREATE'"
	          }]
	        },
	        {
	          "name": "NumBindBufferEvents",
	          "description": "Number of bind buffer events",
	          "queries": [{
	            "query": "SELECT COUNT(*) AS num_bind_buffers FROM vulkan_memory_allocations WHERE source = 'GPU_BUFFER' AND operation = 'BIND'"
	          }]
	        },
	        {
	          "name": "SumBoundBuffersPerMemoryType",
	          "description": "Sum of bound buffer sizes per memory type in bytes",
	          "queries": [{
	            "query": "SELECT memory_type, SUM(memory_size) AS bound_buffers_in_bytes FROM vulkan_memory_allocations WHERE source = 'GPU_BUFFER' AND operation = 'BIND' GROUP BY memory_type"
	          }]
	        },
	        {
	          "name": "NumCreateImageEvents",
	          "description": "Number of create image events",
	          "queries": [{
	            "query": "SELECT COUNT(*) AS num_create_images FROM vulkan_memory_allocations WHERE source = 'GPU_IMAGE' AND operation = 'CREATE'"
	          }]
	        },
	        {
	          "name": "NumBindImageEvents",
	          "description": "Number of bind image events",
	          "queries": [{
	            "query": "SELECT COUNT(*) AS num_bind_images FROM vulkan_memory_allocations WHERE source = 'GPU_IMAGE' AND operation = 'BIND'"
	          }]
	        },
	        {
	          "name": "SumBoundImagesPerMemoryType",
	          "description": "Sum of bound image sizes per memory type in bytes",
	          "queries": [{
	            "query": "SELECT memory_type, SUM(memory_size) AS bound_images_in_bytes FROM vulkan_memory_allocations WHERE source = 'GPU_IMAGE' AND operation = 'BIND' GROUP BY memory_type"
	          }]
	        }
	      ]
	    }
	  ]
	}`
	return metricsJsonString
}
