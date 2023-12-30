# clAI

A co-pilot for your terminal


- Bring your own openAI API key
- Smart context aware results


<blockquote class="twitter-tweet" data-media-max-width="560"><p lang="en" dir="ltr">Weekend project:<br>I was having some issues with GitHub cli copilot so I built my own. Some features:<br>- bring own OpenAI api key<br>- model and parameters selection<br>- better environment context <br>- markdown rendering for code explanation<br>- Code to clipboard<br>- Inline editor for the code <a href="https://t.co/GTuuiZkEZA">pic.twitter.com/GTuuiZkEZA</a></p>&mdash; Vik 💿 (@kivSegrob) <a href="https://twitter.com/kivSegrob/status/1682887514002948098?ref_src=twsrc%5Etfw">July 22, 2023</a></blockquote> <script async src="https://platform.twitter.com/widgets.js" charset="utf-8"></script>


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