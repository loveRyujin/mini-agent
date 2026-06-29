# Mini-Agent

A terminal-based coding agent that reads, writes, and executes within a local codebase on the developer's machine.

## Language

**Coding Agent**:
An autonomous assistant whose primary job is to help the developer modify and understand a local codebase.
_Avoid_: chatbot, general assistant, conversational AI

**Workspace**:
The local directory tree the agent may inspect and act upon during a session, fixed to the process working directory at launch.
_Avoid_: project root, cwd, working folder

**Approval Gate**:
A pause where the developer must explicitly allow a tool action before it runs, presented as a modal overlay above the Transcript.
_Avoid_: confirmation prompt, permission dialog, inline y/n

**Shell Execution**:
Running an operating-system command within or against the workspace; always subject to an Approval Gate.
_Avoid_: bash, terminal command, exec

**File Mutation**:
Creating, modifying, or deleting files within the workspace; permitted without an Approval Gate.
_Avoid_: write, edit, patch

**Session**:
The in-memory conversation between the developer and the agent for one launch of the application; discarded when the process exits.
_Avoid_: chat history, conversation log, thread

**Inference Backend**:
The language-model service the agent delegates reasoning to during a session.
_Avoid_: LLM provider, API, Ollama

**Transcript**:
The scrollable chronological record of every event in a Session—developer messages, agent replies, tool invocations, and approval prompts—in a single column.
_Avoid_: chat log, message list, history pane

**Turn**:
One cycle beginning with a developer message and ending when the agent finishes responding, including any tool loops within it.
_Avoid_: round, exchange, iteration

**Tool**:
A named capability the agent may invoke to inspect or act on the Workspace.
_Avoid_: function, plugin, skill

**Built-in Tool**:
A Tool shipped with the application and available without additional configuration.
_Avoid_: native tool, default tool, core tool

**Workspace Search**:
Finding files or text within the Workspace by pattern.
_Avoid_: grep, ripgrep, find

**Slash Command**:
A developer-initiated action outside the Turn loop, invoked by typing `/` followed by a name in the input box.
_Avoid_: CLI command, meta command, shortcut

**System Prompt**:
The standing instructions that define how the agent behaves across a Session; overridable at launch while a coding-agent default is provided.
_Avoid_: system message, persona, instructions

**Reasoning**:
The model's chain-of-thought text emitted before or alongside its reply; shown inline in the Transcript when the Inference Backend provides it.
_Avoid_: thinking, internal monologue, scratchpad
