# Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability in Vervet, please report it responsibly via [GitHub Security Advisories](https://github.com/blacktau/vervet/security/advisories/new) rather than opening a public issue.

Please include:

- A description of the vulnerability
- Steps to reproduce
- Potential impact

## Security Considerations

- Connection URIs are stored in the OS keyring and are never written to config files on disk
- Vervet connects directly to MongoDB servers — ensure your network and MongoDB authentication are properly configured
