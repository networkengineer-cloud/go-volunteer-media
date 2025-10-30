# Custom Agents

This directory contains custom agent configurations for specialized development tasks within the go-volunteer-media repository.

## What are Custom Agents?

Custom agents are specialized AI assistants configured with specific expertise, tools, and guidelines for particular domains or tasks. They provide focused, expert-level guidance and can perform complex operations within their area of specialization.

## Available Agents

### Security and Observability Expert
**File:** `security-observability-expert.agent.md`

**Purpose:** Specialized agent for security hardening and observability implementation

**Use When:**
- Implementing security features (authentication, authorization, encryption)
- Reviewing code for security vulnerabilities
- Setting up logging, metrics, and monitoring
- Configuring Docker and deployment security
- Implementing rate limiting and DDoS protection
- Adding health checks and readiness probes
- Auditing API security
- Reviewing database security configurations

**Key Expertise:**
- JWT authentication and authorization patterns
- Password security and account protection
- API security (CORS, rate limiting, input validation)
- Data protection and privacy
- Container security (Docker, Kubernetes)
- Structured logging and distributed tracing
- Metrics collection and alerting
- Security headers and HTTPS configuration

**Example Invocations:**
```
"Review the authentication handlers for security vulnerabilities"
"Implement structured logging with request IDs throughout the application"
"Add Prometheus metrics for API endpoint monitoring"
"Audit the Docker configuration for security best practices"
"Implement rate limiting for the login endpoint"
```

## How to Use Custom Agents

### With GitHub Copilot

1. **Explicit Invocation:**
   When you need specialized expertise, reference the agent directly:
   ```
   @security-observability-expert review the auth middleware for security issues
   ```

2. **Task Assignment:**
   Delegate specific tasks to the appropriate agent:
   ```
   I need to implement health checks. Please use the security-observability-expert agent.
   ```

3. **Code Review:**
   Request focused code reviews:
   ```
   Have the security expert review my new API endpoint for vulnerabilities.
   ```

### Agent Responsibilities

Custom agents are expected to:
- ✅ Provide expert guidance in their domain
- ✅ Identify issues and vulnerabilities
- ✅ Suggest specific, actionable improvements
- ✅ Explain the "why" behind recommendations
- ✅ Provide code examples and implementations
- ✅ Consider context and trade-offs
- ✅ Prioritize issues by severity

Custom agents should NOT:
- ❌ Work outside their area of expertise
- ❌ Make changes without explanation
- ❌ Ignore security or best practice concerns
- ❌ Provide generic or vague advice

## Creating New Custom Agents

### File Naming Convention
```
<agent-name>.agent.md
```

### Required Frontmatter
```yaml
---
name: 'Agent Display Name'
description: 'Brief description of agent purpose'
tools: ['list', 'of', 'relevant', 'tools']
mode: 'agent'
---
```

### Agent Structure

1. **Mission Statement**: Clear purpose and goals
2. **Expertise Areas**: Detailed breakdown of knowledge domains
3. **Implementation Guidelines**: Specific to this repository
4. **Best Practices**: Checklists and standards
5. **Communication Style**: How to provide feedback
6. **Tools & Techniques**: Relevant tools and methods
7. **Examples**: Demonstration of agent interactions

### Example Template

```markdown
---
name: 'Your Agent Name'
description: 'What this agent specializes in'
tools: ['relevant', 'tools']
mode: 'agent'
---

# Your Agent Name

You are a specialized agent focused on [domain].

## Core Mission
[What the agent does]

## Expertise Areas
[Detailed breakdown]

## Implementation Guidelines
[Repository-specific guidance]

## Best Practices
[Standards and checklists]
```

## Integration with Existing Files

### Relationship to Other .github Files

**Chatmodes** (`/.github/chatmodes/`)
- Provide general development modes and expert personas
- Broader scope than agents
- Example: `expert-react-frontend-engineer.chatmode.md`

**Prompts** (`/.github/prompts/`)
- Task-specific prompt templates
- Guide specific operations
- Example: `multi-stage-dockerfile.prompt.md`

**Instructions** (`/.github/instructions/`)
- Language and framework-specific rules
- Applied automatically based on file patterns
- Example: `go.instructions.md`, `reactjs.instructions.md`

**Agents** (`/.github/agents/`)
- **Specialized, domain-focused experts**
- **Most focused and expert-level guidance**
- **Invoked explicitly for complex tasks**
- This is where you are now!

### When to Use Each

| Type | Scope | Use Case |
|------|-------|----------|
| Instructions | Language/Framework | Auto-applied coding standards |
| Prompts | Specific Tasks | Template for common operations |
| Chatmodes | Development Role | Expert persona for consultation |
| **Agents** | **Domain Expert** | **Complex, specialized work** |

## Best Practices for Agent Usage

1. **Match Task to Agent**: Choose the agent whose expertise aligns with your task
2. **Provide Context**: Give the agent relevant information about your specific situation
3. **Be Specific**: Clear, specific questions get better responses
4. **Iterate**: Use agent feedback to refine your implementation
5. **Combine Agents**: Different agents can work on different aspects of the same feature
6. **Review Agent Work**: Always review and understand agent recommendations before applying

## Contributing

### Adding a New Agent

1. Create a new `.agent.md` file in this directory
2. Follow the naming convention: `<agent-name>.agent.md`
3. Include complete frontmatter
4. Provide comprehensive expertise areas
5. Include repository-specific guidelines
6. Add examples and use cases
7. Update this README with the new agent

### Updating Existing Agents

1. Keep expertise areas current with best practices
2. Add new patterns as they emerge in the codebase
3. Update repository-specific guidelines as the project evolves
4. Ensure examples remain relevant
5. Test agent effectiveness and refine as needed

## Examples from This Repository

### Security Review Example

**Task:** "Review the authentication system for security vulnerabilities"

**Agent Used:** Security and Observability Expert

**Agent Actions:**
1. Reviewed JWT implementation
2. Checked password hashing
3. Analyzed account lockout mechanism
4. Validated CORS configuration
5. Checked for SQL injection vulnerabilities
6. Reviewed error handling
7. Provided prioritized recommendations

### Observability Implementation Example

**Task:** "Add structured logging with request tracing"

**Agent Used:** Security and Observability Expert

**Agent Actions:**
1. Analyzed current logging approach
2. Recommended structured logging library
3. Designed request ID middleware
4. Implemented correlation ID propagation
5. Updated log statements with context
6. Added logging best practices documentation
7. Ensured no sensitive data in logs

## FAQ

**Q: Can agents work together?**
A: Yes! Different agents can work on different aspects of the same feature. For example, the Security Expert might review security while a frontend expert handles UI.

**Q: How do agents differ from chatmodes?**
A: Agents are more specialized and focused on specific domains. Chatmodes provide broader expert personas for general development guidance.

**Q: Can I create an agent for my specific need?**
A: Yes! Follow the template and contribution guidelines. Agents can be very specific to your project's needs.

**Q: Do agents replace instructions files?**
A: No, they complement each other. Instructions provide coding standards, agents provide expert guidance for complex tasks.

**Q: Are agents only for security and observability?**
A: No! Create agents for any specialized domain: performance optimization, accessibility, testing strategies, database design, etc.

## Support

For questions or issues with custom agents:
1. Review the agent documentation
2. Check examples in this README
3. Look at how existing agents are structured
4. Create an issue in the repository

---

**Remember:** Custom agents are powerful tools for maintaining high standards in specialized domains. Use them wisely and keep them updated as your codebase evolves.
