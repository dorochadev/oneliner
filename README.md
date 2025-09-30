# oneliner cli v1

a cli tool that generates shell one-liners from natural language using llms.

## features

- 🤖 generate shell commands from natural language
- 🔌 support for openai and claude apis
- 🎨 clean, styled terminal output
- ⚡ optional command execution (--execute, --sudo)
- 📝 explain mode for command breakdowns
- 🔧 context-aware (os, shell, directory, user)

## installation

### prerequisites

- go 1.25.1 or higher
- an api key from openai or anthropic

### build from source

```bash
# clone the repository
git clone https://github.com/dorochadev/oneliner.git
cd oneliner

# initialize go modules and download dependencies
go mod download

# build the binary
go build -o oneliner .

# (optional) install to your PATH
sudo mv oneliner /usr/local/bin/
````

## configuration

on first run, a default config file will be created at `~/.config/oneliner/config.json`:

```json
{
  "llm_api": "openai",
  "api_key": "",
  "model": "gpt-4.1-nano",
  "default_shell": "bash",
  "safe_execution": true,
  "local_llm_endpoint": "http://localhost:8000/v1/completions"
}
```

### setting up your api key

edit the config file and add your api key:

```bash
# for openai
nano ~/.config/oneliner/config.json
# set "api_key": "sk-..."

# for claude
# set "llm_api": "claude"
# set "api_key": "sk-ant-..."
# set "model": "claude-sonnet-4-20250514"
```


### configuration options

* `llm_api`: api provider (`openai`, `claude`, or `local`)
* `api_key`: your api key (for openai/claude)
* `model`: model to use (e.g., `gpt-4.1-nano`, `claude-sonnet-4-20250514`)
* `default_shell`: preferred shell (`bash`, `zsh`, `fish`, `powershell`)
* `safe_execution`: enable safety checks (recommended: `true`)
* `local_llm_endpoint`: URL for your locally hosted LLM (used if `llm_api` is `local`)
#### example: use a local LLM

To use a locally hosted LLM API:

```json
{
  "llm_api": "local",
  "local_llm_endpoint": "http://localhost:8000/v1/completions"
}
```

#### example: use fish or powershell

To generate commands for fish shell:

```json
{
  "default_shell": "fish"
}
```

To generate commands for PowerShell:

```json
{
  "default_shell": "powershell"
}
```

## usage

### basic usage

generate a command:

```bash
oneliner "find all jpg files larger than 10MB"
```

output:

```
  find . -type f -name "*.jpg" -size +10M
```

you'll be prompted to confirm before execution.

### execute the command

Use `--execute` or `-e` to run the generated command as-is:

```bash
oneliner --execute "update all system packages"
```

To run the command with `sudo`, add the `--sudo` flag:

```bash
oneliner --execute --sudo "update all system packages"
```

You can use `--sudo` with or without `--execute` (but it only has effect if `--execute` is set). You'll be prompted to confirm before execution.

### get an explanation

use `--explain` to see what the command does:

```bash
oneliner --explain "compress all log files"
```

output:

```
  find . -name "*.log" -exec gzip {} \;

  explanation:
  - searches for all .log files and compresses them using gzip
```

### custom config file

```bash
oneliner --config /path/to/config.json "your query"
```

## examples

```bash
# file operations
oneliner "recursively delete empty directories"
oneliner "find files modified in the last 24 hours"

# system administration
oneliner "check disk usage of all mounted filesystems"
oneliner "list all running processes using more than 1GB of ram"

# text processing
oneliner "count unique ip addresses in access.log"
oneliner "replace all spaces with underscores in filenames"

# network
oneliner "show all listening tcp ports"
oneliner "test if port 8080 is open on localhost"
```

## future enhancements

* plugin system for common tools (ffmpeg, imagemagick, jq)
* command history and favorites?

## security

* By default, commands are only executed if you use `--execute`.
* Use `--sudo` to prepend `sudo` to the generated command (not automatic).
* Interactive confirmation required before execution.
* `safe_execution` mode helps prevent destructive commands.
* API keys stored in config file with restrictive permissions (0600).