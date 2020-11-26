# servedir

The simple file server command.

This code is literally to simplify my life by making it so that instead of typing a few words in the command line to open a file server, I only have to write one word (not including options like port and directory).

Why did I do this? Because I'm a lazy guy who would rather make my life easier by writing less in the terminal.

## Installation

```bash
go get github.com/coby-spotim/servedir
cd $(go env GOPATH)/src/github.com/coby-spotim/servedir
go install
```

## Usage

To serve the current directory on port `8080`:
```bash
servedir
```

To serve the current directory on a different port:
```bash
servedir --port 8000
```

Or, using the shorthand:
```bash
servedir -p 8000
```

To serve a different directory on port `8080`:
```bash
servedir --dir /some/random/dir
```

Or, using the shorthand:
```bash
servedir -d /some/random/dir
```

## Now With Brotli Compression!!

Well, some time passed, and then I realized, what if I want to see how my files are compressed? So... I added some brotli compression!

To serve files with compression, simply run the normal command. Compression and best compression is on by default.

### To adjust the settings

To turn off compression:
```bash
servedir --compression false
```

Or, using the shorthand:
```bash
servedir -c false
```

To change the level of compression (can be between 0 and 11):
```bash
servedir --level 1
```

Or, using the shorthand:
```bash
servedir -l 1
```
