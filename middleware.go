package upload

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type postedFiles func(key string) (multipart.File, *multipart.FileHeader, error)
type graphqlParams struct {
	Variables  interface{}            `json:"variables"`
	Query      interface{}            `json:"query"`
	Operations map[string]interface{} `json:"operations"`
	Map        map[string][]string    `json:"map"`
}

type fileData struct {
	Fields        interface{}
	Files         postedFiles
	MapEntryIndex string
	EntryPaths    []string
}

var (
	mapEntries  map[string][]string
	operations  map[string]interface{}
	fileChannel = make(chan fileData)
	wg          sync.WaitGroup
)

// Handler is the middleware function that retrieves the incoming HTTP request and
// in case it is a POST and multipart/form-data request, re-maps the field values
// given in the GraphQL format and saves uploaded files.
//
// Here is how to implement the middleware handler (see upload.Handler use below):
//
//  h := handler.GraphQL{
//      Schema: graphql.MustParseSchema(schema.String(), root, graphql.MaxParallelism(maxParallelism), graphql.MaxDepth(maxDepth)),
//      Handler: handler.NewHandler(conf, &m),
//  }
//
//  mux := mux.NewRouter()
//  mux.Handle("/graphql", upload.Handler(h))
//
//  s := &http.Server{
//      Addr:    ":8000",
//      Handler: mux,
//  }
//
func Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isMiddlewareSupported(r) {
			next.ServeHTTP(w, r)
			return
		}

		r.ParseMultipartForm((1 << 20) * 64)
		m := r.PostFormValue("map")
		if &m == nil {
			http.Error(w, "Missing map field parameter", http.StatusBadRequest)
			return
		}

		o := r.PostFormValue("operations")
		if &o == nil {
			http.Error(w, "Missing operations field parameter", http.StatusBadRequest)
			return
		}

		err := json.Unmarshal([]byte(o), &operations)
		if err != nil {
			http.Error(w, "Cannot unmarshal operations: malformed query", http.StatusBadRequest)
			return
		}

		err = json.Unmarshal([]byte(m), &mapEntries)
		if err != nil {
			http.Error(w, "Cannot unmarshal map entries: malformed query", http.StatusBadRequest)
			return
		}

		mapOperations(mapEntries, operations, r)

		graphqlParams := graphqlParams{
			Variables:  operations["variables"],
			Query:      operations["query"],
			Operations: operations,
			Map:        mapEntries,
		}

		body, err := json.Marshal(graphqlParams)
		if err == nil {
			r.Body = ioutil.NopCloser(bytes.NewReader(body))
			w.Header().Set("Content-Type", "application/json")
		}

		next.ServeHTTP(w, r)
	})
}

func isMiddlewareSupported(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	contentType := r.Header.Get("Content-Type")
	mediatype, _, _ := mime.ParseMediaType(contentType)
	if contentType == "" || mediatype != "multipart/form-data" {
		return false
	}

	return true
}

func mapOperations(mapEntries map[string][]string, operations map[string]interface{}, r *http.Request) {
	for idx, mapEntry := range mapEntries {
		for _, entry := range mapEntry {
			entryPaths := strings.Split(entry, ".")
			fields := findFields(operations, entryPaths[:len(entryPaths)-1])

			if value := r.PostForm.Get(idx); value != "" { // Form field values
				entryPaths := strings.Split(entry, ".")
				operations[entryPaths[0]].(map[string]interface{})[entryPaths[1]] = value
			} else { // Try to catch an uploaded file
				wg.Add(1)
				go func() {
					defer wg.Done()
					mapTemporaryFileToOperations()
				}()

				fileChannel <- fileData{
					Fields:        fields,
					Files:         r.FormFile,
					MapEntryIndex: idx,
					EntryPaths:    entryPaths,
				}
			}
		}
	}
	wg.Wait()
}

func findFields(operations interface{}, entryPaths []string) map[string]interface{} {
	for i := 0; i < len(entryPaths); i++ {
		if arr, ok := operations.([]map[string]interface{}); ok {
			operations = arr[i]

			return findFields(operations, entryPaths)
		} else if op, ok := operations.(map[string]interface{}); ok {
			operations = op[entryPaths[i]]
		}
	}

	return operations.(map[string]interface{})
}

func mapTemporaryFileToOperations() error {
	params := <-fileChannel
	file, handle, err := params.Files(params.MapEntryIndex)
	if err != nil {
		return fmt.Errorf("Could not access multipart file. Reason: %v", err)
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return fmt.Errorf("Could not read multipart file. Reason: %v", err)
	}

	f, err := ioutil.TempFile(os.TempDir(), fmt.Sprintf("graphqlupload-*%s", filepath.Ext(handle.Filename)))
	if err != nil {
		return fmt.Errorf("Unable to create temporary file. Reason: %v", err)
	}

	_, err = f.Write(data)
	if err != nil {
		return fmt.Errorf("Could not write temporary file. Reason: %v", err)
	}

	upload := &GraphQLUpload{
		MIMEType: handle.Header.Get("Content-Type"),
		Filename: handle.Filename,
		Filepath: f.Name(),
	}

	if op, ok := params.Fields.(map[string]interface{}); ok {
		op[params.EntryPaths[len(params.EntryPaths)-1]] = upload
	}

	return nil
}
