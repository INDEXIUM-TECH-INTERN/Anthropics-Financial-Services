# Domain Glossary: Financial AI Agent

## Architecture Concepts

### Message
A neutral, provider-agnostic container for conversation data. It encapsulates the role (system, user, assistant, tool), text content, and structured data for tool calls or responses.

### Provider
A deep module interface that abstracts a Large Language Model (LLM) vendor. It acts as an adapter, translating neutral `Message` objects and `ToolSchema` definitions into vendor-specific API calls.

### ContextWindow
The central repository for conversation history. It maintains the "source of truth" for a dialogue as a sequence of neutral `Message` objects, ensuring locality of state.

### ToolSchema
A neutral definition of a function or capability that an agent can invoke. It describes the name, purpose, and required parameters in a format that adapters can translate for specific LLMs.

### Orchestrator
The flow-control module responsible for the main loop of interaction: receiving input, updating memory, calling providers, and dispatching tool results.
