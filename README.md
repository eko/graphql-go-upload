graphql-go-upload
=================

[![TravisBuildStatus](https://api.travis-ci.org/eko/graphql-go-upload.svg?branch=master)](https://travis-ci.org/eko/graphql-go-upload)
[![GoDoc](https://godoc.org/github.com/eko/graphql-go-upload?status.png)](https://godoc.org/github.com/eko/graphql-go-upload)
[![GoReportCard](https://goreportcard.com/badge/github.com/eko/graphql-go-upload)](https://goreportcard.com/badge/github.com/eko/graphql-go-upload)

This library exposes a middleware for the [GraphQL-Go](https://github.com/graph-gophers/graphql-go) project in order to expose a new `Upload` scalar type and allow you to send `multipart/form-data` POST requests containing files and fields data.

Installation
------------

```bash
$ dep ensure --add github.com/eko/graphql-go-upload
```

Add the middleware handler in your GraphQL project
--------------------------------------------------

Once the dependency is installed, simply update your GraphQL project code in order to add this middleware:

```go
import (
    "github.com/eko/graphql-go-upload"
)

// ...

h := handler.GraphQL{
    Schema: graphql.MustParseSchema(schema.String(), root, graphql.MaxParallelism(maxParallelism), graphql.MaxDepth(maxDepth)),
    Handler: handler.NewHandler(conf, &m),
}

mux := mux.NewRouter()
mux.Handle("/graphql", upload.Handler(h)) // Add the middleware here (wrap the original handler)

s := &http.Server{
    Addr:    ":8000",
    Handler: mux,
}
```

You're ready to use the new middleware!

Use the new Upload scalar type
------------------------------

In order to use the new Upload scalar type, you have to declare it in your GraphQL schema and use it in your mutations, this way:

```graphql
scalar Upload

type Mutation {
    myUploadMutation(file: Upload!, title: String!): Boolean
}
```

Usage on client side
--------------------

On a client point of view, requests have to be formed this way:

```
$ curl http://localhost:8000/graphql \
  -F operations='{ "query": "mutation DoUpload($file: Upload!, $title: String!) { upload(file: $file, title: $title) }", "variables": { "file": null, "title": null } }' \
  -F map='{ "file": ["variables.file"], "title": ["variables.title"] }' \
  -F file=@myfile.txt \
  -F title="My content title"
```
