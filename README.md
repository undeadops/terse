# terse

Simple URL Shortener backed by DynamoDB

## Features

- Fast URL shortening with 16-character alphanumeric keys
- DynamoDB backend for high scalability
- Access counting for analytics
- RESTful API for management
- Docker Compose setup for easy local development
- Multi-architecture container images (amd64/arm64)

## Quick Start

### Using Docker Compose

```bash
# Start the services
docker-compose up -d

# The API will be available at http://localhost:8080
```

### Building from Source

```bash
# Build the binary
go build -o terse ./cmd/terse

# Run with environment variables
export AWS_REGION=us-east-1
export AWS_ACCESS_KEY_ID=dummy1
export AWS_SECRET_ACCESS_KEY=dummy1
export DYNAMODB_TABLE=terse
export DEBUG=true
./terse
```

## API Usage

### Create a Short URL

Create a new shortened URL:

```bash
curl -X POST http://localhost:5000/manage/ \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://example.com/very/long/url/path"
  }'
```

**Response:**

```json
{
  "short_url": "localhost:5000/g/abc123XYZ456789"
}
```

### Access a Short URL

Navigate to the short URL to be redirected:

```bash
# Browser or curl
curl -L http://localhost:5000/g/abc123XYZ456789
```

This will redirect (HTTP 302) to the original URL and increment the access counter.

### List All URLs

Get a list of all shortened URLs:

```bash
curl http://localhost:5000/manage/
```

**Response:**

```json
{
  "urls": [
    {
      "key": "abc123XYZ456789",
      "url": "https://example.com/very/long/url/path",
      "redirect_count": 5
    }
  ]
}
```

### Delete a Short URL

Remove a shortened URL:

```bash
curl -X DELETE http://localhost:5000/manage/abc123XYZ456789
```

**Response:**

```json
{
  "message": "Redirect deleted successfully"
}
```

## Configuration

Configure the service using CLI flags or environment variables:

| CLI Flag | Environment Variable | Default | Description |
|----------|---------------------|---------|-------------|
| `--region` | `AWS_REGION` | `us-east-1` | AWS region for DynamoDB |
| `--table` | `DYNAMODB_TABLE` | `terse` | DynamoDB table name |
| `--debug` | `DEBUG` | `false` | Enable debug logging |
| `--redirect-url` | `REDIRECT_URL` | `` | Base URL for shortened links |
| `--ttl-days` | `TTL_DAYS` | `30` | Default Redirect Time to Live |

**Example:**

```bash
./terse --region us-west-2 --table my-urls --debug
```

## Development

### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/api/...
```

### Local DynamoDB

The Docker Compose setup includes DynamoDB Local for development:

```yaml
# DynamoDB is available at:
# http://localhost:8000

# AWS CLI example
aws dynamodb list-tables \
  --endpoint-url http://localhost:8000 \
  --region us-east-1
```

## Deployment

Container images are automatically published to GitHub Container Registry on releases:

```bash
# Pull the latest image
docker pull ghcr.io/undeadops/terse:latest

# Run the container
docker run -p 8080:8080 \
  -e AWS_REGION=us-east-1 \
  -e AWS_ACCESS_KEY_ID=your-key \
  -e AWS_SECRET_ACCESS_KEY=your-secret \
  -e DYNAMODB_TABLE=terse \
  ghcr.io/undeadops/terse:latest
```

## License

See [LICENSE](LICENSE) file for details.
