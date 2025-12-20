# LinkFlow AI Documentation

Welcome to the LinkFlow AI documentation. This guide covers everything you need to know about building, deploying, and extending the platform.

## Documentation Structure

```
docs/
├── README.md                    # This file
├── getting-started/             # Setup and quick start guides
│   ├── installation.md
│   ├── quickstart.md
│   ├── local-deployment.md
│   └── configuration.md
├── architecture/                # System architecture docs
│   ├── overview.md
│   ├── domain-driven-design.md
│   └── services.md
├── api/                         # API documentation
│   ├── overview.md
│   ├── authentication.md
│   ├── workflows.md
│   ├── executions.md
│   └── webhooks.md
├── guides/                      # How-to guides
│   ├── creating-workflows.md
│   ├── adding-nodes.md
│   ├── adding-integrations.md
│   └── deployment.md
├── reference/                   # Reference documentation
│   ├── environment-variables.md
│   ├── node-types.md
│   └── expression-functions.md
└── development/                 # Developer documentation
    ├── contributing.md
    ├── testing.md
    └── coding-standards.md
```

## Quick Links

- **[Setup Guide](getting-started/SETUP.md)** - Start here! Step-by-step setup
- [Installation Guide](getting-started/installation.md)
- [Quick Start](getting-started/quickstart.md)
- [Local Deployment Reference](getting-started/local-deployment.md)
- [API Reference](api/overview.md)
- [Architecture Overview](architecture/overview.md)
- [Node Types Reference](reference/node-types.md)

## What is LinkFlow AI?

LinkFlow AI is a production-ready workflow automation platform similar to n8n or Zapier. It allows you to:

- **Create Workflows**: Build automated workflows with a visual editor
- **Connect Services**: Integrate with 13+ third-party services
- **Execute at Scale**: Run workflows with a distributed execution engine
- **Monitor & Debug**: Track executions with detailed logging and debugging tools

## Key Features

| Feature | Description |
|---------|-------------|
| Workflow Engine | DAG-based execution with parallel processing |
| 15+ Node Types | Triggers, actions, conditions, loops |
| 13+ Integrations | Slack, GitHub, Google Sheets, PostgreSQL, etc. |
| Expression System | 50+ built-in functions for data transformation |
| Multi-tenancy | Workspace-based isolation |
| Billing | Stripe integration with usage tracking |
| Real-time | WebSocket support for live updates |

## Tech Stack

- **Backend**: Go 1.25+
- **Database**: PostgreSQL
- **Cache**: Redis (optional)
- **Message Queue**: Kafka (optional)
- **Observability**: Prometheus, Jaeger

## Getting Help

- Check the [Troubleshooting Guide](guides/troubleshooting.md)
- Review [Common Issues](guides/common-issues.md)
- Open an issue on GitHub

## License

LinkFlow AI is proprietary software. See LICENSE file for details.
