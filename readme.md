A program that talks to the OpenAI (chat) API.

You'll need an API key, which can be generated [here](https://platform.openai.com/account/api-keys).

Make sure you have [Go](https://go.dev) installed, and run the program with

```
go run .  # or go run main.go
```

Run tests with:

```
go test ./... -count=1 -cover -coverprofile=cov.txt -race -v
```

Assuming you have the [cover](https://pkg.go.dev/cmd/cover)  tool installed, view the coverage in your browser with:

```
go tool cover -html=cov.txt
```

