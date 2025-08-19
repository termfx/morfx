# Contributing Guidelines

Welcome to the morfx project! This directory contains all the information you need to contribute effectively.

## Contents

- [Code Standards](./CODE_STANDARDS.md) - Coding conventions and best practices
- [Pull Request Guidelines](./pull-request-guidelines.md) - How to submit effective PRs
- [Issue Templates](./issue-templates.md) - Reporting bugs and requesting features
- [Release Process](./release-process.md) - How releases are managed

## Quick Start for Contributors

1. **Fork and Clone**

   ```bash
   git clone https://github.com/termfx/morfx.git
   cd morfx
   ```

2. **Set Up Development Environment**

   ```bash
   go mod download
   make test
   ```

3. **Make Your Changes**

   - Follow the [Code Standards](./CODE_STANDARDS.md)
   - Write tests for new functionality
   - Update documentation as needed

4. **Test Your Changes**

   ```bash
   make fixtest    # Format and run full test suite
   make gate       # Run integration tests
   ```

5. **Submit a Pull Request**
   - Follow the [PR Guidelines](./pull-request-guidelines.md)
   - Ensure all CI checks pass
   - Respond to review feedback promptly

## Code of Conduct

This project follows the [Go Community Code of Conduct](https://golang.org/conduct).
By participating, you agree to uphold this code.

## Getting Help

- **Questions**: Open a GitHub Discussion
- **Bugs**: Create an Issue with the bug template
- **Features**: Create an Issue with the feature request template
- **Chat**: Join our community discussions

## Types of Contributions

- **Bug Fixes** - Help make morfx more reliable
- **Features** - Add new transformation capabilities
- **Documentation** - Improve guides and API docs
- **Performance** - Optimize critical paths
- **Language Support** - Add new language providers
- **Testing** - Improve test coverage and quality
