package upload

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"os"
)

// GraphQLUpload is the struct used for the new "Upload" GraphQL scalar type
//
// It allows you to use the Upload type in your GraphQL schema, this way:
//
//  scalar Upload
//
//  type Mutation {
//    upload(file: Upload!, title: String!, description: String!): Boolean
//  }
type GraphQLUpload struct {
	Filename string `json:"filename"`
	MIMEType string `json:"mimetype"`
	Filepath string `json:"filepath"`
}

// ImplementsGraphQLType is implemented to respect the GraphQL-Go Unmarshaler interface.
// It allows to chose the name of the GraphQL scalar type you want to implement
//
// Reference: https://github.com/graph-gophers/graphql-go/blob/bb9738501bd42a6536227b96068349b814379d6e/internal/exec/packer/packer.go#L319
func (u GraphQLUpload) ImplementsGraphQLType(name string) bool {
	return name == "Upload"
}

// UnmarshalGraphQL is implemented to respect the GraphQL-Go Unmarshaler interface.
// It hydrates the GraphQLUpload struct with input data
//
// Reference: https://github.com/graph-gophers/graphql-go/blob/bb9738501bd42a6536227b96068349b814379d6e/internal/exec/packer/packer.go#L319
func (u *GraphQLUpload) UnmarshalGraphQL(input interface{}) error {
	switch input := input.(type) {
	case map[string]interface{}:
		data, err := json.Marshal(input)
		if err != nil {
			u = &GraphQLUpload{}
		} else {
			json.Unmarshal(data, u)
		}

		return nil
	default:
		return errors.New("Cannot unmarshal received type as a GraphQLUpload type")
	}
}

// GetReader returns the buffer of the uploaded (and temporary saved) file.
func (u *GraphQLUpload) GetReader() (io.Reader, error) {
	f, err := os.Open(u.Filepath)
	if err == nil {
		return bufio.NewReader(f), nil
	}

	return nil, err
}
