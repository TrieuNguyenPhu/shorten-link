# Backend (AWS SAM + Lambda + API Gateway + DynamoDB)

Serverless backend that creates and resolves short URLs.

## API Endpoints

- `POST /api/generate-short-url`
  - Input body: `{"url":"https://example.com"}`
  - Output body: `{"short_url_code":"<generated-id>"}`
- `GET /link/{short_url}`
  - Looks up `short_url` in DynamoDB
  - Returns HTTP redirect to original URL when found

## Tech Stack

- AWS Lambda (Python 3.12 runtime)
- Amazon API Gateway
- Amazon DynamoDB
- AWS SAM

## Prerequisites

- Python 3.12.x
- AWS CLI configured with credentials
- AWS SAM CLI 1.160.1

Optional checks:

```bash
python --version
aws --version
sam --version
aws sts get-caller-identity
```

## Install Dependencies

```bash
pip install -r requirements.txt
```

## Build and Deploy

```bash
sam build
sam deploy --guided
```

After deployment, check CloudFormation outputs for:

- `ApiBaseUrl`
- `GenerateShortUrlEndpoint`

## Local Development

Run API locally:

```bash
sam local start-api
```

Test locally:

```bash
curl -X POST http://127.0.0.1:3000/api/generate-short-url \
  -H "Content-Type: application/json" \
  -d "{\"url\":\"https://example.com\"}"
```

## Tests

Install test dependencies:

```bash
pip install -r tests/requirements.txt
```

Run all tests:

```bash
pytest tests
```

Notes:

- Integration test `tests/integration/test_api_gateway.py` needs `API_BASE_URL`.
- Without `API_BASE_URL`, integration test is skipped.

## Cleanup

Delete deployed resources:

```bash
sam delete --stack-name shorten-link-backend
```
