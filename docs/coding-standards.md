# Coding Standards

- **Vertical Slicing**: The project uses vertical slicing for its architecture. Each feature (e.g., GitHub webhook processing, command handling) has its own folder and is implemented in a modular way.
- **Error Handling**: Errors are handled by returning informative messages and using standard Go error types.
- **Testing**: Unit tests are written for each handler and parser to ensure correctness.
