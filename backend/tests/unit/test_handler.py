import json
import sys
from unittest.mock import MagicMock, patch
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[2]))

from generate_short_url import app as generate_app
from get_url import app as redirect_app


@patch("generate_short_url.app.boto3.resource")
def test_generate_short_url_success(mock_resource):
    table = MagicMock()
    mock_resource.return_value.Table.return_value = table

    event = {"body": json.dumps({"url": "https://example.com"})}
    response = generate_app.lambda_handler(event, None)

    assert response["statusCode"] == 200
    payload = json.loads(response["body"])
    assert "short_url_code" in payload
    assert len(payload["short_url_code"]) == 15
    table.put_item.assert_called_once()


def test_generate_short_url_missing_body():
    event = {"body": None}
    response = generate_app.lambda_handler(event, None)

    assert response["statusCode"] == 400
    assert "Request body is empty" in response["body"]


@patch("get_url.app.boto3.resource")
def test_get_url_success(mock_resource):
    table = MagicMock()
    table.get_item.return_value = {"Item": {"original_url": "https://example.com"}}
    mock_resource.return_value.Table.return_value = table

    event = {"pathParameters": {"short_url": "abc123"}}
    response = redirect_app.lambda_handler(event, None)

    assert response["statusCode"] == 308
    assert response["headers"]["Location"] == "https://example.com"


@patch("get_url.app.boto3.resource")
def test_get_url_not_found(mock_resource):
    table = MagicMock()
    table.get_item.return_value = {}
    mock_resource.return_value.Table.return_value = table

    event = {"pathParameters": {"short_url": "not-found"}}
    response = redirect_app.lambda_handler(event, None)

    assert response["statusCode"] == 404
    assert "URL not found" in response["body"]
