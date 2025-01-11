# Contributing Guide

Thank you for considering contributing to the GreatSQL Operator project! This document will guide you through the contribution process.

## Code of Conduct

This project follows the [Contributor Covenant](https://www.contributor-covenant.org/version/2/0/code_of_conduct/). By participating, you are expected to uphold this code.

## How to Contribute

### Reporting Bugs

- Before submitting a bug, please search existing Issues to avoid duplicates
- Use the Issue template to submit bug reports
- Include detailed problem description and steps to reproduce
- Provide a minimal reproduction example if possible

### Suggesting Features

- Search existing Issues before submitting feature requests
- Use the feature request template
- Explain why the feature would be valuable to the project

### Contributing Code

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some feature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Create a Pull Request

### Development Environment Setup

1. Ensure you have the following dependencies:
   - Go 1.23.0+
   - Docker 17.03+
   - kubectl 1.11.3+
   - Access to a Kubernetes v1.22.0+ cluster

2. Clone the repository:
```bash
git clone https://github.com/greatsql-sigs/greatsql-operator
cd greatsql-operator
```

3. Run tests to ensure everything is working:
```bash
make test
```

## Coding Standards

### Go Code Standards

- Follow the [Go Code Standards](https://golang.org/doc/effective_go)
- Use `gofmt` to format code
- Add appropriate comments and documentation
- Ensure code passes `golint` and `go vet` checks

### Commit Message Convention

Commit messages should follow this format:
```
<type>: <description>

[optional body]

[optional footer]
```

Types include:
- feat: New feature
- fix: Bug fix
- docs: Documentation changes
- style: Code style changes
- refactor: Code refactoring
- test: Testing related changes
- chore: Build process or auxiliary tool changes

### Testing Requirements

- Add unit tests for new features
- Ensure all tests pass
- Maintain test coverage

## Pull Request Process

1. Update your branch with the latest main branch changes
2. Ensure all tests pass
3. Update relevant documentation
4. Fill out the PR template
5. Wait for code review
6. Make requested changes
7. Wait for merge

## Documentation Contributions

- Ensure documentation is clear and understandable
- Update README.md and other relevant docs
- Add examples and usage instructions
- Fix spelling and grammar errors

## Getting Help

- Check the project [README.md](README.md)
- Search existing Issues
- Ask questions in Issues
- Join community discussions

## License

By submitting code to this project, you agree that your contributions will be licensed under the project's Apache 2.0 License. 