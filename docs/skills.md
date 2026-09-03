# AI Skills

TOPF ships a set of **skills** — prompt-level workflows that guide an AI coding agent (Claude, opencode, etc.) through complex, multi-step tasks involving TOPF and Talos configuration. They live in the [`skills/`](https://github.com/postfinance/topf/tree/main/skills) directory of the repository and can be loaded by any agent that supports the skill format.

## What is a skill?

A skill is a Markdown file (`SKILL.md`) plus optional reference material that an AI agent reads before performing a task. It tells the agent *when* to use it, *what* steps to follow, and *where* to find authoritative references. Skills do not execute code — they shape how the agent reasons about a problem.

## Available skills

| Skill | Purpose |
| ----- | ------- |
| [`migrate-talhelper-to-topf`](https://github.com/postfinance/topf/tree/main/skills/migrate-talhelper-to-topf) | Migrate a cluster from talhelper (now archived) to TOPF: convert `talconfig.yaml` to `topf.yaml` + patch files, rewrite JSON patches as strategic merge, move envsubst/talenv values into SOPS-encrypted `data` + Go templates. |
| [`talos-v114-migration`](https://github.com/postfinance/topf/tree/main/skills/talos-v114-migration) | Migrate Talos config patches from the v1.13 (or earlier) single-document v1alpha1 format to the v1.14 multi-document config format (dedicated kinds like `KubeletConfig`, `VolumeConfig`, `KubeNodeConfig`, …). |

## How to use a skill

How you load a skill depends on your agent. With opencode, skills under `skills/` are discovered automatically and the agent decides when to invoke them based on their `description`. With Claude Code, place the `skills/` directory under `~/.claude/skills/` (or the project's `.claude/skills/`) so the agent can load it via the skill tool.

You can also just read the `SKILL.md` file directly and follow the workflow yourself — the steps are written for a human to follow too.

---

!!! warning "AI-generated content"

    The skill files under `skills/` are **entirely AI-generated**, unlike the TOPF
    source code which is hand-written and reviewed. They were produced by an AI
    assistant and have **not** been line-by-line verified against the Talos and
    TOPF reference documentation.

    They may contain **errors**, outdated field mappings, or instructions that do
    not match the current behaviour of Talos or TOPF. Treat them as a starting
    point, not as authoritative reference. Always cross-check against:

    - the [TOPF configuration reference](configuration.md)
    - the [TOPF configuration model](configuration-model.md)
    - the [Talos v1.14 configuration reference](https://docs.siderolabs.com/talos/v1.14/reference/configuration/)
    - the [upstream migration guide](migration-from-talhelper.md)

    If you spot an error in a skill, please open an
    [issue](https://github.com/postfinance/topf/issues) or a pull request.
