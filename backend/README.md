# Backend (AWS SAM + Lambda + API Gateway + DynamoDB)

This service exposes two endpoints:

- `POST /api/generate-short-url`: creates a short code from a full URL.
- `GET /link/{short_url}`: resolves the code and redirects to the original URL.

## Prerequisites

- Python 3.12.x
- AWS CLI configured
- AWS SAM CLI 1.160.1

## Install

```bash
pip install -r requirements.txt
```

## Build and Deploy

```bash
sam build
sam deploy --guided
```

## Local Run

```bash
sam local start-api
```

## Tests

Install test dependencies:

```bash
pip install -r tests/requirements.txt
```

Run tests:

```bash
pytest tests
```
