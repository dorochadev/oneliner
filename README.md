# oneliner 🧠

> Turn plain English into shell commands using OpenAI, Claude, or local LLMs,  **designed to teach, not replace your knowledge**.

We’ve all been there: you know what command you want to run, but the syntax, `awk`, `find`, or `sed`, slips your mind. `oneliner` helps you **figure it out in your terminal**, so you can learn as you go, without leaving the shell or installing heavyweight tools like Warp or Claude CLI.


![Demo](./demo-assets-ignore/demo.gif)

---

## 🚀 Install

Requires Go **1.25.1+**

```bash
go install github.com/dorochadev/oneliner@latest
````

Or build from source:

```bash
git clone https://github.com/dorochadev/oneliner.git
cd oneliner
go build -o oneliner .
```

Run setup:

```bash
oneliner setup
```

---

## ⚡ Quick Start

Generate commands:

```bash
oneliner "find all jpg files larger than 10MB"
oneliner --explain "delete node_modules recursively"
oneliner --clipboard "compress all pdfs"
```

> Commands are **shown, not executed** by default. Use `--run` only when you’re sure.

For configuration details, see the **Configuration** section below.

---

## ✨ Features

* Supports OpenAI, Claude, and local LLMs
* Context-aware (OS, shell, directory)
* Pretty terminal UI (Lipgloss & Bubble Tea)
* Fast, cached results
* Clipboard copy & explanations

---

## ⚠️ Safety

`oneliner` never runs commands unless you explicitly use `--run`.
A regex-based safety check warns about dangerous commands, but **do not rely on it blindly**.

> Use `--run` and `--sudo` only when 100% sure what the command does.

---

## 🧰 Usage Flags

| Flag            | Short | Description                     |
| -------------   | ----- | ------------------------------- |
| `--run`         | `-r`  | Execute the command immediately |
| `--sudo`        |       | Prepend `sudo` (Unix only)      |
| `--explain`     | `-e`  | Show what the command does      |
| `--clipboard`   | `-c`  | Copy to clipboard               |
| `--interactive` | `-i`  | Run in interactive mode         |
| `--config`      |       | Custom config path              |

---

## ⚙️ Configuration

Manage your LLM setup in one place:

* **Interactive Setup:**

```bash
oneliner setup
```

* **View Current Config:**

```bash
oneliner config list
```

* **Set Config Manually:**

```bash
oneliner config set llm_api openai
oneliner config set api_key sk-xxxx
oneliner config set model gpt-4o
```

* **Local LLM Example:**

```bash
oneliner config set llm_api local
oneliner config set local_llm_endpoint "http://localhost:8000/v1/completions"
oneliner config set model llama3
```

* **Config File:** `~/.config/oneliner/config.json`

---

## 🧩 Cache Management

```bash
oneliner cache list
oneliner cache clear
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

Pull requests are welcome! Open issues for bugs or feature requests.

---

## 📜 License

MIT

---

> Built with ❤️ using [Cobra](https://github.com/spf13/cobra), [Bubble Tea](https://github.com/charmbracelet/bubbletea), and [Lipgloss](https://github.com/charmbracelet/lipgloss)
