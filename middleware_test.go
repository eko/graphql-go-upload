package upload

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

type GraphQLRequestFile struct {
	Filename string `json:"filename"`
	Filepath string `json:"filepath"`
	MIMEType string `json:"mimetype"`
}

type GraphQLRequestVariables struct {
	File  GraphQLRequestFile `json:"file"`
	Title string             `json:"title"`
}

type GraphQLRequest struct {
	Variables GraphQLRequestVariables `json:"variables"`
}

func openTestTempFile(filename string) *os.File {
	file, err := os.Open(filename)
	if err != nil {
		panic(err)
	}

	return file
}

func TestHandlerWhenSuccess(t *testing.T) {
	// Given
	filePath, _ := os.Getwd()
	filePath += "/middleware.go"

	file := openTestTempFile(filePath)
	defer file.Close()

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	fw, err := writer.CreateFormFile("file", "middleware.go")
	if err != nil {
		panic(err)
	}

	if _, err = io.Copy(fw, file); err != nil {
		panic(err)
	}

	// Add the other fields
	_ = writer.WriteField("operations", "{ \"query\": \"mutation DoUpload($file: Upload!, $title: String) { upload(file: $file, title: $title) { code } }\", \"variables\": { \"file\": null, \"title\": \"my test title\" } }")
	_ = writer.WriteField("map", "{ \"file\": [\"variables.file\"], \"title\": [\"variables.title\"] }")
	_ = writer.WriteField("title", "my test title")

	writer.Close()

	req, _ := http.NewRequest("POST", "/test", &requestBody)
	req.Header.Add("Content-Type", writer.FormDataContentType())

	w := httptest.NewRecorder()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Do nothing, this is a fake HTTP request handler: goal is to test our upload middleware.
	})

	// When
	nextMidware := Handler(next)
	nextMidware.ServeHTTP(w, req)

	// Then
	body, _ := ioutil.ReadAll(req.Body)

	var data GraphQLRequest
	if err := json.Unmarshal(body, &data); err != nil {
		panic(err)
	}

	// Should render GraphQL variables entry containing parameters and file data.
	// Also, the content-type of the request have to be switched to "application/json".
	assert.Equal(t, w.Header().Get("Content-Type"), "application/json")

	assert.NotEmpty(t, data.Variables.File.Filepath)
	assert.Equal(t, data.Variables.File.Filename, "middleware.go")
	assert.Equal(t, data.Variables.File.MIMEType, "text/plain")

	assert.Equal(t, data.Variables.Title, "my test title")
}

func TestIsMiddlewareSupportedWhenSupported(t *testing.T) {
	// Given
	req, _ := http.NewRequest("POST", "/test", nil)
	req.Header.Add("Content-Type", "multipart/form-data")

	// When
	isSupported := isMiddlewareSupported(req)

	// Then
	assert.True(t, isSupported)
}

func TestIsMiddlewareSupportedWhenNotSupported(t *testing.T) {
	// Given
	req, _ := http.NewRequest("POST", "/test", nil)
	req.Header.Add("Content-Type", "application/json")

	// When
	isSupported := isMiddlewareSupported(req)

	// Then
	assert.False(t, isSupported)
}
