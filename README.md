# oneliner

> 🧠 Generate shell one-liners from natural language — safely, instantly, and beautifully.

Turn natural language into shell commands using LLMs (OpenAI, Claude, or local models).  
Stop searching Stack Overflow, just tell your terminal what you want.

![Demo](./demo-assets-ignore/demo.gif)

---

## 🚀 Quick Install

Requires Go **1.25.1+**

```bash
go install github.com/dorochadev/oneliner@latest
```

Or build from source:

```bash
git clone https://github.com/dorochadev/oneliner.git
cd oneliner
go build -o oneliner .
```

**After installation, run the setup wizard:**

```bash
oneliner setup
```

---

## ⚡ Quick Start

### First Time Setup

Run the interactive setup wizard to configure your LLM provider:

```bash
oneliner setup
```

This will guide you through:
- 🎯 Choosing your LLM provider (OpenAI, Claude, or Local)
- 🔑 Entering your API key
- 🤖 Selecting a model
- ⚙️ Additional configuration options

### Generate Commands

```bash
oneliner "find all jpg files larger than 10MB"
```

👉 The command is **shown, not executed** by default — review before running.

### Manual Configuration (Optional)

If you prefer to configure manually, the config file is located at:

```
~/.config/oneliner/config.json
```

Set your API key manually:

```bash
oneliner config set llm_api openai
oneliner config set api_key sk-xxxx
oneliner config set model gpt-4o
```

---

## ✨ Features

* 🤖 Supports **OpenAI, Claude, or local LLMs**
* 🎯 **Interactive setup wizard** for easy configuration
* 🧠 **Context-aware** — OS, shell, and directory detection
* 🎨 **Pretty terminal UI** with Lipgloss & Bubble Tea
* ⚡ **Fast** — cached results for repeated queries
* 📋 **Clipboard copy**, explanations, and multi-shell output

---

## 💡 Examples

```bash
oneliner "find all files larger than 1GB"
oneliner "show top 10 processes by CPU usage"
oneliner "convert all png files to jpg"
oneliner --explain "recursively delete node_modules folders"
oneliner --clipboard "compress all pdfs in current directory"
```

---

## ⚠️ Safety (Read)

`oneliner` never runs commands unless you explicitly tell it to with `--run`.
Even then, it performs a regex-based safety check to warn about dangerous commands.

> Use `--run` and especially `--sudo` only when you're 100% sure what the command does.

---

## 🧰 Advanced Usage

| Flag          | Short | Description                     |
| ------------- | ----- | ------------------------------- |
| `--run`       | `-r`  | Execute the command immediately |
| `--sudo`      |       | Prepend `sudo` (Unix only)      |
| `--explain`   | `-e`  | Show what the command does      |
| `--clipboard` | `-c`  | Copy to clipboard               |
| `--config`    |       | Custom config path              |

---

## ⚙️ Configuration

### Interactive Setup

Run the setup wizard anytime to reconfigure:

```bash
oneliner setup
```

### View Configuration

```bash
oneliner config list
```

### Update Configuration

```bash
oneliner config set llm_api claude
oneliner config set model gpt-4o
oneliner config set api_key sk-xxxx
```

---

## 🧩 Local LLM Support

You can connect to your own model by selecting "local" in the setup wizard, or manually:

```json
{
  "llm_api": "local",
  "local_llm_endpoint": "http://localhost:8000/v1/completions",
  "model": "llama3"
}
```

Or via config command:

```bash
oneliner config set llm_api local
oneliner config set local_llm_endpoint "http://localhost:8000/v1/completions"
oneliner config set model llama3
```

---

## 🧹 Cache Management

View cached commands:

```bash
oneliner cache list
```

Clear cache:

```bash
oneliner cache clear
```

Remove specific cached entry:

```bash
oneliner cache rm <id>
```

---

## 🛠️ Troubleshooting

| Issue                         | Solution                           |
| ----------------------------- | ---------------------------------- |
| `oneliner: command not found` | Add `$(go env GOPATH)/bin` to PATH |
| Configuration incomplete      | Run `oneliner setup`               |
| API errors                    | Check API key and connectivity     |
| Cache issues                  | Run `oneliner cache clear`         |

---

## 🧑‍💻 Contributing

Pull requests are welcome! Feel free to open issues for bugs or feature requests.

---

## 📜 License

MIT

---

> Built with ❤️ using [Cobra](https://github.com/spf13/cobra), [Bubble Tea](https://github.com/charmbracelet/bubbletea), and [Lipgloss](https://github.com/charmbracelet/lipgloss).