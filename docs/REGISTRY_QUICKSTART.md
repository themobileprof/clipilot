# CLIPilot Registry - Quick Start Guide

This guide will help you run the CLIPilot module registry and upload your first module.

## Starting the Registry

### 1. Build the registry (if not done)

```bash
cd /home/samuel/sites/clipilot
go build -o bin/registry ./cmd/registry
```

### 2. Start the server

**Option A: Using environment variables (recommended)**

```bash
# Create .env file
cat > .env << 'EOF'
ADMIN_PASSWORD=demo123
PORT=8080
EOF

# Start server (loads .env automatically)
./bin/registry
```

**Option B: Using command-line flags**

```bash
./bin/registry --password=demo123
```

**Option C: Using environment variables directly**

```bash
ADMIN_PASSWORD=demo123 ./bin/registry
```

The server will start on http://localhost:8080 with:
- Admin username: `admin` (default)
- Admin password: `demo123`

See [Environment Variables Configuration](REGISTRY_ENV_CONFIG.md) for more options.

### 3. Access the web interface

Open your browser to http://localhost:8080

## Uploading a Module

### 1. Login

Visit http://localhost:8080/login
- Username: `admin`
- Password: `demo123` (or whatever you set)

### 2. Go to Upload page

After login, you'll be redirected to http://localhost:8080/upload

### 3. Prepare your YAML module

Example Python venv setup module:

```yaml
name: python_venv_setup
version: "1.0.0"
description: Create and configure a Python virtual environment
tags:
  - python
  - venv
  - virtualenv
metadata:
  author: Your Name
  license: MIT
flows:
  main:
    start: check_python
    steps:
      check_python:
        type: action
        message: "Checking Python installation..."
        command: "python3 --version"
        next: create_venv
      
      create_venv:
        type: action
        message: "Creating virtual environment..."
        command: "python3 -m venv venv"
        next: activate_instructions
      
      activate_instructions:
        type: instruction
        message: |
          ✓ Virtual environment created!
          
          To activate:
            source venv/bin/activate  # Linux/Mac
            venv\\Scripts\\activate    # Windows
        next: install_packages
      
      install_packages:
        type: action
        message: "Installing common packages..."
        command: |
          source venv/bin/activate
          pip install --upgrade pip
          pip install requests pytest black
          pip list
        next: success
      
      success:
        type: terminal
        message: "✓ Python virtual environment ready!"
```

### 4. Upload the file

- Click "Choose File" and select your YAML
- Click "Upload Module"
- You'll see a success message

### 5. Verify the upload

- Go to http://localhost:8080/modules to see your module listed
- Go to http://localhost:8080/my-modules to see your uploaded modules

## Using Modules from CLI

### 1. Configure CLI to use registry

```bash
# The CLI defaults to localhost:8080
# To change:
# (This will be automatic once we add settings UI)
```

### 2. Install a module

```bash
clipilot modules install 1
```

Where `1` is the module ID from the registry.

### 3. Use the module

```bash
clipilot "setup python environment"
```

Or directly:

```bash
clipilot run python_venv_setup
```

## Using ChatGPT to Generate Modules

The upload page includes a prompt template. Here's how to use it:

### 1. Copy the prompt from /upload page

Or use this template:

```
Create a CLIPilot module YAML file for [YOUR TASK].

The YAML must follow this structure:
- name: lowercase_underscore format
- version: semantic version (e.g., "1.0.0")
- description: Brief description
- tags: Array of keywords
- metadata: author, license
- flows: main flow with steps
- step types: instruction (message only), action (command + message), branch (conditional), terminal (end)

Requirements:
1. Use clear, descriptive step names
2. Include helpful messages for users
3. Add validation where commands might fail
4. Use terminal steps for success/failure endings
5. Choose relevant tags/keywords

Output ONLY the YAML code, no explanations.
```

### 2. Ask ChatGPT

Example:

```
Create a CLIPilot module YAML file for setting up PostgreSQL on Ubuntu with database creation and user configuration.
```

### 3. Review and upload

- ChatGPT will generate the YAML
- Review it for correctness
- Save to a .yaml file
- Upload via the web interface

## API Usage

The registry provides a JSON API for programmatic access:

### List all modules

```bash
curl http://localhost:8080/api/modules
```

Response:
```json
[
  {
    "id": 1,
    "name": "python_venv_setup",
    "version": "1.0.0",
    "description": "Create and configure a Python virtual environment",
    "author": "Your Name",
    "downloads": 0
  }
]
```

### Download a module

```bash
curl -o module.yaml http://localhost:8080/modules/1
```

## Next Steps

- Explore the [full registry documentation](REGISTRY.md)
- Learn about [module development](../README.md#creating-custom-modules)
- Share your modules with the community!

## Troubleshooting

### Port already in use

```bash
./bin/registry --port=8081 --password=demo123
```

### Templates not found

Make sure you're running from the project root:

```bash
cd /home/samuel/sites/clipilot
./bin/registry --password=demo123
```

### Database errors

Delete and recreate:

```bash
rm -rf ./data
./bin/registry --password=demo123
```

## Security Notes

⚠️ **For production use:**

1. Use a strong password
2. Enable HTTPS (use reverse proxy like nginx)
3. Set up proper authentication
4. Regular database backups
5. Consider user registration system

This demo setup is for development/testing only!
