List imports

# install
```bash
go get github.com/scorredoira/deps
```

# Usage

List all imported packages:

```bash
deps .
```

Execute a command on all imported packages:

```bash
deps -e "go generate" .
```

Filter imports by regex and pass environment variables:

```bash
debug=true deps -p "workspace/" -e "go generate" .
```
