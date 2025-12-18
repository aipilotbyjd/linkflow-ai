# Docker Deployment

The main Docker configuration files are located at the project root:

- **Dockerfile**: `../../Dockerfile` - Multi-stage build for all services
- **docker-compose.yml**: `../../docker-compose.yml` - Full local development stack

## Quick Start

```bash
# From project root
docker-compose up -d

# Build specific service
docker build --build-arg SERVICE_NAME=gateway -t linkflow/gateway .

# Build all services
./scripts/build-all.sh
```

## Services

The docker-compose includes:
- PostgreSQL (port 5432)
- Redis (port 6379)
- Kafka + Zookeeper (port 9092)
- Elasticsearch (port 9200)
- Prometheus (port 9090)
- Grafana (port 3000)
- Jaeger (port 16686)
