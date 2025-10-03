# Contributing to GitHub Copilot Metrics Exporter

Thank you for your interest in contributing to this project! Here are some guidelines to help you get started.

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/YOUR_USERNAME/github-copilot-metrics-exporter.git`
3. Create a new branch: `git checkout -b feature/your-feature-name`
4. Make your changes
5. Test your changes: `make test`
6. Commit your changes: `git commit -am 'Add some feature'`
7. Push to the branch: `git push origin feature/your-feature-name`
8. Submit a pull request

## Development Setup

### Prerequisites

- Go 1.21 or later
- Make (optional, but recommended)

### Building

```bash
make build
```

Or without make:

```bash
go build -o github-copilot-metrics-exporter .
```

### Running Tests

```bash
make test
```

Or without make:

```bash
go test -v ./...
```

### Code Style

- Run `go fmt ./...` before committing
- Run `go vet ./...` to check for common mistakes
- Follow Go best practices and idioms

## Pull Request Guidelines

- Write clear, descriptive commit messages
- Update documentation if you're changing functionality
- Add tests for new features
- Ensure all tests pass before submitting
- Keep pull requests focused on a single change

## Reporting Issues

When reporting issues, please include:

- Go version
- Operating system
- Steps to reproduce the issue
- Expected behavior
- Actual behavior
- Any relevant logs or error messages

## License

By contributing to this project, you agree that your contributions will be licensed under the Apache License 2.0.
