# oneliner 🧠

> Turn plain English into shell commands using OpenAI, Claude, or local LLMs, **designed to teach, not replace your knowledge**.

We’ve all been there: you know what command you want to run, but the syntax, `awk`, `find`, or `sed` slips your mind. `oneliner` helps you **figure it out in your terminal**, so you can learn as you go, without leaving the shell or installing heavyweight tools like Warp or Claude CLI.

![Demo](./demo-assets-ignore/demo.gif)

---

## 🚀 Install

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
oneliner --breakdown "list all active network connections with details"
```

> Commands are **shown, not executed** by default. Use `--run` only when you’re sure.

For configuration details, see the **Configuration** section below.

---

## ✨ Features

* Supports OpenAI, Claude, and local LLMs
* Context-aware (OS, shell, directory)
* Pretty terminal UI (Lipgloss & Bubble Tea)
* Fast, cached results
* Clipboard copy, explanations, and detailed command breakdowns

---

## ⚠️ Safety

`oneliner` never runs commands unless you explicitly use `--run`.
A regex-based safety check warns about dangerous commands, but **do not rely on it blindly**.

> Use `--run` and `--sudo` only when 100% sure what the command does.

---

## 🧰 Usage Flags

| Flag            | Short | Description                                  |
| --------------- | ----- | -------------------------------------------- |
| `--run`         | `-r`  | Execute the command immediately              |
| `--sudo`        |       | Prepend `sudo` (Unix only)                   |
| `--explain`     | `-e`  | Show a brief explanation of the command      |
| `--clipboard`   | `-c`  | Copy command to clipboard                    |
| `--interactive` | `-i`  | Run in interactive mode                      |
| `--breakdown`   | `-b`  | Full educational breakdown of command stages |
| `--config`      |       | Use a custom configuration file              |

---

## 🧠 Learn with `--breakdown`

The `--breakdown` (`-b`) flag is designed for **learning and understanding**, not just quick answers. It provides a **detailed, step-by-step explanation** of how a shell command works.

When you use `--breakdown`, `oneliner` will:

* **Show the command first** (no execution unless `--run` is also used).
* **Explain every stage** in a numbered list:

  1. Each pipe, flag, or redirection is explained.
  2. Data transformations between stages are described.
  3. Shell expansions, substitutions, or environment effects are clarified.
* **Provide multiple points as needed** — there’s no limit, so even complex commands are fully unpacked.
* **Serve as an educational tool** so you can learn why and how commands work, not just copy them.

### Example Usage

```bash
oneliner --breakdown "find all jpg files larger than 10MB"
```

> Output will include a numbered breakdown explaining each stage of the `find` command, what each option does, and how the pipeline processes files.

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
oneliner config set blacklisted_binaries '["rm", "dd", "mkfs"]'
```

* **Local LLM Example:**

```bash
oneliner config set llm_api local
oneliner config set local_llm_endpoint "http://localhost:8000/v1/completions"
oneliner config set model llama3
```

* **Config File:** `~/.config/oneliner/config.json`

* **Blacklisted Binaries:**

`oneliner` automatically blocks generation or execution of unsafe commands.  
The `blacklisted_binaries` list defines binaries considered dangerous, such as:

```json
"blacklisted_binaries": ["rm", "dd", "mkfs", "fdisk", "parted", "shred", "curl", "wget", "nc", "ncat"]
```
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