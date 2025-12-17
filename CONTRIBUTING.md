# Contributing to CLIPilot

Thank you for your interest in contributing to CLIPilot! We welcome contributions from the community and are grateful for any help you can provide.

## 🌟 Ways to Contribute

- **Report Bugs**: File issues for bugs you encounter
- **Suggest Features**: Share ideas for new features or improvements
- **Write Code**: Submit pull requests for bug fixes or new features
- **Improve Documentation**: Help make our docs clearer and more comprehensive
- **Create Modules**: Share YAML modules for common tasks (via PR or web registry)
- **Test**: Try CLIPilot on different platforms and report your experience

## 📦 Contributing Modules

There are two ways to contribute modules:

### 1. Via Pull Request (Built-in Modules)

Built-in modules are packaged with CLIPilot and work offline:

1. Fork the repository
2. Add your module YAML to `modules/` directory
3. Test it thoroughly
4. Submit a Pull Request
5. See [Module Guidelines](#module-guidelines) below

### 2. Via Web Registry (Community Modules)

Quick way to share experimental or personal modules:

1. Go to https://clipilot.themobileprof.com
2. Click "Login with GitHub"
3. Upload your module YAML
4. Module available immediately to all users

**When to use which:**
- **Built-in (PR)**: Stable, well-tested, broadly useful modules
- **Registry (Web)**: Experimental, personal, or niche-specific modules

## 🚀 Getting Started

### Development Setup

1. **Fork and Clone**
   ```bash
   git clone https://github.com/YOUR_USERNAME/clipilot.git
   cd clipilot
   ```

2. **Install Go 1.21+**
   - Download from [golang.org](https://golang.org/dl/)
   - Or use your package manager

3. **Build from Source**
   ```bash
   go build -o bin/clipilot ./cmd/clipilot
   go build -o bin/registry ./cmd/registry
   ```

4. **Run Tests**
   ```bash
   go test ./...
   ```

### Project Structure

```
clipilot/
├── cmd/              # Application entry points
│   ├── clipilot/     # Main CLI binary
│   └── registry/     # Web registry server
├── internal/         # Private application code
│   ├── db/           # Database layer
│   ├── engine/       # Flow execution engine
│   ├── intent/       # Intent detection
│   ├── modules/      # Module loader
│   └── ui/           # REPL interface
├── pkg/              # Public libraries
│   ├── config/       # Configuration
│   └── models/       # Data models
├── server/           # Registry web server
│   ├── auth/         # Authentication
│   ├── handlers/     # HTTP handlers
│   ├── static/       # CSS/JS assets
│   └── templates/    # HTML templates
├── modules/          # Default YAML modules
└── docs/             # Documentation
```

## 📝 Contribution Guidelines

### Reporting Issues

When filing an issue, please include:
- **Description**: Clear description of the issue
- **Steps to Reproduce**: Step-by-step instructions
- **Expected Behavior**: What you expected to happen
- **Actual Behavior**: What actually happened
- **Environment**: OS, Go version, CLIPilot version
- **Logs**: Relevant error messages or logs

### Submitting Pull Requests

1. **Create a Feature Branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make Your Changes**
   - Write clean, readable code
   - Follow Go conventions and best practices
   - Add comments for complex logic
   - Update documentation as needed

3. **Test Your Changes**
   ```bash
   go test ./...
   go build -o bin/clipilot ./cmd/clipilot
   ./bin/clipilot --help
   ```

4. **Commit Your Changes**
   ```bash
   git add .
   git commit -m "feat: add your feature description"
   ```
   
   Use conventional commits:
   - `feat:` - New feature
   - `fix:` - Bug fix
   - `docs:` - Documentation only
   - `style:` - Code style/formatting
   - `refactor:` - Code refactoring
   - `test:` - Adding tests
   - `chore:` - Maintenance tasks

5. **Push and Create PR**
   ```bash
   git push origin feature/your-feature-name
   ```
   Then open a pull request on GitHub

### Code Style

- **Go**: Follow standard Go conventions
  - Use `gofmt` to format code
  - Use meaningful variable names
  - Keep functions focused and small
  - Add godoc comments for exported functions

- **YAML Modules**: Follow the module specification in the registry

- **Commit Messages**: Use clear, descriptive commit messages

## 🎯 Creating YAML Modules

One of the easiest ways to contribute is by creating modules for common tasks:

1. **Write Your Module**
   ```yaml
   metadata:
     name: your_module_name
     version: "1.0.0"
     description: Brief description
     author: Your Name
     keywords:
       - keyword1
       - keyword2
   
   flows:
     main:
       steps:
         - type: instruction
           name: step1
           message: "Your instruction"
   ```

2. **Test Locally**
   ```bash
   clipilot --load=your_module.yaml
   clipilot "trigger your module with keywords"
   ```

3. **Share via Registry**
   - Visit your deployed registry
   - Login and upload your module
   - Or submit via pull request to `modules/` directory

## 🐛 Debugging

- **Enable Verbose Logging**
  ```bash
  export CLIPILOT_LOG_LEVEL=debug
  clipilot "your command"
  ```

- **Check Database**
  ```bash
  sqlite3 ~/.clipilot/clipilot.db
  .tables
  SELECT * FROM modules;
  ```

- **Run Registry in Debug Mode**
  ```bash
  PORT=8080 LOG_LEVEL=debug ./bin/registry
  ```

## 🤝 Community

- **Discussions**: Use GitHub Discussions for questions and ideas
- **Issues**: File bugs and feature requests on GitHub Issues
- **Pull Requests**: Submit code contributions via PRs
- **Be Respectful**: Follow our code of conduct (treat everyone with respect)

## 📚 Documentation

Help improve our documentation:
- Fix typos or unclear explanations
- Add examples and use cases
- Translate documentation
- Create tutorials or guides

## ✅ PR Checklist

Before submitting a pull request:

- [ ] Code builds successfully (`go build ./...`)
- [ ] Tests pass (`go test ./...`)
- [ ] Documentation updated if needed
- [ ] Commit messages are clear and descriptive
- [ ] Code follows Go conventions
- [ ] No sensitive information in commits
- [ ] PR description explains the changes

## 📄 License

By contributing to CLIPilot, you agree that your contributions will be licensed under the MIT License.

## 🙏 Thank You!

Your contributions make CLIPilot better for everyone. We appreciate your time and effort!

---

**Questions?** Open a GitHub Discussion or reach out through GitHub Issues.
