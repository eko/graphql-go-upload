package upload

import (
	"bufio"
	"io/ioutil"
	"os"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestImplementsGraphQLType(t *testing.T) {
	// Given
	upload := GraphQLUpload{}

	// When - Then
	assert.True(t, upload.ImplementsGraphQLType("Upload"))
	assert.False(t, upload.ImplementsGraphQLType("AnotherThing"))
}

func TestUnmarshalGraphQLWhenMapString(t *testing.T) {
	// Given
	upload := GraphQLUpload{}

	input := map[string]interface{}{
		"filename": "my-test.jpg",
		"mimetype": "image/jpeg",
		"filepath": "/tmp/my-test.jpg",
	}

	// When
	err := upload.UnmarshalGraphQL(input)

	// Then
	assert.Nil(t, err)

	assert.Equal(t, upload.Filename, input["filename"])
	assert.Equal(t, upload.MIMEType, input["mimetype"])
	assert.Equal(t, upload.Filepath, input["filepath"])
}

func TestUnmarshalGraphQLWhenAnotherType(t *testing.T) {
	// Given
	upload := GraphQLUpload{}

	input := map[int]interface{}{
		1: "my-test.jpg",
		2: "image/jpeg",
		3: "/tmp/my-test.jpg",
	}

	// When
	err := upload.UnmarshalGraphQL(input)

	// Then
	assert.Equal(t, "Cannot unmarshal received type as a GraphQLUpload type", err.Error())
}

func TestGetReaderWhenFileExists(t *testing.T) {
	// Given
	f, err := ioutil.TempFile("", "testfile")
	if err != nil {
		panic(err)
	}
	defer syscall.Unlink(f.Name())

	upload := GraphQLUpload{
		Filepath: f.Name(),
	}

	// When
	reader, err := upload.GetReader()

	// Then
	assert.IsType(t, new(bufio.Reader), reader)
	assert.Nil(t, err)
}

func TestGetReaderWhenFileDoesNotExists(t *testing.T) {
	// Given
	upload := GraphQLUpload{
		Filepath: "unknown-file.jpeg",
	}

	// When
	reader, err := upload.GetReader()

	// Then
	assert.Nil(t, reader)
	assert.IsType(t, new(os.PathError), err)
}
