# Test Organization

This directory holds focused black-box tests for behavior that spans module boundaries or is clearer outside module-local test clutter.

Policy:
- Prefer package-external tests here (`<module>_test`) when behavior can be verified through exported APIs.
- Keep white-box tests inside `internal/modules/<module>` only when they need unexported helpers, unexported types, or package-private state.
- Preserve module-local tests for low-level private utilities, such as generated IDs/barcodes or other unexported helpers.

Current layout:
- `store/`: store logo replacement and safe logo deletion behavior.
- `product/`: product image replacement and safe product image deletion behavior.
