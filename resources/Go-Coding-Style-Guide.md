# Go Coding Style Guide

> [!NOTE]
> This is a living document that will evolve with our team's practices and
> experiences. While the guidelines are presented prescriptively for clarity,
> the intention is to be descriptive of the existing practices and conventions
> our team has found effective. We welcome suggestions and improvements through
> the standard review process.

## Purpose and Usage

This document serves as an easy reference guide for Go developers during code
reviews, providing concrete examples and rationale for our coding conventions.
It is designed to complement, not replace, the advice from the Go Language
Design Team:

- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)
- [Go Test Comments](https://go.dev/wiki/TestComments)
- [Go Proverbs](https://go-proverbs.github.io/)

<details>

<summary>For example</summary>

...if you see this during a code review:

```go
type Connection interface {

	// Connection retrieves a Connection.
	Connection(ctx context.Context, id int64) (*db.Connection, error)

	// Connections retrieves all connections.
	Connections(ctx context.Context) ([]db.Connection, error)

	// CreateConnection creates a new connection.
	CreateConnection(ctx context.Context, request *ConnectionCreate, hostURL string) (*db.Connection, error)

	// UpdateConnection updates a connection according to the given update request.
	UpdateConnection(ctx context.Context, id int64, u *ConnectionUpdate) error

	// DeleteConnections deletes multiple Connections by ID.
	DeleteConnections(ctx context.Context, ids ...int64) error

	// AuthorizeConnection authorizes a connection.
	AuthorizeConnection(ctx context.Context, id int64) error

	// AuthorizeConnectionCallback authorizes a connection via a callback.
	AuthorizeConnectionCallback(ctx context.Context, id int64, state, code string) error

	// GetAuthCodeURL retrieves the URL for authorising an OAuth2 connection.
	GetAuthCodeURL(ctx context.Context, id int64) (string, error)

	// RevokeConnection revokes a connection's authorisation (e.g. OAuth2 tokens).
	RevokeConnection(ctx context.Context, id int64) error
}
```

You can simply say:

> [The bigger the interface, the weaker the
> abstraction.](https://www.youtube.com/watch?v=PAAkCSZUG1c&t=5m17s) Let's talk
> about how we can break this up. Is there a possibility that it could be
> removed?

Remember, we do code reviews so that we can build software *together*. Maybe
there's a good reason for the interface to have over half a dozen methods. If
that's the case then communicate, build consensus and write it down.

</details>

## Table of Contents

1. [Formatting](#formatting)
2. [Naming Conventions](#naming-conventions)
3. [Error Handling](#error-handling)
4. [Comments and Documentation](#comments-and-documentation)
5. [Package Design](#package-design)
6. [Interfaces](#interfaces)
7. [Concurrency](#concurrency)
8. [Testing](#testing)
9. [Performance](#performance)
10. [Dependencies](#dependencies)

## Formatting

### Use gofmt, goimports and staticcheck

All code must be formatted with `gofmt`, imports organised with `goimports`
(which automatically runs `gofmt`) and checked by [`govulncheck`],
[`staticcheck`] and [`errcheck`].

```bash
goimports -w .
govulncheck ./...
staticcheck ./...
errcheck ./...
```

[`errcheck`]: https://github.com/kisielk/errcheck
[`govulncheck`]: https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck
[`staticcheck`]: https://staticcheck.dev/

### Line Length

While Go doesn't enforce a strict line length, aim for 100-120 characters for
readability.

> [!CAUTION]
> ```go
> func processUserDataAndSendNotificationEmailWithAttachmentsAndLogging(user *User, data *ProcessingData, attachments []Attachment, logger *Logger, emailService EmailService) error {
> ```

> [!TIP]
> ```go
> func processUserData(
> 	user *User,
> 	data *ProcessingData,
> 	attachments []Attachment,
> 	logger *Logger,
> 	emailService EmailService,
> ) error {
> ```

## Naming Conventions

### Package Names

Package names should be short, concise, lowercase, and singular. Avoid
underscores or mixedCaps.

> [!CAUTION]
> ```go
> package user_manager
> package userManager
> package users
> ```

> [!TIP]
> ```go
> package user
> package userauth
> ```

### Interface Names

Single-method interfaces should be named with the method name plus an "-er"
suffix.

> [!TIP]
> ```go
> type Reader interface {
> 	Read([]byte) (int, error)
> }
>
> type Upserter interface {
> 	Upsert(context.Context, []db.StepDefinition) error
> }
> ```

For multi-method interfaces, use descriptive names that explain the interface's
purpose.

> [!TIP]
> ```go
> type FileSystem interface {
> 	Open(name string) (File, error)
> 	Stat(name string) (FileInfo, error)
> }
> ```

### Receiver Names

Receiver names should be short (one or two characters), representative of the
type name and consistent across methods.

> [!CAUTION]
> ```go
> func (customer *Customer) Name() string { } // too long
> func (c *Customer) Age() int { }            // differs from other methods
> func (self *Customer) Address() string { }  // betrays your Java background!
> ```

> [!TIP]
> ```go
> func (c *Customer) Name() string { }
> func (c *Customer) Age() int { }
> func (c *Customer) Address() string { }
> ```

> [!NOTE]
> Consistency is more important for readability than choosing the best name.
> This is especially important for short methods. When the method names are
> aligned it allows the eye to quickly scan over them.

### Acronyms

Acronyms should be all capitals or all lowercase for consistency.

> [!CAUTION]
> ```go
> var xmlHttpRequest XMLHttpRequest
> var userId string
> var resourceUri string
> ```

> [!TIP]
> ```go
> var xmlHTTPRequest XMLHTTPRequest
> var userID string
> var resourceURI string
> ```

## Error Handling

### Error Strings

Error strings should not be capitalized or end with punctuation.

> [!CAUTION]
> ```go
> return fmt.Errorf("Failed to connect to database.")
> ```

> [!TIP]
> ```go
> return fmt.Errorf("failed to connect to database")
> ```

This allows errors to be easily chained, for example:

> [!CAUTION]
> ```text
> Error: Failed to fetch resource.: Failed to connect to database.
> ```

### Error Wrapping

Use `%w` verb with `fmt.Errorf` to wrap errors for proper error chain handling.

> [!TIP]
> ```go
> if err != nil {
> 	return conf, fmt.Errorf("failed to get default gRPC service config: %w", err)
> }
> ```

### Word choice

For client errors, the error string should be instructive or identify persicely
the part of the provided information that needs to be changed to successfully
make the call.

> [!CAUTION]
> ```go
> fmt.Errorf("invalid format") // Format of what? Why, how should the caller fix it?
> ```

> [!TIP]
> ```go
> fmt.Errorf("invalid date format, single quote not closed")
> fmt.Errorf("key %q was not provided", key)
> ```

For wrapped errors, the error string should describe the action that was
attempted. Use the phrase "failed to X" over "unable to X".

> [!CAUTION]
> ```go
> fmt.Errorf("failed to get cron details: %w", err) // What was the ID/name of the cron?
> fmt.Errorf("unable to read content: %w", err)     // Prefer "failed to read ..."
> ```

> [!TIP]
> ```go
> fmt.Errorf("failed to retrieve %q from URL path: %w", key, err)
> ```

When looping over a range of values, add the current value in the loop to the
error string. Do not add values that have been passed in from the caller, this
is the responsibility of the caller to correctly wrap the error string.

> [!TIP]
> ```go
> 	for _, container := range resp.ContainerItems {
> 		_, err := client.DeleteContainer(ctx, *container.Name, nil)
> 		if err != nil {
> 			return fmt.Errorf("container %s: failed to delete: %w", *container.Name, err)
> 		}
> 	}
> ```

> [!CAUTION]
> ```go
> func (c *Client) DeleteContainer(name *string) error {
> 	...
> 	return fmt.Errorf("container %s is running", name)
> }
> ```

> [!IMPORTANT]
> **DO NOT** put Personal Identifying Information (PII) or other sensitive
> information into an error string.

### Error Checking

Check errors immediately after the function call that may return them.

> [!CAUTION]
> ```go
> result, err := someFunction()
> doSomethingElse()
> if err != nil {
> 	return err
> }
> ```

> [!TIP]
> ```go
> result, err := someFunction()
> if err != nil {
> 	return fmt.Errorf("someFunction failed: %w", err)
> }
> doSomethingElse()
> ```

### Named Return Values for Errors

Avoid naked returns except in very short functions. Be explicit about what
you're returning.

> [!CAUTION]
> ```go
> func getUser(id string) (user *User, err error) {
> 	user, err = db.Query(id)
> 	return // naked return
> }
> ```

> [!TIP]
> ```go
> func getUser(id string) (*User, error) {
> 	user, err := db.Query(id)
> 	if err != nil {
> 		return nil, fmt.Errorf("querying user %s: %w", id, err)
> 	}
> 	return user, nil
> }
> ```

## Comments and Documentation

Comments should describe **intentions**, not explain logic. Ideally, the code
should be readable so that the logic is clear, that is what the machine
executes. The machine doesn't need to know your intentions and motivations.
Users of your package need to know this and you communicate it via code
comments and published documentation (i.e. `go doc`).

### Package Comments

Every package should have a package comment. For multi-file packages, the
comment only needs to be in one file. If in doubt, create a new file called
`doc.go` in the package and write the package comment there without any
symbols.

The package comment should start with "// Package <pkgname>".

> [!CAUTION]
> ```go
> // This package provides types and functions for user management.
> package user
> ```

> [!TIP]
> ```go
> // Package user provides types and functions for user management,
> // including authentication, authorization, and profile management.
> //
> // ... detailed overview ...
> //
> // ... high-level package organisation ...
> //
> // ... common package entrypoint(s) and usage ...
> package user
> ```

### Function Comments

Exported functions must have comments starting with the function name.

> [!CAUTION]
> ```go
> // For the given price and discount percentage, this function returns the
> // discount amount. It returns an error if the percentage is negative or
> // greater than 100.
> func CalculateDiscount(price, percentage float64) (float64, error) {
> 	// ...
> }
> ```

> [!TIP]
> ```go
> // CalculateDiscount returns the discount amount for the given price
> // and discount percentage. It returns an error if the percentage is
> // negative or greater than 100.
> func CalculateDiscount(price, percentage float64) (float64, error) {
> 	// ...
> }
> ```

### TODO Comments

TODO comments should include an issue number. This means you should create an
issue if one doesn't exist. Alternatively, address the issue and remove the
TODO.

```go
// TODO(machship/v3-issues#1438): remove this endpoint.
```

## Package Design

### Package Structure

Keep packages focused on a single responsibility. Avoid circular dependencies.
If your application contains a single executable, keep it simple and place the
`main.go` file at the root of the repository, otherwise use `cmd` as a path
name. If you application requires complex tests that go beyond a simple `go
test` call, then consider using a `tests` path to add tests and and a `cmd/test`
path for tools to run these tests. Otherwise, all other packages should by
default be placed under the `internal` path.

For more complex projects consider [Hexagonal Architecture].

```
project/
├── cmd/                     // Executables for the app's tools and services
│   ├── api/main.go
│   └── test/                // Tools to help test difficult and complex scenarios
│       └── k6/main.go
├── tests/                   // Manual tests the require significant setup to run
└── internal/                // All other packages live here, by default
    ├── user/
    │   ├── service.go
    │   ├── service_test.go
    │   └── repository.go
    ├── auth/
    │       ├── jwt.go
    │       └── jwt_test.go
    └── logger/
        └── logger.go
```

[Hexagonal Architecture]: https://medium.com/@mike_polo/structuring-a-golang-pr

### Internal Packages

Use `internal/` for packages that shouldn't be imported by external projects.

```go
import (
	"myproject/internal/auth"  // Only importable within myproject
)
```

### Main Package

The `main` package is special; it defines an executable and is an entrypoint
into your application. For testability, your `main` function should be as small
as possible and handle **global resources exclusively**. These global resources
should then be passed off to an internal app package.

> [!TIP]
> ```go
> import (
> 	"log"
> 	"os"
> 	"signal"
>
> 	"myproject/internal/app"
> )
>
> func main() {
> 	name, args := os.Args[0], os.Args[1:]
>
> 	log.SetPrefix(name + ": ")
>
> 	rootFS, err := os.OpenRoot("/")
> 	if err != nil {
> 		log.Fatalln(err)
> 	}
>
> 	app := app.New(name, args, rootFS)
>
> 	ctx, stop = signal.NotifyContext(context.Background(), os.Interrupt)
> 	defer stop()
>
> 	status := app.Run(ctx)
>
> 	os.Exit(status)
> }
> ```

## Interfaces

### Accept Interfaces, Return Structs

Functions should accept interfaces and return concrete types when possible. The
`error` interface is the one exception to this rule (builtins are special).

> [!CAUTION]
> ```go
> func NewWebReport(url string) Report {
> 	return &WebReport{url: url}
> }
> ```

> [!TIP]
> ```go
> // Return concrete type
> func NewWebReport(url string) *WebReport {
> 	return &WebReport{url: url}
> }
>
> // Accept io.Writer interface
> func (r *WebReport) WriteReport(w io.Writer) error {
> 	// ...
> }
> ```

### Small Interfaces

Prefer many small interfaces to large ones.

> [!CAUTION]
> ```go
> type Storage interface {
> 	Get(key string) ([]byte, error)
> 	Set(key string, value []byte) error
> 	Delete(key string) error
> 	List() ([]string, error)
> 	Stats() (*StorageStats, error)
> 	Backup() error
> 	Restore(backup []byte) error
> }
> ```

> [!TIP]
> ```go
> type Getter interface {
> 	Get(key string) ([]byte, error)
> }
>
> type Setter interface {
> 	Set(key string, value []byte) error
> }
>
> type GetSetter interface {
> 	Getter
> 	Setter
> }
> ```

### The Consumer Owns The Interface

Don't define the interface if it's not needed. Packages that implement an
interface are typically separate from the package that consume the structs from
the implementations. This means that there's no need for the implementor to
import or define the interface, this is the responsibility of each package that
uses the interface.

## Concurrency

### Channel Ownership

The goroutine that writes to a channel is responsible for closing it. If there
are multiple goroutines writing to a channel then the creator of the channel
should manage and coordinate the goroutines and then close the channel.

> [!TIP]
> ```go
> func producer() <-chan int {
> 	ch := make(chan int)
> 	go func() {
> 		defer close(ch)
> 		for i := 0; i < 10; i++ {
> 			ch <- i
> 		}
> 	}()
> 	return ch
> }
> ```

### Context Usage

Pass context as the first parameter to functions that support cancellation.

> [!TIP]
> ```go
> func FetchUser(ctx context.Context, id string) (*User, error) {
> 	// Proceed with operation
> 	return db.QueryUser(ctx, id)
> }
> ```

### Pessimistic Callers

Don't assume the functions that you pass context to will respect it's
cancellation. Always listen to the Done channel yourself.

> [!CAUTION]
> ```go
> func fetchData(ctx context.Context) (string, error) {
> 	result := make(chan string, 1)
> 	errc := make(chan error, 1)
>
> 	go func() {
> 		data, err := slowOperation(ctx) // Might not respect ctx
> 		if err != nil {
> 			errc <- err
> 		} else {
> 			result <- data
> 		}
> 	}()
>
> 	select {
> 	case data := <-result:
> 		return data, nil
> 	case err := <-errc:
> 		return "", err
> 		// BUG: Will block forever if ctx cancelled but slowOperation ignores it
> 	}
> }
> ```

> [!TIP]
> ```go
> func fetchData(ctx context.Context) (string, error) {
> 	result := make(chan string, 1)
> 	errc := make(chan error, 1)
> 	defer close(result)
> 	defer close(eerc)
>
> 	go func() {
> 		data, err := slowOperation(ctx)
> 		if err != nil {
> 			errc <- err
> 		} else {
> 			result <- data
> 		}
> 	}()
>
> 	select {
> 	case data := <-result:
> 		return data, nil
> 	case err := <-errc:
> 		return "", err
> 	case <-ctx.Done():
> 		return "", ctx.Err() // Always provide escape hatch
> 	}
> }
> ```

### Mutex Placement

Keep mutexes close to the data they protect.

> [!TIP]
> ```go
> type Counter struct {
> 	mu    sync.Mutex
> 	value int
> }
>
> func (c *Counter) Increment() {
> 	c.mu.Lock()
> 	defer c.mu.Unlock()
> 	c.value++
> }
> ```

## Testing

### Use `gotestsum`

Use `gotestsum` as a convenience wrapper around `go test` that is both CI and
human friendly.

> [!TIP]
> ```sh
> go get -tool gotest.tools/gotestsum@latest
> go tool gotestsum
> ```

### Table-Driven Tests

Prefer table-driven tests for testing multiple scenarios. [Table-Driven Tests vs Multiple Test Functions](https://go.dev/wiki/TestComments#table-driven-tests-vs-multiple-test-functions) notwithstanding. Also, use a `map[string]struct` over `[]struct`.

> [!TIP]
> ```go
> func TestCalculateDiscount(t *testing.T) {
> 	tests := map[string]struct {
> 		price      float64
> 		percentage float64
> 		want       float64
> 		wantErr    bool
> 	}{
> 		"valid discount": {
> 			price:      100,
> 			percentage: 10,
> 			want:       10,
> 			wantErr:   false,
> 		},
> 		"negative percentage": {
> 			price:      100,
> 			percentage: -10,
> 			want:       0,
> 			wantErr:   true,
> 		},
> 	}
>
> 	for name, tt := range tests {
> 		t.Run(name, func(t *testing.T) {
> 			got, err := CalculateDiscount(tt.price, tt.percentage)
> 			if (err != nil) != tt.wantErr {
> 				t.Errorf("CalculateDiscount() error = %v, wantErr %v", err, tt.wantErr)
> 				return
> 			}
> 			if got != tt.want {
> 				t.Errorf("CalculateDiscount() = %v, want %v", got, tt.want)
> 			}
> 		})
> 	}
> }
> ```

### Test Names

Test names should clearly describe what is being tested.

> [!TIP]
> ```go
> func TestUserService_CreateUser_DuplicateEmail(t *testing.T) {
> 	// Test that creating a user with duplicate email returns error
> }
>
> func TestUserService_CreateUser_Success(t *testing.T) {
> 	// Test successful user creation
> }
> ```

### Parallel Tests

Use `t.Parallel()` for tests that can run concurrently.

> [!TIP]
> ```go
> func TestSomething(t *testing.T) {
> 	t.Parallel()
> 	// test implementation
> }
> ```

> [!CAUTION]
> Avoid `t.Parallel()` for tests that bind to network ports or have other resource
> conflicts. These tests should run sequentially to prevent race conditions.
>
> ```go
> func TestServer_StartStop(t *testing.T) {
> 	// Note: Not parallel due to port allocation
> 	listener, _ := net.Listen("tcp", ":0")
> 	port := listener.Addr().(*net.TCPAddr).Port
> 	listener.Close()
> 	// ... use port
> }
> ```

### Use t.Context()

Always use `t.Context()` instead of `context.Background()` in tests. This ensures
proper timeout handling and automatic cancellation when tests complete or timeout.

> [!CAUTION]
> ```go
> func TestSomething(t *testing.T) {
> 	ctx := context.Background()  // Won't respect test timeouts
> 	// ...
> }
> ```

> [!TIP]
> ```go
> func TestSomething(t *testing.T) {
> 	ctx := t.Context()  // Automatically cancelled when test completes
> 	// ...
> }
>
> func TestWithTimeout(t *testing.T) {
> 	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
> 	defer cancel()
> 	// ...
> }
> ```

### HTTP Server Testing

When testing HTTP servers, use `httptest.NewUnstartedServer()` with your server
configuration to test the complete server setup including BaseContext, timeouts,
and other settings.

> [!CAUTION]
> ```go
> func TestServer(t *testing.T) {
> 	// Only tests the handler, not server configuration
> 	testSrv := httptest.NewServer(handler)
> 	defer testSrv.Close()
> 	// ...
> }
> ```

> [!TIP]
> ```go
> func TestServer(t *testing.T) {
> 	app := New(config)
> 	ctx := t.Context()
> 	srv := app.prepareServer(ctx)
>
> 	// Test the complete server configuration
> 	testSrv := httptest.NewUnstartedServer(nil)
> 	testSrv.Config = srv
> 	testSrv.Start()
> 	defer testSrv.Close()
> 	// ...
> }
> ```

> [!NOTE]
> Use `httptest.NewServer(handler)` directly when testing simple handlers that
> don't require server-level configuration:
>
> ```go
> func TestHandler(t *testing.T) {
> 	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
> 		w.WriteHeader(http.StatusOK)
> 	})
> 	testSrv := httptest.NewServer(handler)
> 	defer testSrv.Close()
> 	// ...
> }
> ```

## Performance

### Avoid Premature Optimization

Write clear, correct code first. Optimize only after profiling identifies
bottlenecks. Measure, refactor, measure.

### Preallocate Slices

When the size is known, preallocate slices to avoid multiple allocations.

> [!CAUTION]
> ```go
> var users []*User
> for _, id := range userIDs {
> 	users = append(users, fetchUser(id))
> }
> ```

> [!TIP]
> ```go
> users := make([]*User, 0, len(userIDs))
> for _, id := range userIDs {
> 	users = append(users, fetchUser(id))
> }
> ```

### String Building

For loops with many iterations (>100) or when building large strings, use
`strings.Builder`. For simple cases with a few concatenations, `+` is fine and
more readable.

**Use `+` for simple cases:**

```go
fullName := firstName + " " + lastName
```

**Use Builder for loops:**

```go
var builder strings.Builder
for _, line := range manyLines {  // When manyLines could be large
	builder.WriteString(line)
	builder.WriteString("\n")
}
result := builder.String()
```

## Dependencies

### Minimal Dependencies

Avoid unnecessary external dependencies. Prefer standard library when possible.

### Dependency Injection

Use dependency injection for better testability.

> [!CAUTION]
> ```go
> func ProcessOrder(orderID string) error {
> 	db := database.GetConnection()  // Hard-coded dependency
> 	// ...
> }
> ```

> [!TIP]
> ```go
> type OrderProcessor struct {
> 	db Database
> }
>
> func New(db Database) *OrderProcessor {
> 	// ...
> }
>
> func (p *OrderProcessor) ProcessOrder(orderID string) error {
> 	// Use injected dependency
> 	order, err := p.db.GetOrder(orderID)
> 	// ...
> }
> ```

### Module Versioning

Use semantic versioning for modules and specify exact versions in go.mod.

> [!TIP]
> ```go
> module github.com/machship/project
>
> go 1.24
>
> require (
> 	github.com/jackc/pgx/v5 v5.7.1
> 	github.com/shopspring/decimal v1.3.1
> )
> ```

### Tool use

Specify tools in the same way as modules by using `go get -tool pkgname`.

> [!TIP]
> ```go
> module github.com/machship/project
>
> go 1.24
>
> tool (
> 	github.com/kisielk/errcheck
> 	golang.org/x/vuln/cmd/govulncheck
> 	gotest.tools/gotestsum
> 	honnef.co/go/tools/cmd/staticcheck
> )
> ```

Invoke the tool using `go tool`, e.g. `go tool gotestsum`.

## Additional Resources

- [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)
- [Google Go Style Guide](https://google.github.io/styleguide/go/)
- [The Go Programming Language Specification](https://go.dev/ref/spec)

---

*To propose changes to this guide, please submit a pull request with your suggested modifications and rationale.*
