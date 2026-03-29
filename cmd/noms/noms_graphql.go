// Copyright 2025 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/attic-labs/kingpin"
	"github.com/attic-labs/noms/cmd/util"
	"github.com/attic-labs/noms/go/config"
	"github.com/attic-labs/noms/go/ngql"
	"github.com/attic-labs/noms/go/types"
)

func nomsGraphQL(noms *kingpin.Application) (*kingpin.CmdClause, util.KingpinHandler) {
	cmd := noms.Command("graphql", "Execute GraphQL queries against Noms datasets.")
	query := cmd.Arg("query", "GraphQL query string (e.g., '{root{hash}}')").Required().String()
	dataset := cmd.Arg("dataset", "Dataset to query").Default("").String()

	return cmd, func(input string) int {
		return runGraphQL(*query, *dataset, outputFormat)
	}
}

func runGraphQL(queryStr, datasetName, format string) int {
	cfg := config.NewResolver()
	db, err := cfg.GetDatabase("")
	if err != nil {
		if format == "json" {
			json.NewEncoder(os.Stdout).Encode(map[string]string{
				"error": "no database initialized: " + err.Error(),
			})
		} else {
			fmt.Println("No Noms database found. Run 'noms init' to initialize.")
		}
		return 1
	}
	defer db.Close()

	var rootValue types.Value

	if datasetName == "" {
		// Use database datasets as root
		datasets := db.Datasets()
		rootValue = datasets
	} else {
		// Get specific dataset
		dataset := db.GetDataset(datasetName)
		if !dataset.HasHead() {
			if format == "json" {
				json.NewEncoder(os.Stdout).Encode(map[string]string{
					"error": "dataset not found or empty: " + datasetName,
				})
			} else {
				fmt.Printf("Dataset not found or empty: %s\n", datasetName)
			}
			return 1
		}
		rootValue = dataset.Head()
	}

	if rootValue == nil {
		if format == "json" {
			json.NewEncoder(os.Stdout).Encode(map[string]string{
				"error": "no data found",
			})
		} else {
			fmt.Println("No data found")
		}
		return 1
	}

	// Execute GraphQL query
	buf := &bytes.Buffer{}
	ngql.Query(rootValue, queryStr, db, buf)

	// Output result
	if format == "json" {
		fmt.Print(buf.String())
	} else {
		// Pretty print JSON
		var result map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			fmt.Print(buf.String())
			return 0
		}

		// Check for errors
		if errors, ok := result["errors"].([]interface{}); ok && len(errors) > 0 {
			fmt.Println("GraphQL Errors:")
			for _, err := range errors {
				if errMap, ok := err.(map[string]interface{}); ok {
					fmt.Printf("  - %v\n", errMap["message"])
				}
			}
			fmt.Println()
		}

		// Print data if present
		if data, ok := result["data"].(map[string]interface{}); ok {
			fmt.Println("Result:")
			fmt.Println("======")
			printGraphQLData(data, 0)
		}
	}

	return 0
}

func printGraphQLData(data map[string]interface{}, indent int) {
	prefix := strings.Repeat("  ", indent)
	for key, value := range data {
		switch v := value.(type) {
		case map[string]interface{}:
			fmt.Printf("%s%s:\n", prefix, key)
			printGraphQLData(v, indent+1)
		case []interface{}:
			fmt.Printf("%s%s: [\n", prefix, key)
			for _, item := range v {
				if itemMap, ok := item.(map[string]interface{}); ok {
					printGraphQLData(itemMap, indent+2)
					fmt.Println()
				} else {
					fmt.Printf("%s  %v\n", prefix, item)
				}
			}
			fmt.Printf("%s]\n", prefix)
		default:
			fmt.Printf("%s%s: %v\n", prefix, key, v)
		}
	}
}
