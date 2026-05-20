# Architecture — tfscli

## Tech Stack

### Language: Go

**Rationale:**
- Compiles to a single static binary with zero runtime dependencies. Users download one file and it works.
- Cross-compilation built into the toolchain (`GOOS=windows GOARCH=amd64 go build`). Primary target is Windows, but Linux/macOS binaries are free.
- Fast cold start (~10ms). Important for a tool called repeatedly by an AI agent.
- `net/http` in standard library is sufficient for REST API calls — no need for third-party HTTP clients.
- `encoding/json` in standard library handles JSON parsing and serialization.

**Considered alternatives:**

| Alternative | Why rejected |
|---|---|
| C# | Initial candidate due to SSPI/`UseDefaultCredentials` for Windows auth. After choosing PAT over SSPI, the main advantage disappeared. Requires .NET runtime (unless using Native AOT, which increases build complexity). Larger binary size (~15-30 MB self-contained vs ~5-10 MB Go). |
| PowerShell | Good for prototyping. Rejected for distribution: Execution Policy blocks scripts on corporate machines, slow startup (~300-500ms), encoding/output quirks when called by an agent, harder to distribute as a single unit. |
| Rust | Comparable binary properties to Go. Rejected due to steeper learning curve and slower development speed for a project of this scope. No meaningful runtime advantage for an I/O-bound CLI. |

**Known risks:**
- Go module ecosystem is less mature than npm/NuGet for some corporate tooling. Mitigated by minimal dependency count (2 external packages total).

---

### CLI Framework: cobra (`github.com/spf13/cobra`)

**Rationale:**
- De-facto standard for Go CLI tools. Used by kubectl, gh (GitHub CLI), docker CLI, hugo.
- Native support for nested subcommands — fits the `tfscli <resource> <action>` pattern.
- Auto-generated help and usage output. AI agents and humans can call `tfscli --help`, `tfscli workitem --help`, `tfscli workitem get --help`.
- Pairs with `viper` for config file / env / flag merging, but viper is not required — cobra works standalone.

**Considered alternatives:**

| Alternative | Why rejected |
|---|---|
| `urfave/cli` | Capable, but less idiomatic for deep subcommand trees. Cobra's command registration model (one file per command) maps directly to the "one file per API domain" architecture goal. |
| No framework (bare `flag` package) | Feasible for MVP, but becomes painful when adding subcommands, per-command flags, and help generation. Not worth the initial savings. |

---

### HTML-to-Markdown: `github.com/JohannesKaufmann/html-to-markdown/v2`

**Rationale:**
- Mature, well-tested library (golden file tests, active maintenance, v2 released).
- Handles bold, italic, lists, tables, code blocks, links — all of which appear in TFS work item rich text fields.
- Extensible via plugins and custom rules. If TFS generates non-standard HTML in rich text fields, custom rules can handle edge cases without modifying core conversion logic.
- MIT licensed.

**Considered alternatives:**

| Alternative | Why rejected |
|---|---|
| `github.com/mrjoshuak/go-markdownify` | Port of Python `markdownify`. Less mature, no plugin system, zero downstream importers. |
| Manual regex stripping | Fragile, doesn't handle nested HTML structures, would need constant maintenance. |
| Returning raw HTML | Defeats the purpose — wastes tokens, AI agents parse HTML worse than markdown. |

---

### HTTP Client: Go standard library (`net/http`)

**Rationale:**
- All tfscli needs is `GET` and `POST` with JSON body, `Authorization` header, and TLS. Standard library covers this completely.
- No external dependency for core functionality.

**Considered alternatives:**

| Alternative | Why rejected |
|---|---|
| `github.com/go-resty/resty` | Convenient fluent API, but adds a dependency for features tfscli doesn't need (retries, middleware, file upload). |
| `github.com/hashicorp/go-retryablehttp` | Retry logic is useful, but for v1 simplicity — if the request fails, report the error. Retry can be added later without changing the interface. |

---

### Configuration: `encoding/json` (stdlib)

**Rationale:**
- Config file is `~/.tfscli/config.json` — a flat JSON object with 5 fields. Standard library `encoding/json` reads and writes it in a few lines.
- Config merging (file → env → flags) is handled by cobra's flag binding + a small custom resolver. No need for viper.

**Considered alternatives:**

| Alternative | Why rejected |
|---|---|
| `github.com/spf13/viper` | Full-featured config library (YAML, TOML, env, remote config). Massive dependency tree for a 5-field config. Adds complexity without proportional value. |
| TOML / YAML config format | JSON is sufficient for a flat config. Every language and tool reads JSON natively. No benefit from TOML/YAML for this use case. |

---

### Summary

| Component | Choice | External dependency? |
|---|---|---|
| Language | Go | — |
| CLI framework | cobra | Yes (`github.com/spf13/cobra`) |
| HTML → Markdown | html-to-markdown v2 | Yes (`github.com/JohannesKaufmann/html-to-markdown/v2`) |
| HTTP client | `net/http` | No (stdlib) |
| JSON parsing | `encoding/json` | No (stdlib) |
| Config format | JSON via `encoding/json` | No (stdlib) |

Total external dependencies: **2** (cobra, html-to-markdown). Minimal dependency footprint for a CLI tool.
