# Security Policy

## Supported Versions

We release patches for security vulnerabilities in the following versions:

| Version | Supported          |
| ------- | ------------------ |
| Latest  | :white_check_mark: |
| < Latest| :x:                |

We recommend always using the latest version of CLIPilot.

## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues.**

If you discover a security vulnerability in CLIPilot, please report it privately:

1. **Email**: Send details to the project maintainers through GitHub
2. **GitHub Security**: Use GitHub's private security advisory feature
3. **Include**:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fix (if any)

### What to Expect

- **Acknowledgment**: Within 48 hours of your report
- **Initial Assessment**: Within 5 business days
- **Status Updates**: We'll keep you informed of progress
- **Fix Timeline**: Critical issues will be prioritized
- **Credit**: Security researchers will be credited (unless they prefer anonymity)

## Security Best Practices

When using CLIPilot:

1. **Review Commands**: Always review commands before confirming execution
2. **Module Sources**: Only install modules from trusted sources
3. **Credentials**: Don't hardcode credentials in modules
4. **Registry**: Use strong passwords for registry authentication
5. **Updates**: Keep CLIPilot updated to the latest version
6. **Permissions**: Run with minimum required permissions

## Registry Security

For self-hosted registry instances:

- Change default admin password immediately
- Use HTTPS in production
- Implement rate limiting
- Validate all uploaded modules
- Regularly backup your database
- Keep server software updated

## Known Limitations

- Commands are executed with user's shell permissions
- YAML modules are parsed and executed - review before installation
- SQLite database is stored unencrypted (contains no sensitive data by default)
- Web registry sessions use HTTP-only cookies

## Security Updates

Security updates will be:
- Released as soon as possible
- Documented in release notes
- Announced through GitHub releases
- Tagged with "security" label

Thank you for helping keep CLIPilot and its users safe!
