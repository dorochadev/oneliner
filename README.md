# oneliner cli v1

a cli tool that generates shell one-liners from natural language using llms.

## features

- 🤖 generate shell commands from natural language
- 🔌 support for openai and claude apis
- 🎨 clean, styled terminal output
- ⚡ optional command execution with sudo
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
  "model": "gpt-4o-mini",
  "default_shell": "bash",
  "safe_execution": true
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

* `llm_api`: api provider (`openai` or `claude`)
* `api_key`: your api key
* `model`: model to use (e.g., `gpt-4o-mini`, `claude-sonnet-4-20250514`)
* `default_shell`: preferred shell (`bash`, `zsh`, `fish`)
* `safe_execution`: enable safety checks (recommended: `true`)

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

### execute the command

use `--execute` or `-e` to run the generated command with sudo:

```bash
oneliner --execute "update all system packages"
```

you'll be prompted to confirm before execution.

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
* command caching for frequently used queries
* additional shell support (fish, powershell)
* local llm integration
* command history and favorites?

## security

* all commands are executed with `sudo` for safety
* interactive confirmation required before execution
* `safe_execution` mode helps prevent destructive commands
* api keys stored in config file with restrictive permissions (0600)