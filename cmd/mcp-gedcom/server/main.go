package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/flecoufle/mcp-gedcom/internal/gedcom"
	"github.com/flecoufle/mcp-gedcom/internal/mcp"
	"github.com/flecoufle/mcp-gedcom/internal/tools"
)

func main() {
	gedcomFile := flag.String("gedcom-file", "", "GEDCOM file name (used with --gedcom-path or in current directory)")
	gedcomPath := flag.String("gedcom-path", "", "Directory to search for GEDCOM files")
	flag.Parse()

	gedcomFilePath, err := resolveGedcomPath(*gedcomFile, *gedcomPath)
	if err != nil {
		log.Fatal(err)
	}

	if err := gedcom.Init(gedcomFilePath); err != nil {
		log.Fatalf("Failed to load GEDCOM file: %v", err)
	}

	log.Printf("Loaded GEDCOM file: %s", gedcomFilePath)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	tracker := mcp.NewRequestTracker()

	scanner := bufio.NewScanner(os.Stdin)
	lines := make(chan string)

	go func() {
		for scanner.Scan() {
			lines <- scanner.Text()
		}
		close(lines)
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case line, ok := <-lines:
			if !ok {
				return
			}
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			req, err := mcp.ParseRequest([]byte(line))
			if err != nil {
				sendError(nil, mcp.InvalidRequest, err.Error())
				continue
			}

			if tracker.IsIDUsed(req.ID) {
				sendError(req.ID, mcp.DuplicateIDError, "id already used in this session")
				continue
			}

			resp := handleRequest(req)
			tracker.MarkIDUsed(req.ID)

			if resp != nil {
				fmt.Println(string(mcp.MarshalResponse(*resp)))
			}
		}
	}
}

func resolveGedcomPath(gedcomFile, gedcomPath string) (string, error) {
	switch {
	case gedcomFile != "":
		if _, err := os.Stat(gedcomFile); os.IsNotExist(err) {
			return "", fmt.Errorf("GEDCOM file not found: %s", gedcomFile)
		}
		absPath, _ := filepath.Abs(gedcomFile)
		return absPath, nil

	case gedcomPath != "":
		return resolveFromPath(gedcomPath)

	default:
		return resolveFromPath(".")
	}
}

func resolveFromPath(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("cannot read directory: %s", dir)
	}

	var gedFiles []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".ged") {
			gedFiles = append(gedFiles, e.Name())
		}
	}

	if len(gedFiles) == 0 {
		if dir == "." {
			return "", fmt.Errorf("no GEDCOM file found in current directory")
		}
		return "", fmt.Errorf("no .ged file found in GEDCOM path: %s", dir)
	}

	if len(gedFiles) > 1 {
		return "", fmt.Errorf("multiple .ged files found in GEDCOM path (%s): %s", dir, strings.Join(gedFiles, ", "))
	}

	absDir, _ := filepath.Abs(dir)
	return filepath.Join(absDir, gedFiles[0]), nil
}

func handleRequest(req *mcp.JSONRPCRequest) *mcp.JSONRPCResponse {
	id := req.ID

	switch req.Method {
	case "initialize":
		return &mcp.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Result:  mcp.NewInitializeResult("mcp-gedcom", "1.0.1"),
		}

	case "tools/list":
		tools := []mcp.Tool{
			{
				Name:        "search_person",
				Description: "Search for individuals in full name, and optionally with approximate birth year. (2 years)",
				InputSchema: mcp.InputSchema{
					Type: "object",
					Properties: map[string]mcp.Property{
						"pattern":   {Type: "string", Description: "Full name pattern to search for"},
						"birthYear": {Type: "number", Description: "Approximate birth year (optional, 2 years)"},
						"givenName": {Type: "boolean", Description: "Show given name instead of usage name (default: false)"},
					},
					Required: []string{"pattern"},
				},
			},
			{
				Name:        "search_surnames",
				Description: "Search surnames by pattern (partial match, case-insensitive) with pagination.",
				InputSchema: mcp.InputSchema{
					Type: "object",
					Properties: map[string]mcp.Property{
						"pattern": {Type: "string", Description: "Surname pattern to search for"},
						"offset":  {Type: "number", Description: "Pagination offset (default: 0)"},
						"limit":   {Type: "number", Description: "Max results (default: 20)"},
					},
					Required: []string{"pattern"},
				},
			},
			{
				Name:        "get_person_details",
				Description: "Get full details for an individual by Individual ID",
				InputSchema: mcp.InputSchema{
					Type: "object",
					Properties: map[string]mcp.Property{
						"id":           {Type: "string", Description: "Individual ID (e.g., I1, @I1@)"},
						"withSpouse":   {Type: "boolean", Description: "Include spouse info (default: true)"},
						"withChildren": {Type: "boolean", Description: "Include children info (default: true)"},
						"givenName":    {Type: "boolean", Description: "Show given name instead of usage name (default: false)"},
					},
					Required: []string{"id"},
				},
			},
			{
				Name:        "get_relatives",
				Description: "Get family relationships (spouses and parents) for an individual",
				InputSchema: mcp.InputSchema{
					Type: "object",
					Properties: map[string]mcp.Property{
						"id":        {Type: "string", Description: "Individual ID (e.g., I1, @I1@)"},
						"givenName": {Type: "boolean", Description: "Show given name instead of usage name (default: false)"},
					},
					Required: []string{"id"},
				},
			},
			{
				Name:        "get_family_details",
				Description: "Get full details for a family by Family ID",
				InputSchema: mcp.InputSchema{
					Type: "object",
					Properties: map[string]mcp.Property{
						"id":        {Type: "string", Description: "Family ID (e.g., F0000, @F0000@)"},
						"givenName": {Type: "boolean", Description: "Show given name instead of usage name (default: false)"},
					},
					Required: []string{"id"},
				},
			},
			{
				Name:        "get_children",
				Description: "Get children of an individual with pagination",
				InputSchema: mcp.InputSchema{
					Type: "object",
					Properties: map[string]mcp.Property{
						"id":        {Type: "string", Description: "Individual ID (e.g., I1)"},
						"offset":    {Type: "number", Description: "Pagination offset (default: 0)"},
						"limit":     {Type: "number", Description: "Max results (default: 20)"},
						"givenName": {Type: "boolean", Description: "Show given name instead of usage name (default: false)"},
					},
					Required: []string{"id"},
				},
			},
			{
				Name:        "get_parents",
				Description: "Get parents of an individual",
				InputSchema: mcp.InputSchema{
					Type: "object",
					Properties: map[string]mcp.Property{
						"id":        {Type: "string", Description: "Individual ID (e.g., I1)"},
						"givenName": {Type: "boolean", Description: "Show given name instead of usage name (default: false)"},
					},
					Required: []string{"id"},
				},
			},
			{
				Name:        "search_by_date_range",
				Description: "Search individuals by birth or death year range",
				InputSchema: mcp.InputSchema{
					Type: "object",
					Properties: map[string]mcp.Property{
						"startYear": {Type: "number", Description: "Start year"},
						"endYear":   {Type: "number", Description: "End year"},
						"event":     {Type: "string", Description: "Event type: 'birth' or 'death'"},
						"givenName": {Type: "boolean", Description: "Show given name instead of usage name (default: false)"},
					},
					Required: []string{"startYear", "endYear"},
				},
			},
			{
				Name:        "get_ancestors",
				Description: "Get ancestors of an individual up to N generations",
				InputSchema: mcp.InputSchema{
					Type: "object",
					Properties: map[string]mcp.Property{
						"id":          {Type: "string", Description: "Individual ID (e.g., I1)"},
						"generations": {Type: "number", Description: "Number of generations (default: 3)"},
						"givenName":   {Type: "boolean", Description: "Show given name instead of usage name (default: false)"},
					},
					Required: []string{"id"},
				},
			},
			{
				Name:        "get_descendants",
				Description: "Get descendants of an individual up to N generations",
				InputSchema: mcp.InputSchema{
					Type: "object",
					Properties: map[string]mcp.Property{
						"id":          {Type: "string", Description: "Individual ID (e.g., I1)"},
						"generations": {Type: "number", Description: "Number of generations (default: 3)"},
						"givenName":   {Type: "boolean", Description: "Show given name instead of usage name (default: false)"},
					},
					Required: []string{"id"},
				},
			},
			{
				Name:        "get_statistics",
				Description: "Get statistics about the GEDCOM file",
				InputSchema: mcp.InputSchema{
					Type:       "object",
					Properties: map[string]mcp.Property{},
				},
			},
			{
				Name:        "find_relationship_path",
				Description: "Calculates the shortest kinship path between two individuals and defines their relationship (e.g., first cousin, uncle).",
				InputSchema: mcp.InputSchema{
					Type: "object",
					Properties: map[string]mcp.Property{
						"id1":           {Type: "string", Description: "ID of the first person"},
						"id2":           {Type: "string", Description: "ID of the second person"},
						"AncestorsOnly": {Type: "boolean", Description: "Only search through parents/ancestors, disabling spouse and children (default: true)"},
						"givenName":     {Type: "boolean", Description: "Show given name instead of usage name (default: false)"},
					},
					Required: []string{"id1", "id2"},
				},
			},
			{
				Name:        "find_all_relationships",
				Description: "Finds ALL kinship relationships between two individuals, including consanguinity. Returns all distinct relationship paths with a global kinship coefficient.",
				InputSchema: mcp.InputSchema{
					Type: "object",
					Properties: map[string]mcp.Property{
						"id1":        {Type: "string", Description: "ID of the first person"},
						"id2":        {Type: "string", Description: "ID of the second person"},
						"maxDepth":   {Type: "number", Description: "Maximum generations to search (default: 10)"},
						"maxResults": {Type: "number", Description: "Maximum number of relationship links to return (default: 5)"},
						"givenName":  {Type: "boolean", Description: "Show given name instead of usage name (default: false)"},
					},
					Required: []string{"id1", "id2"},
				},
			},
			{
				Name:        "load_gedcom_file",
				Description: "Load a new GEDCOM file at runtime, replacing current data",
				InputSchema: mcp.InputSchema{
					Type: "object",
					Properties: map[string]mcp.Property{
						"path": {Type: "string", Description: "Path to the GEDCOM file"},
					},
					Required: []string{"path"},
				},
			},
		}
		return &mcp.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Result:  mcp.NewListToolsResult(tools),
		}

	case "tools/call":
		var params struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments,omitempty"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return &mcp.JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      id,
				Result:  mcp.NewCallToolError("Invalid parameters: " + err.Error()),
			}
		}

		result, isError := callTool(params.Name, params.Arguments)
		if isError {
			return &mcp.JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      id,
				Result:  result,
			}
		}
		return &mcp.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Result:  result,
		}

	default:
		return sendError(id, -32601, "Method not found: "+req.Method)
	}
}

func callTool(name string, args json.RawMessage) (*mcp.CallToolResult, bool) {
	switch name {
	case "search_person":
		var a struct {
			Pattern   string `json:"pattern"`
			BirthYear int    `json:"birthYear"`
			GivenName *bool  `json:"givenName"`
		}
		if err := json.Unmarshal(args, &a); err != nil {
			return mcp.NewCallToolError("Invalid parameters: " + err.Error()), true
		}
		if strings.TrimSpace(a.Pattern) == "" {
			return mcp.NewCallToolError("Missing required parameter: pattern"), true
		}
		useGiven := false
		if a.GivenName != nil {
			useGiven = *a.GivenName
		}
		result, err := tools.HandleSearchPerson(a.Pattern, a.BirthYear, useGiven)
		if err != nil {
			return mcp.NewCallToolError(err.Error()), true
		}
		return formatSearchByNameResult(result), false

	case "search_surnames":
		var a struct {
			Pattern string `json:"pattern"`
			Offset  int    `json:"offset"`
			Limit   int    `json:"limit"`
		}
		if err := json.Unmarshal(args, &a); err != nil {
			return mcp.NewCallToolError("Invalid parameters: " + err.Error()), true
		}
		if strings.TrimSpace(a.Pattern) == "" {
			return mcp.NewCallToolError("Missing required parameter: pattern"), true
		}
		if a.Limit == 0 {
			a.Limit = 20
		}
		result, err := tools.HandleSearchSurnames(a.Pattern, a.Offset, a.Limit)
		if err != nil {
			return mcp.NewCallToolError(err.Error()), true
		}
		return formatSearchSurnamesResult(result), false

	case "get_person_details":
		var a struct {
			ID           string `json:"id"`
			WithSpouse   *bool  `json:"withSpouse"`
			WithChildren *bool  `json:"withChildren"`
			GivenName    *bool  `json:"givenName"`
		}
		if err := json.Unmarshal(args, &a); err != nil {
			return mcp.NewCallToolError("Invalid parameters: " + err.Error()), true
		}
		if strings.TrimSpace(a.ID) == "" {
			return mcp.NewCallToolError("Missing required parameter: id"), true
		}
		withSpouse := true
		if a.WithSpouse != nil {
			withSpouse = *a.WithSpouse
		}
		withChildren := true
		if a.WithChildren != nil {
			withChildren = *a.WithChildren
		}
		useGiven := false
		if a.GivenName != nil {
			useGiven = *a.GivenName
		}
		result, err := tools.HandleGetPersonDetails(a.ID, withSpouse, withChildren, useGiven)
		if err != nil {
			return mcp.NewCallToolError(err.Error()), true
		}
		return formatGetPersonalDetailsResult(result), false

	case "get_relatives":
		var a struct {
			ID        string `json:"id"`
			GivenName *bool  `json:"givenName"`
		}
		if err := json.Unmarshal(args, &a); err != nil {
			return mcp.NewCallToolError("Invalid parameters: " + err.Error()), true
		}
		if strings.TrimSpace(a.ID) == "" {
			return mcp.NewCallToolError("Missing required parameter: id"), true
		}
		useGiven := false
		if a.GivenName != nil {
			useGiven = *a.GivenName
		}
		result, err := tools.HandleGetRelatives(a.ID, useGiven)
		if err != nil {
			return mcp.NewCallToolError(err.Error()), true
		}
		return formatGetRelativesResult(result), false

	case "get_family_details":
		var a struct {
			ID        string `json:"id"`
			GivenName *bool  `json:"givenName"`
		}
		if err := json.Unmarshal(args, &a); err != nil {
			return mcp.NewCallToolError("Invalid parameters: " + err.Error()), true
		}
		if strings.TrimSpace(a.ID) == "" {
			return mcp.NewCallToolError("Missing required parameter: id"), true
		}
		useGiven := false
		if a.GivenName != nil {
			useGiven = *a.GivenName
		}
		result, err := tools.HandleGetFamilyDetails(a.ID, useGiven)
		if err != nil {
			return mcp.NewCallToolError(err.Error()), true
		}
		return formatGetFamilyByIDResult(result), false

	case "get_children":
		var a struct {
			ID        string `json:"id"`
			Offset    int    `json:"offset"`
			Limit     int    `json:"limit"`
			GivenName *bool  `json:"givenName"`
		}
		if err := json.Unmarshal(args, &a); err != nil {
			return mcp.NewCallToolError("Invalid parameters: " + err.Error()), true
		}
		if strings.TrimSpace(a.ID) == "" {
			return mcp.NewCallToolError("Missing required parameter: id"), true
		}
		if a.Limit == 0 {
			a.Limit = 20
		}
		useGiven := false
		if a.GivenName != nil {
			useGiven = *a.GivenName
		}
		result, err := tools.HandleGetChildren(a.ID, a.Offset, a.Limit, useGiven)
		if err != nil {
			return mcp.NewCallToolError(err.Error()), true
		}
		return formatGetChildrenResult(result), false

	case "get_parents":
		var a struct {
			ID        string `json:"id"`
			GivenName *bool  `json:"givenName"`
		}
		if err := json.Unmarshal(args, &a); err != nil {
			return mcp.NewCallToolError("Invalid parameters: " + err.Error()), true
		}
		if strings.TrimSpace(a.ID) == "" {
			return mcp.NewCallToolError("Missing required parameter: id"), true
		}
		useGiven := false
		if a.GivenName != nil {
			useGiven = *a.GivenName
		}
		result, err := tools.HandleGetParents(a.ID, useGiven)
		if err != nil {
			return mcp.NewCallToolError(err.Error()), true
		}
		return formatGetParentsResult(result), false

	case "search_by_date_range":
		var a struct {
			StartYear int    `json:"startYear"`
			EndYear   int    `json:"endYear"`
			Event     string `json:"event"`
			GivenName *bool  `json:"givenName"`
		}
		if err := json.Unmarshal(args, &a); err != nil {
			return mcp.NewCallToolError("Invalid parameters: " + err.Error()), true
		}
		if a.StartYear == 0 {
			return mcp.NewCallToolError("Missing required parameter: startYear"), true
		}
		if a.EndYear == 0 {
			return mcp.NewCallToolError("Missing required parameter: endYear"), true
		}
		if a.Event == "" {
			a.Event = "birth"
		}
		useGiven := false
		if a.GivenName != nil {
			useGiven = *a.GivenName
		}
		result, err := tools.HandleSearchByDateRange(a.StartYear, a.EndYear, a.Event, useGiven)
		if err != nil {
			return mcp.NewCallToolError(err.Error()), true
		}
		return formatSearchByDateRangeResult(result), false

	case "get_ancestors":
		var a struct {
			ID          string `json:"id"`
			Generations int    `json:"generations"`
			GivenName   *bool  `json:"givenName"`
		}
		if err := json.Unmarshal(args, &a); err != nil {
			return mcp.NewCallToolError("Invalid parameters: " + err.Error()), true
		}
		if strings.TrimSpace(a.ID) == "" {
			return mcp.NewCallToolError("Missing required parameter: id"), true
		}
		if a.Generations == 0 {
			a.Generations = 3
		}
		useGiven := false
		if a.GivenName != nil {
			useGiven = *a.GivenName
		}
		result, err := tools.HandleGetAncestors(a.ID, a.Generations, useGiven)
		if err != nil {
			return mcp.NewCallToolError(err.Error()), true
		}
		return formatGetAncestorsResult(result), false

	case "get_descendants":
		var a struct {
			ID          string `json:"id"`
			Generations int    `json:"generations"`
			GivenName   *bool  `json:"givenName"`
		}
		if err := json.Unmarshal(args, &a); err != nil {
			return mcp.NewCallToolError("Invalid parameters: " + err.Error()), true
		}
		if strings.TrimSpace(a.ID) == "" {
			return mcp.NewCallToolError("Missing required parameter: id"), true
		}
		if a.Generations == 0 {
			a.Generations = 3
		}
		useGiven := false
		if a.GivenName != nil {
			useGiven = *a.GivenName
		}
		result, err := tools.HandleGetDescendants(a.ID, a.Generations, useGiven)
		if err != nil {
			return mcp.NewCallToolError(err.Error()), true
		}
		return formatGetDescendantsResult(result), false

	case "find_relationship_path":
		var a struct {
			ID1           string `json:"id1"`
			ID2           string `json:"id2"`
			AncestorsOnly *bool  `json:"AncestorsOnly"`
			GivenName     *bool  `json:"givenName"`
		}
		if err := json.Unmarshal(args, &a); err != nil {
			return mcp.NewCallToolError("Invalid parameters: " + err.Error()), true
		}
		if strings.TrimSpace(a.ID1) == "" {
			return mcp.NewCallToolError("Missing required parameter: id1"), true
		}
		if strings.TrimSpace(a.ID2) == "" {
			return mcp.NewCallToolError("Missing required parameter: id2"), true
		}
		ancestorsOnly := true
		if a.AncestorsOnly != nil {
			ancestorsOnly = *a.AncestorsOnly
		}
		useGiven := false
		if a.GivenName != nil {
			useGiven = *a.GivenName
		}
		result, err := tools.HandleFindRelationshipPath(a.ID1, a.ID2, ancestorsOnly, useGiven)
		if err != nil {
			return mcp.NewCallToolError(err.Error()), true
		}
		return formatFindRelationshipPathResult(result), false

	case "find_all_relationships":
		var a struct {
			ID1        string `json:"id1"`
			ID2        string `json:"id2"`
			MaxDepth   int    `json:"maxDepth"`
			MaxResults *int   `json:"maxResults"`
			GivenName  *bool  `json:"givenName"`
		}
		if err := json.Unmarshal(args, &a); err != nil {
			return mcp.NewCallToolError("Invalid parameters: " + err.Error()), true
		}
		if strings.TrimSpace(a.ID1) == "" {
			return mcp.NewCallToolError("Missing required parameter: id1"), true
		}
		if strings.TrimSpace(a.ID2) == "" {
			return mcp.NewCallToolError("Missing required parameter: id2"), true
		}
		if a.MaxDepth <= 0 {
			a.MaxDepth = 15
		}
		maxResults := 10
		if a.MaxResults != nil {
			maxResults = *a.MaxResults
		}
		useGiven := false
		if a.GivenName != nil {
			useGiven = *a.GivenName
		}
		links, coeff, err := tools.HandleFindAllRelationships(a.ID1, a.ID2, a.MaxDepth, maxResults)
		if err != nil {
			return mcp.NewCallToolError(err.Error()), true
		}
		return formatFindAllRelationshipsResult(links, coeff, useGiven), false

	case "get_statistics":
		result := tools.HandleGetStatistics()
		return formatGetStatisticsResult(result), false

	case "load_gedcom_file":
		var a struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal(args, &a); err != nil {
			return mcp.NewCallToolError("Invalid parameters: " + err.Error()), true
		}
		if strings.TrimSpace(a.Path) == "" {
			return mcp.NewCallToolError("Missing required parameter: path"), true
		}
		result, err := tools.HandleLoadGedcomFile(a.Path)
		if err != nil {
			return mcp.NewCallToolError(err.Error()), true
		}
		return formatLoadGedcomFileResult(result), false
	}
	return mcp.NewCallToolError("Unknown tool: " + name), true
}

func sendError(id any, code int, msg string) *mcp.JSONRPCResponse {
	return &mcp.JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   mcp.NewRPCError(code, msg),
	}
}
