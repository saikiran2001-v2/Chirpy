go-servers servers:

## Definitions

**Mux (ServeMux)**
- A *multiplexer* (often called a **router**) that matches incoming request URLs to registered handler functions. In Go's standard library, `http.ServeMux` implements this behavior. It stores a map of pattern strings to `http.Handler`s and selects the most specific pattern for each request.

**http.Server**
- Represents an HTTP server. It holds configuration such as the address to listen on (`Addr`) and the handler that processes each request (`Handler`). Calling `ListenAndServe` starts the server and blocks until it stops.

**http.FileServer**
- An `http.Handler` that serves static files from a directory on the filesystem. It reads files from the given directory and writes them to the response, handling content‑type detection, range requests, and directory listings.

**http.Dir**
- A wrapper that implements the `http.FileSystem` interface for a local directory. `http.Dir(".")` means “use the current working directory as the root of the file server”.

**Handler**
- Any type that implements the `ServeHTTP(ResponseWriter, *Request)` method. Functions like `http.FileServer` and custom functions can be registered as handlers.


mux.Handle("/app", http.StripPrefix("/app", http.FileServer(http.Dir(".")))) // Why do i need strip prefix

In this setup the handler is registered for the path “/app” but the files you want to serve live in the directory you pass to http.FileServer (the current directory in this case). When a request comes in for “/app/index.html”, the request URL that the file server sees is “/app/index.html”. If you didn’t strip the “/app” prefix, the file server would look for a file literally named “app/index.html” inside the root directory, which isn’t what you want. By using http.StripPrefix(“/app”, …) you remove the “/app” part of the URL before passing it to the file server, so the file server receives just “/index.html” and can correctly locate the file in the directory. In short, StripPrefix maps the URL path you expose (“/app/…”) to the actual filesystem layout you’re serving.

---

new http.ServeMux: 
type ServeMux ¶
type ServeMux struct {
	// contains filtered or unexported fields
}
Patterns
Precedence
Trailing-slash redirection
Request sanitizing
Compatibility
ServeMux is an HTTP request multiplexer. It matches the URL of each incoming request against a list of registered patterns and calls the handler for the pattern that most closely matches the URL.

Patterns ¶
Patterns can match the method, host and path of a request. Some examples:

"/index.html" matches the path "/index.html" for any host and method.
"GET /static/" matches a GET request whose path begins with "/static/".
"example.com/" matches any request to the host "example.com".
"example.com/{$}" matches requests with host "example.com" and path "/".
"/b/{bucket}/o/{objectname...}" matches paths whose first segment is "b" and whose third segment is "o". The name "bucket" denotes the second segment and "objectname" denotes the remainder of the path.

new http.ServeMux: 
type ServeMux ¶
type ServeMux struct {
	// contains filtered or unexported fields
}
Patterns
Precedence
Trailing-slash redirection
Request sanitizing
Compatibility
ServeMux is an HTTP request multiplexer. It matches the URL of each incoming request against a list of registered patterns and calls the handler for the pattern that most closely matches the URL.

Patterns ¶
Patterns can match the method, host and path of a request. Some examples:

"/index.html" matches the path "/index.html" for any host and method.
"GET /static/" matches a GET request whose path begins with "/static/".
"example.com/" matches any request to the host "example.com".
"example.com/{$}" matches requests with host "example.com" and path "/".
"/b/{bucket}/o/{objectname...}" matches paths whose first segment is "b" and whose third segment is "o". The name "bucket" denotes the second segment and "objectname" denotes the remainder of the path.
