# Contribution Guidelines

When enhancing this project:

1. **Maintain Simplicity**: Don't add features that significantly increase complexity
2. **Performance First**: Profile before and after changes; maintain lightweight footprint
3. **Backward Compatibility**: Preserve existing CLI flags and behavior
4. **Document Decisions**: Update `doc/HISTORY.md` with rationale for significant changes
5. **Test Coverage**: Add tests for new functionality
6. **Error Handling**: Follow Go conventions; return errors, don't panic
7. **Logging**: Use structured logging; respect --verbose flag
