# clAI

A co-pilot for your terminal


- Bring your own openAI API key
- Smart context aware results


## Build and install the binary locally

```bash
go build main.go && mv main  ~/.local/bin/clai
```

## How to debug

- Run the dlv server command on the terminal:
```bash
dlv debug --api-version 2  --headless --listen=:2345 .
```

- Connect to server in vscode debug

- Terminal will run the commandline app

- Add breakpoints and debug away!