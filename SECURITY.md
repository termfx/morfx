# Security Policy

## Table of Contents

- [Reporting Security Vulnerabilities](#reporting-security-vulnerabilities)
- [Supported Versions](#supported-versions)
- [Security Features](#security-features)
- [Security Considerations](#security-considerations)
- [Best Practices](#best-practices)
- [Security Architecture](#security-architecture)
- [Incident Response](#incident-response)
- [Security Updates](#security-updates)

## Reporting Security Vulnerabilities

We take the security of morfx seriously. If you discover a security vulnerability, please follow these guidelines:

### 🚨 **DO NOT** create a public GitHub issue for security vulnerabilities

Instead, please report security vulnerabilities responsibly through one of these channels:

### Primary Reporting Method

**Email**: [security@morfx.dev](mailto:security@morfx.dev)

Please include the following information in your report:

- **Summary**: A brief description of the vulnerability
- **Impact**: The potential impact of the vulnerability
- **Reproduction**: Step-by-step instructions to reproduce the issue
- **Environment**: Version of morfx and operating system details
- **PoC**: Proof-of-concept code if applicable (optional)
- **Suggested Fix**: If you have ideas for a fix (optional)

### Alternative Reporting Methods

If email is not available, you can also:

1. **GitHub Security Advisory**: Use the "Report a vulnerability" feature in the Security tab of the repository
2. **Direct Contact**: Contact maintainers privately through GitHub

### What to Expect

1. **Acknowledgment**: We'll acknowledge receipt within 24 hours
2. **Initial Response**: We'll provide an initial response within 72 hours
3. **Investigation**: We'll investigate and provide updates every 7 days
4. **Resolution**: We'll work to resolve the issue as quickly as possible
5. **Disclosure**: We'll coordinate disclosure timing with you

### Responsible Disclosure

We follow responsible disclosure principles:

- **Embargo Period**: We request at least 90 days from initial report to public disclosure
- **Coordinated Disclosure**: We'll work with you to determine appropriate disclosure timing
- **Credit**: We'll credit you in security advisories (if you wish)
- **Bug Bounty**: While we don't have a formal bug bounty program, we deeply appreciate security research

## Supported Versions

We provide security updates for the following versions of morfx:

| Version | Supported  | Support Level                      |
| ------- | ---------- | ---------------------------------- |
| 1.x.x   | ✅ Yes     | Full support (features + security) |
| 0.x.x   | ⚠️ Limited | Security fixes only                |
| < 0.1   | ❌ No      | End of life                        |

### Security Update Policy

- **Current Major Version (1.x.x)**: Receives all security updates immediately
- **Previous Major Version**: Receives critical security updates for 6 months after new major release
- **Pre-1.0 Versions**: Limited security updates only for high/critical severity issues

## Security Features

morfx includes several security features designed to protect your code transformations:

### 1. Data Encryption

```bash
# Enable encryption mode
export MORFX_ENCRYPTION_MODE=blob
export MORFX_MASTER_KEY=your-256-bit-key-here

# morfx will encrypt sensitive data at rest
morfx transform --input sensitive-code/ --output results/
```

**Features**:

- AES-256 encryption for sensitive data
- Configurable encryption keys
- Optional encryption for database storage
- Secure key derivation functions

### 2. Input Validation

- **Query Parsing**: Strict DSL query validation prevents injection attacks
- **File Path Validation**: Prevents directory traversal attacks
- **Content Sanitization**: Input sanitization for all user-provided content
- **Size Limits**: Configurable limits on file size and processing time

### 3. Secure File Handling

- **Temporary Files**: Secure creation and cleanup of temporary files
- **File Permissions**: Appropriate file permissions for created files
- **Path Resolution**: Safe path resolution to prevent traversal attacks
- **Atomic Operations**: Atomic file operations to prevent race conditions

### 4. Database Security

- **SQLite Security**: Secure SQLite configuration with prepared statements
- **Encryption at Rest**: Optional database encryption
- **Access Control**: File system permissions for database files
- **Transaction Integrity**: ACID properties maintained

### 5. Memory Safety

- **Go Memory Safety**: Leverages Go's memory safety features
- **Buffer Overflow Protection**: Prevents buffer overflows in parsing
- **Resource Limits**: Configurable resource limits to prevent DoS
- **Garbage Collection**: Automatic memory management

## Security Considerations

### Using morfx Securely

#### 1. Sensitive Code Handling

When working with sensitive code:

```bash
# Enable encryption for sensitive transformations
export MORFX_ENCRYPTION_MODE=blob
export MORFX_MASTER_KEY=$(openssl rand -hex 32)

# Use temporary directories with restricted permissions
mkdir -m 700 /tmp/morfx-work
morfx transform --workdir /tmp/morfx-work --input ./sensitive/
```

#### 2. Configuration Security

- **Environment Variables**: Use environment variables for sensitive configuration
- **File Permissions**: Restrict access to configuration files (chmod 600)
- **Key Management**: Use proper key management systems for encryption keys
- **Audit Configuration**: Regularly audit configuration for security issues

#### 3. Network Security

morfx is designed to work offline, but when using in networked environments:

- **No Network Access Required**: morfx doesn't require internet connectivity
- **Firewall Configuration**: If running in container, configure network policies
- **VPN/Secure Networks**: Use VPNs when transferring sensitive code

#### 4. Logging and Monitoring

- **Sensitive Data in Logs**: morfx avoids logging sensitive data
- **Audit Logs**: Enable audit logging for compliance requirements
- **Log Rotation**: Implement proper log rotation and retention policies
- **Monitoring**: Monitor for unusual activity or errors

### Common Attack Vectors and Mitigations

#### 1. Code Injection Attacks

**Risk**: Malicious code in query strings or input files
**Mitigation**:

- Strict DSL parsing with whitelist approach
- Input validation and sanitization
- Tree-sitter parsing prevents most injection attacks

#### 2. Path Traversal Attacks

**Risk**: Accessing files outside intended directories
**Mitigation**:

- Path validation and canonicalization
- Chroot-like restrictions in processing
- Whitelist of allowed file extensions

#### 3. Denial of Service (DoS)

**Risk**: Resource exhaustion through large files or complex queries
**Mitigation**:

- Configurable timeouts and resource limits
- Memory usage monitoring
- Graceful error handling

#### 4. Information Disclosure

**Risk**: Leaking sensitive information through errors or logs
**Mitigation**:

- Careful error message design
- Sensitive data exclusion from logs
- Secure temporary file handling

## Best Practices

### For Users

1. **Keep morfx Updated**

   ```bash
   # Check for updates regularly
   morfx version --check-updates

   # Update to latest version
   go install github.com/termfx/morfx/cmd/morfx@latest
   ```

2. **Secure Configuration**

   ```bash
   # Use environment variables for secrets
   export MORFX_MASTER_KEY=$(vault kv get -field=key secret/morfx/encryption)

   # Restrict config file permissions
   chmod 600 ~/.morfx/config.yml
   ```

3. **Input Validation**

   ```bash
   # Validate inputs before processing
   morfx validate --query "your-query" --input ./code/
   ```

4. **Audit and Monitoring**

   ```bash
   # Enable audit logging
   export MORFX_AUDIT_LOG=/var/log/morfx-audit.log

   # Monitor for errors
   tail -f /var/log/morfx.log | grep ERROR
   ```

### For Developers

1. **Secure Development Practices**

   - Follow the [Code Standards](docs/contributing/CODE_STANDARDS.md)
   - Use static analysis tools (gosec, golangci-lint)
   - Implement proper error handling
   - Validate all inputs

2. **Testing Security**

   ```bash
   # Run security tests
   make test-security

   # Check for vulnerabilities
   govulncheck ./...

   # Static security analysis
   gosec ./...
   ```

3. **Dependency Management**

   ```bash
   # Audit dependencies
   go mod audit

   # Update dependencies
   go get -u ./...
   go mod tidy
   ```

### For Operators

1. **Deployment Security**

   - Use minimal container images
   - Run with least privilege principles
   - Enable security monitoring
   - Implement proper access controls

2. **Monitoring and Alerting**
   - Monitor for security events
   - Set up alerts for unusual activity
   - Regular security audits
   - Incident response procedures

## Security Architecture

### Threat Model

morfx's threat model considers:

1. **Malicious Input**: Untrusted code or query input
2. **Privilege Escalation**: Attempting to access unauthorized files/data
3. **Data Exfiltration**: Unauthorized access to sensitive code
4. **Denial of Service**: Resource exhaustion attacks
5. **Supply Chain**: Compromised dependencies

### Defense in Depth

morfx employs multiple layers of security:

1. **Input Layer**: Validation, sanitization, size limits
2. **Processing Layer**: Safe parsing, resource limits, isolation
3. **Storage Layer**: Encryption, access controls, audit logging
4. **Output Layer**: Safe file creation, permission management
5. **Infrastructure Layer**: Container security, network policies

### Security Controls

| Control Type   | Implementation                    | Coverage |
| -------------- | --------------------------------- | -------- |
| **Preventive** | Input validation, access controls | High     |
| **Detective**  | Logging, monitoring, auditing     | Medium   |
| **Corrective** | Error handling, failsafe defaults | High     |
| **Recovery**   | Backup, rollback capabilities     | Medium   |

## Incident Response

### Incident Classification

- **Critical**: Remote code execution, data breach, privilege escalation
- **High**: Local code execution, information disclosure
- **Medium**: Denial of service, data corruption
- **Low**: Information leakage, minor security features

### Response Timeline

- **Critical**: Immediate response (< 4 hours)
- **High**: Same day response (< 24 hours)
- **Medium**: Weekly response (< 7 days)
- **Low**: Monthly response (< 30 days)

### Response Process

1. **Confirmation**: Verify and reproduce the vulnerability
2. **Assessment**: Determine impact and severity
3. **Containment**: Develop and test fixes
4. **Communication**: Notify affected users
5. **Resolution**: Release security updates
6. **Post-Incident**: Review and improve security measures

## Security Updates

### Update Channels

1. **GitHub Releases**: Primary channel for security updates
2. \*\*
