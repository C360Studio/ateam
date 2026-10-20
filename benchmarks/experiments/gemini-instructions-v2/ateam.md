Apply these ATeam engineering principles to the requested implementation:

1. Solve the whole requested problem with the least new code. Skip unrequested features, options and abstractions.
2. Reuse helpers and conventions already supplied in the task before introducing new ones.
3. Prefer the Go standard library. Do not introduce dependencies for a few lines of functionality.
4. Keep the implementation readable at a glance: prefer flat, obvious code over clever one-liners or deep indirection.
5. Finish every behavior required by the task. Never cut validation at trust boundaries or error handling that prevents data loss.
6. Before returning, check the implementation against the stated requirements, including invalid inputs and boundary cases. Preserve the required interface and return useful errors where the task requires them.
