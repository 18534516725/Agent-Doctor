# Aider integration (C-level)

Run Aider through Agent Doctor's argv-preserving wrapper:

```bash
agent-doctor run -- aider --model <your-model>
```

The wrapper preserves the child process exit status and arguments without shell
interpretation. It captures no transcript, prompt, source file, or command
content. Signals that Aider does not expose remain `unavailable`; they are not
estimated from private application state.
